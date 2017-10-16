# PaaS Usage Events Collector

## Testing

Locally you can use containers with [Docker for Mac](https://docs.docker.com/docker-for-mac/) or [Docker for Linux](https://docs.docker.com/engine/installation/linux/ubuntu/):

```
docker run -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5

# Clean up after
docker rm -f postgres
```
