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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	previewappv1alpha1 "github.com/danieljcheung/previewapp/api/v1alpha1"
)

var _ = Describe("PreviewApp Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		previewapp := &previewappv1alpha1.PreviewApp{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind PreviewApp")
			err := k8sClient.Get(ctx, typeNamespacedName, previewapp)
			if err != nil && errors.IsNotFound(err) {
				resource := &previewappv1alpha1.PreviewApp{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: previewappv1alpha1.PreviewAppSpec{
						Image:      "ghcr.io/danieljcheung/whisper@sha256:c7a67bf56c3c868e41a196d39893778a0005463eb417d4e1ef95f1f4114350b6",
						AppPort:    3000,
						TTLSeconds: 3600,
						Route: previewappv1alpha1.PreviewAppRouteSpec{
							Host: "test-resource",
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &previewappv1alpha1.PreviewApp{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance PreviewApp")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &PreviewAppReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			Expect(result.RequeueAfter).To(BeNumerically("<=", time.Hour))
			deployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, deployment)).To(Succeed())
			Expect(deployment.OwnerReferences).To(HaveLen(1))
			Expect(deployment.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(deployment.Spec.Template.Spec.AutomountServiceAccountToken).NotTo(BeNil())
			Expect(*deployment.Spec.Template.Spec.AutomountServiceAccountToken).To(BeFalse())

			podSecurity := deployment.Spec.Template.Spec.SecurityContext
			Expect(podSecurity).NotTo(BeNil())
			Expect(podSecurity.RunAsNonRoot).NotTo(BeNil())
			Expect(*podSecurity.RunAsNonRoot).To(BeTrue())
			Expect(podSecurity.RunAsUser).NotTo(BeNil())
			Expect(*podSecurity.RunAsUser).To(Equal(int64(1000)))
			Expect(podSecurity.RunAsGroup).NotTo(BeNil())
			Expect(*podSecurity.RunAsGroup).To(Equal(int64(1000)))
			Expect(podSecurity.FSGroup).NotTo(BeNil())
			Expect(*podSecurity.FSGroup).To(Equal(int64(1000)))
			Expect(podSecurity.SeccompProfile).NotTo(BeNil())
			Expect(podSecurity.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault))

			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Image).To(Equal("ghcr.io/danieljcheung/whisper@sha256:c7a67bf56c3c868e41a196d39893778a0005463eb417d4e1ef95f1f4114350b6"))
			Expect(container.Ports[0].ContainerPort).To(Equal(int32(3000)))
			Expect(container.SecurityContext).NotTo(BeNil())
			Expect(container.SecurityContext.AllowPrivilegeEscalation).NotTo(BeNil())
			Expect(*container.SecurityContext.AllowPrivilegeEscalation).To(BeFalse())
			Expect(container.SecurityContext.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(container.Resources.Requests).To(HaveKey(corev1.ResourceCPU))
			Expect(container.Resources.Requests).To(HaveKey(corev1.ResourceMemory))
			Expect(container.Resources.Limits).To(HaveKey(corev1.ResourceCPU))
			Expect(container.Resources.Limits).To(HaveKey(corev1.ResourceMemory))

			service := &corev1.Service{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, service)).To(Succeed())
			Expect(service.OwnerReferences).To(HaveLen(1))
			Expect(service.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(service.Spec.Selector).To(Equal(deployment.Spec.Template.Labels))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Name).To(Equal("http"))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(80)))
			Expect(service.Spec.Ports[0].TargetPort.IntVal).To(Equal(int32(3000)))
			Expect(service.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))

			ingress := &networkingv1.Ingress{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, ingress)).To(Succeed())
			Expect(ingress.OwnerReferences).To(HaveLen(1))
			Expect(ingress.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(ingress.Spec.IngressClassName).NotTo(BeNil())
			Expect(*ingress.Spec.IngressClassName).To(Equal("nginx"))
			Expect(ingress.Spec.Rules).To(HaveLen(1))
			Expect(ingress.Spec.Rules[0].Host).To(Equal("test-resource.popinvites.com"))
			Expect(ingress.Spec.Rules[0].HTTP).NotTo(BeNil())
			Expect(ingress.Spec.Rules[0].HTTP.Paths).To(HaveLen(1))

			path := ingress.Spec.Rules[0].HTTP.Paths[0]
			Expect(path.Path).To(Equal("/"))
			Expect(path.PathType).NotTo(BeNil())
			Expect(*path.PathType).To(Equal(networkingv1.PathTypePrefix))
			Expect(path.Backend.Service).NotTo(BeNil())
			Expect(path.Backend.Service.Name).To(Equal(resourceName))
			Expect(path.Backend.Service.Port.Number).To(Equal(int32(80)))

			statusPreviewApp := &previewappv1alpha1.PreviewApp{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, statusPreviewApp)).To(Succeed())
			Expect(statusPreviewApp.Status.Phase).To(Equal("Reconciling"))
			Expect(statusPreviewApp.Status.URL).To(BeEmpty())
			Expect(statusPreviewApp.Status.ObservedGeneration).To(Equal(statusPreviewApp.Generation))
			Expect(statusPreviewApp.Status.ExpiresAt).NotTo(BeNil())
			Expect(statusPreviewApp.Status.ExpiresAt.Time).To(BeTemporally("~", statusPreviewApp.CreationTimestamp.Add(time.Hour), time.Second))
			deploymentReady := meta.FindStatusCondition(statusPreviewApp.Status.Conditions, "DeploymentReady")
			Expect(deploymentReady).NotTo(BeNil())
			Expect(deploymentReady.Status).To(Equal(metav1.ConditionFalse))
			Expect(deploymentReady.Reason).To(Equal("WaitingForAvailableReplicas"))
			Expect(deploymentReady.ObservedGeneration).To(Equal(statusPreviewApp.Generation))

			serviceReady := meta.FindStatusCondition(statusPreviewApp.Status.Conditions, "ServiceReady")
			Expect(serviceReady).NotTo(BeNil())
			Expect(serviceReady.Status).To(Equal(metav1.ConditionTrue))
			Expect(serviceReady.Reason).To(Equal("ServiceReconciled"))

			ingressReady := meta.FindStatusCondition(statusPreviewApp.Status.Conditions, "IngressReady")
			Expect(ingressReady).NotTo(BeNil())
			Expect(ingressReady.Status).To(Equal(metav1.ConditionTrue))
			Expect(ingressReady.Reason).To(Equal("IngressConfigured"))

			ready := meta.FindStatusCondition(statusPreviewApp.Status.Conditions, "Ready")
			Expect(ready).NotTo(BeNil())
			Expect(ready.Status).To(Equal(metav1.ConditionFalse))
			Expect(ready.Reason).To(Equal("WaitingForAvailableReplicas"))

			By("marking the Deployment available and reconciling status again")
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			deployment.Status.AvailableReplicas = 1
			Expect(k8sClient.Status().Update(ctx, deployment)).To(Succeed())

			result, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			Expect(result.RequeueAfter).To(BeNumerically("<=", time.Hour))

			Expect(k8sClient.Get(ctx, typeNamespacedName, statusPreviewApp)).To(Succeed())
			Expect(statusPreviewApp.Status.Phase).To(Equal("Ready"))
			Expect(statusPreviewApp.Status.URL).To(Equal("https://test-resource.popinvites.com"))
			Expect(statusPreviewApp.Status.ObservedGeneration).To(Equal(statusPreviewApp.Generation))
			deploymentReady = meta.FindStatusCondition(statusPreviewApp.Status.Conditions, "DeploymentReady")
			Expect(deploymentReady).NotTo(BeNil())
			Expect(deploymentReady.Status).To(Equal(metav1.ConditionTrue))
			Expect(deploymentReady.Reason).To(Equal("AvailableReplicas"))
			Expect(deploymentReady.ObservedGeneration).To(Equal(statusPreviewApp.Generation))

			ready = meta.FindStatusCondition(statusPreviewApp.Status.Conditions, "Ready")
			Expect(ready).NotTo(BeNil())
			Expect(ready.Status).To(Equal(metav1.ConditionTrue))
			Expect(ready.Reason).To(Equal("PreviewReady"))
			Expect(ready.ObservedGeneration).To(Equal(statusPreviewApp.Generation))
		})
	})

	Context("When calculating expiration", func() {
		It("computes expiresAt from creationTimestamp and ttlSeconds", func() {
			createdAt := metav1.NewTime(time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC))
			app := &previewappv1alpha1.PreviewApp{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: createdAt,
				},
				Spec: previewappv1alpha1.PreviewAppSpec{
					TTLSeconds: 3600,
				},
			}

			expiresAt := expiresAtForPreviewApp(app)
			Expect(expiresAt.Time).To(Equal(createdAt.Add(time.Hour)))
		})

		It("detects expired timestamps", func() {
			expiredAt := metav1.NewTime(time.Now().Add(-time.Second))
			futureAt := metav1.NewTime(time.Now().Add(time.Hour))

			Expect(previewAppExpired(expiredAt)).To(BeTrue())
			Expect(previewAppExpired(futureAt)).To(BeFalse())
		})
	})
})
