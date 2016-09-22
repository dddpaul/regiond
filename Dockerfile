FROM dddpaul/oracleclient:12.1.0.2.0-1
MAINTAINER Pavel Derendyaev <pderendyaev@gmail.com>

ENV DEBIAN_FRONTEND noninteractive
ENV INITRD No
ENV LANG en_US.UTF-8
ENV GOVERSION 1.7.1
ENV GOROOT /opt/go
ENV GOPATH /root/.go
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

RUN apt-get update \
    && apt-get install -y wget git pkg-config \
    && cd /opt \
    && wget https://storage.googleapis.com/golang/go${GOVERSION}.linux-amd64.tar.gz \
    && tar zxf go${GOVERSION}.linux-amd64.tar.gz \
    && rm go${GOVERSION}.linux-amd64.tar.gz \
    && ln -s /opt/go/bin/go /usr/bin/ \
    && mkdir $GOPATH \
    && apt-get clean && rm -rf /var/cache/apt/* /var/lib/apt/lists/* /tmp/* /var/tmp/*

ADD root /
ADD oci8.pc /usr/lib/pkgconfig/
RUN go get github.com/mattn/go-oci8

ENTRYPOINT ["/bin/regiond"]
CMD ["proxy", "-p", "80"]
EXPOSE 80
