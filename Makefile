.PHONY: all build release

IMAGE=smile/fedpa-tools
DOCKER_REGISTRY=docker.infoline.ru:5000
VERSION=$(shell cat VERSION)

all: build

build:
	@go test
	@mkdir -p root/bin
	@CGO_ENABLED=0 go build -o root/bin/fedpa
	@docker build --tag=${IMAGE} .

release: build
	@docker build --tag=${IMAGE}:${VERSION} .

deploy: release
	@docker tag ${IMAGE} ${DOCKER_REGISTRY}/${IMAGE}
	@docker tag ${IMAGE} ${DOCKER_REGISTRY}/${IMAGE}:${VERSION}
	@docker push ${DOCKER_REGISTRY}/${IMAGE}
	@docker push ${DOCKER_REGISTRY}/${IMAGE}:${VERSION}
