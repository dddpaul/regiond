FROM dddpaul/ora:12.1.0.2.0-1
MAINTAINER Pavel Derendyaev <pderendyaev@smile-net.ru>

# Paste from https://github.com/docker-library/golang/blob/9f666dc2f4f51df564613f787d28b3a2353243e0/1.6/alpine/Dockerfile
RUN apk add --no-cache ca-certificates

ENV GOLANG_VERSION 1.6.3
ENV GOLANG_SRC_URL https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz
ENV GOLANG_SRC_SHA256 6326aeed5f86cf18f16d6dc831405614f855e2d416a91fd3fdc334f772345b00

# https://golang.org/issue/14851
COPY no-pic.patch /

RUN set -ex \
	&& apk add --no-cache --virtual .build-deps \
		bash \
		gcc \
		musl-dev \
		openssl \
		go \
	\
	&& export GOROOT_BOOTSTRAP="$(go env GOROOT)" \
	\
	&& wget -q "$GOLANG_SRC_URL" -O golang.tar.gz \
	&& echo "$GOLANG_SRC_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz \
	&& cd /usr/local/go/src \
	&& patch -p2 -i /no-pic.patch \
	&& ./make.bash \
	\
	&& rm -rf /*.patch
	# && apk del .build-deps # Commented by Pavel: we will need it later

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
WORKDIR $GOPATH

COPY go-wrapper /usr/local/bin/
# /Paste

ADD root /
RUN apk add --update git
# RUN go get github.com/mattn/go-oci8
ENTRYPOINT ["/bin/fedpa"]
CMD ["proxy", "-p", "80"]
EXPOSE 80
