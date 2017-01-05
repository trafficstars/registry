FROM busybox

ENV DOCKER_HOST=unix:///var/run/docker.sock \
    DOCKER_API_VERSION=1.24 \
    SERVICE_NAME=docker-visor \
    CHECK_HTTP=http://{{address}}/api/v1/check \
    HOST_IP=${HOST} \
    REGISTRY_DSN="http://${HOST}:8500?dc=dc1&refresh_interval=5" 

COPY .build/ts-supervisor /ts-supervisor
CMD ["/ts-supervisor"]
