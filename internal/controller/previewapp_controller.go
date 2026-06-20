/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	previewappv1alpha1 "github.com/danieljcheung/previewapp/api/v1alpha1"
)

const httpPortName = "http"

// PreviewAppReconciler reconciles a PreviewApp object
type PreviewAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=previewapp.danieljcheung.com,resources=previewapps,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=previewapp.danieljcheung.com,resources=previewapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=previewapp.danieljcheung.com,resources=previewapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *PreviewAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var app previewappv1alpha1.PreviewApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	expiresAt := expiresAtForPreviewApp(&app)
	if previewAppExpired(expiresAt) {
		log.Info("Deleting expired PreviewApp", "name", app.Name, "namespace", app.Namespace, "expiresAt", expiresAt)
		if err := r.Delete(ctx, &app); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, nil
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = labelsForPreviewApp(&app)
		deployment.Spec = deploymentSpecForPreviewApp(&app)
		return controllerutil.SetControllerReference(&app, deployment, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled Deployment", "name", deployment.Name, "namespace", deployment.Namespace, "operation", operation)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	operation, err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = labelsForPreviewApp(&app)
		service.Spec.Selector = labelsForPreviewApp(&app)
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Ports = []corev1.ServicePort{{
			Name:       httpPortName,
			Port:       80,
			TargetPort: intstr.FromInt32(app.Spec.AppPort),
			Protocol:   corev1.ProtocolTCP,
		}}
		return controllerutil.SetControllerReference(&app, service, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled Service", "name", service.Name, "namespace", service.Namespace, "operation", operation)

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}

	operation, err = controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = labelsForPreviewApp(&app)
		ingress.Spec = ingressSpecForPreviewApp(&app)
		return controllerutil.SetControllerReference(&app, ingress, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciled Ingress", "name", ingress.Name, "namespace", ingress.Namespace, "operation", operation)

	nextStatus := statusForPreviewApp(&app, deployment, service, ingress, expiresAt)
	if previewAppStatusChanged(app.Status, nextStatus) {
		app.Status = nextStatus
		if err := r.Status().Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Updated PreviewApp status", "name", app.Name, "namespace", app.Namespace, "phase", app.Status.Phase, "url", app.Status.URL)

	return ctrl.Result{RequeueAfter: time.Until(expiresAt.Time)}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PreviewAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&previewappv1alpha1.PreviewApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Named("previewapp").
		Complete(r)
}

func deploymentSpecForPreviewApp(app *previewappv1alpha1.PreviewApp) appsv1.DeploymentSpec {
	labels := labelsForPreviewApp(app)

	return appsv1.DeploymentSpec{
		Replicas: ptr.To[int32](1),
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				AutomountServiceAccountToken: ptr.To(false),
				SecurityContext: &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(true),
					RunAsUser:    ptr.To[int64](1000),
					RunAsGroup:   ptr.To[int64](1000),
					FSGroup:      ptr.To[int64](1000),
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
				Containers: []corev1.Container{{
					Name:  "app",
					Image: app.Spec.Image,
					Ports: []corev1.ContainerPort{{
						Name:          httpPortName,
						ContainerPort: app.Spec.AppPort,
					}},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("25m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				}},
			},
		},
	}
}

func ingressSpecForPreviewApp(app *previewappv1alpha1.PreviewApp) networkingv1.IngressSpec {
	return networkingv1.IngressSpec{
		IngressClassName: ptr.To("nginx"),
		Rules: []networkingv1.IngressRule{{
			Host: hostForPreviewApp(app),
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path:     "/",
						PathType: ptr.To(networkingv1.PathTypePrefix),
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: app.Name,
								Port: networkingv1.ServiceBackendPort{
									Number: 80,
								},
							},
						},
					}},
				},
			},
		}},
	}
}

func statusForPreviewApp(app *previewappv1alpha1.PreviewApp, deployment *appsv1.Deployment, service *corev1.Service, ingress *networkingv1.Ingress, expiresAt metav1.Time) previewappv1alpha1.PreviewAppStatus {
	deploymentReady := deployment.Status.AvailableReplicas > 0
	serviceReady := serviceMatchesPreviewApp(app, service)
	ingressReady := ingressMatchesPreviewApp(app, ingress)
	previewReady := deploymentReady && serviceReady && ingressReady

	status := previewappv1alpha1.PreviewAppStatus{
		Phase:              phaseForPreviewApp(previewReady),
		ExpiresAt:          &expiresAt,
		ObservedGeneration: app.Generation,
	}
	if previewReady {
		status.URL = publicURLForPreviewApp(app)
	}

	deploymentCondition := metav1.Condition{
		Type:               "DeploymentReady",
		Status:             metav1.ConditionFalse,
		Reason:             "WaitingForAvailableReplicas",
		Message:            "Deployment has no available replicas yet.",
		ObservedGeneration: app.Generation,
	}
	if deploymentReady {
		deploymentCondition.Status = metav1.ConditionTrue
		deploymentCondition.Reason = "AvailableReplicas"
		deploymentCondition.Message = "Deployment has available replicas."
	}
	meta.SetStatusCondition(&status.Conditions, deploymentCondition)

	serviceCondition := metav1.Condition{
		Type:               "ServiceReady",
		Status:             metav1.ConditionFalse,
		Reason:             "ServiceMismatch",
		Message:            "Service does not match the desired preview routing spec.",
		ObservedGeneration: app.Generation,
	}
	if serviceReady {
		serviceCondition.Status = metav1.ConditionTrue
		serviceCondition.Reason = "ServiceReconciled"
		serviceCondition.Message = "Service routes traffic to preview pods."
	}
	meta.SetStatusCondition(&status.Conditions, serviceCondition)

	ingressCondition := metav1.Condition{
		Type:               "IngressReady",
		Status:             metav1.ConditionFalse,
		Reason:             "IngressMismatch",
		Message:            "Ingress does not match the desired PopInvites route.",
		ObservedGeneration: app.Generation,
	}
	if ingressReady {
		ingressCondition.Status = metav1.ConditionTrue
		ingressCondition.Reason = "IngressConfigured"
		ingressCondition.Message = "Ingress is configured for " + publicURLForPreviewApp(app) + "."
	}
	meta.SetStatusCondition(&status.Conditions, ingressCondition)

	readyCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reasonForNotReady(deploymentReady, serviceReady, ingressReady),
		Message:            messageForNotReady(deploymentReady, serviceReady, ingressReady),
		ObservedGeneration: app.Generation,
	}
	if previewReady {
		readyCondition.Status = metav1.ConditionTrue
		readyCondition.Reason = "PreviewReady"
		readyCondition.Message = "Preview app is serving traffic."
	}
	meta.SetStatusCondition(&status.Conditions, readyCondition)

	return status
}

func serviceMatchesPreviewApp(app *previewappv1alpha1.PreviewApp, service *corev1.Service) bool {
	if service.Spec.Type != corev1.ServiceTypeClusterIP ||
		len(service.Spec.Ports) != 1 ||
		service.Spec.Ports[0].Name != httpPortName ||
		service.Spec.Ports[0].Port != 80 ||
		service.Spec.Ports[0].TargetPort.IntVal != app.Spec.AppPort ||
		service.Spec.Ports[0].Protocol != corev1.ProtocolTCP {
		return false
	}

	labels := labelsForPreviewApp(app)
	for key, value := range labels {
		if service.Spec.Selector[key] != value {
			return false
		}
	}

	return true
}

func ingressMatchesPreviewApp(app *previewappv1alpha1.PreviewApp, ingress *networkingv1.Ingress) bool {
	if ingress.Spec.IngressClassName == nil ||
		*ingress.Spec.IngressClassName != "nginx" ||
		len(ingress.Spec.Rules) != 1 ||
		ingress.Spec.Rules[0].Host != hostForPreviewApp(app) ||
		ingress.Spec.Rules[0].HTTP == nil ||
		len(ingress.Spec.Rules[0].HTTP.Paths) != 1 {
		return false
	}

	path := ingress.Spec.Rules[0].HTTP.Paths[0]
	return path.Path == "/" &&
		path.PathType != nil &&
		*path.PathType == networkingv1.PathTypePrefix &&
		path.Backend.Service != nil &&
		path.Backend.Service.Name == app.Name &&
		path.Backend.Service.Port.Number == 80
}

func previewAppStatusChanged(current, next previewappv1alpha1.PreviewAppStatus) bool {
	if current.Phase != next.Phase ||
		current.URL != next.URL ||
		!previewAppTimesEqual(current.ExpiresAt, next.ExpiresAt) ||
		current.ObservedGeneration != next.ObservedGeneration ||
		len(current.Conditions) != len(next.Conditions) {
		return true
	}

	for _, nextCondition := range next.Conditions {
		currentCondition := meta.FindStatusCondition(current.Conditions, nextCondition.Type)
		if currentCondition == nil ||
			currentCondition.Status != nextCondition.Status ||
			currentCondition.Reason != nextCondition.Reason ||
			currentCondition.Message != nextCondition.Message ||
			currentCondition.ObservedGeneration != nextCondition.ObservedGeneration {
			return true
		}
	}

	return false
}

func expiresAtForPreviewApp(app *previewappv1alpha1.PreviewApp) metav1.Time {
	return metav1.NewTime(app.CreationTimestamp.Add(time.Duration(app.Spec.TTLSeconds) * time.Second))
}

func previewAppExpired(expiresAt metav1.Time) bool {
	return !time.Now().Before(expiresAt.Time)
}

func previewAppTimesEqual(current *metav1.Time, next *metav1.Time) bool {
	if current == nil || next == nil {
		return current == next
	}

	return current.Equal(next)
}

func phaseForPreviewApp(ready bool) string {
	if ready {
		return "Ready"
	}

	return "Reconciling"
}

func reasonForNotReady(deploymentReady bool, serviceReady bool, ingressReady bool) string {
	if !deploymentReady {
		return "WaitingForAvailableReplicas"
	}
	if !serviceReady {
		return "ServiceNotReady"
	}
	if !ingressReady {
		return "IngressNotReady"
	}

	return "PreviewReconciling"
}

func messageForNotReady(deploymentReady bool, serviceReady bool, ingressReady bool) string {
	if !deploymentReady {
		return "Preview app is waiting for an available pod."
	}
	if !serviceReady {
		return "Preview app is waiting for a matching Service."
	}
	if !ingressReady {
		return "Preview app is waiting for a matching Ingress."
	}

	return "Preview app is reconciling."
}

func hostForPreviewApp(app *previewappv1alpha1.PreviewApp) string {
	return app.Spec.Route.Host + ".popinvites.com"
}

func publicURLForPreviewApp(app *previewappv1alpha1.PreviewApp) string {
	return "https://" + hostForPreviewApp(app)
}

func labelsForPreviewApp(app *previewappv1alpha1.PreviewApp) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "previewapp",
		"app.kubernetes.io/managed-by": "previewapp-controller",
		"app.kubernetes.io/instance":   app.Name,
	}
}
