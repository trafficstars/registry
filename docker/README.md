```
docker run -id \
    --name supervisor \
    --memory 100MB \
    --restart always \
    -p :8000 \
    -e HOST_IP=192.168.20.48 \
    -e REGISTRY_DSN="http://192.168.20.48:8500?dc=dc1&refresh_interval=5" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    trafficstars-supervisor
```