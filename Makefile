.PHONY: all build release

IMAGE=smile/fedpa-proxy
DOCKER_REGISTRY=docker.infoline.ru:5000
VERSION=$(shell cat VERSION)

all: build

build:
	@mkdir -p root/bin
	@CGO_ENABLED=0 go build -o root/bin/server server.go  
	@CGO_ENABLED=0 go build -o root/bin/proxy proxy.go
	@docker build --tag=${IMAGE} .

release: build
	@docker build --tag=${IMAGE}:${VERSION} .

deploy: release
	@docker tag ${IMAGE} ${DOCKER_REGISTRY}/${IMAGE}
	@docker tag ${IMAGE} ${DOCKER_REGISTRY}/${IMAGE}:${VERSION}
	@docker push ${DOCKER_REGISTRY}/${IMAGE}
	@docker push ${DOCKER_REGISTRY}/${IMAGE}:${VERSION}
