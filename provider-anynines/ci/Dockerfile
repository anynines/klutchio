FROM docker
COPY --from=docker/buildx-bin:latest /buildx /usr/libexec/docker/cli-plugins/docker-buildx

RUN apk update && apk add bash make go git nodejs npm curl yq
