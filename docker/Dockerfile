FROM busybox

ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV DOCKER_API_VERSION=1.24
ENV SERVICE_NAME=trafficstars-supervisor
ENV CHECK_HTTP=http://{{address}}/api/v1/check

COPY .build/ts-supervisor /ts-supervisor
CMD ["/ts-supervisor"]
