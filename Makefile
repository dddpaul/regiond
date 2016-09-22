.PHONY: all build release

IMAGE=dddpaul/regiond
VERSION=$(shell cat VERSION)

all: build

build:
	@go test
	@mkdir -p root/bin
	@go build -o root/bin/regiond
	@docker build --tag=${IMAGE} .

debug:
	@docker run -it --entrypoint=sh ${IMAGE}

release: build
	@docker build --tag=${IMAGE}:${VERSION} .

deploy: release
	@docker push ${IMAGE}
	@docker push ${IMAGE}:${VERSION}
