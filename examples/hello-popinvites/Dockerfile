FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY --chown=1000:1000 public/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
