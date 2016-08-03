FROM alpine:3.3
MAINTAINER Pavel Derendyaev <pderendyaev@smile-net.ru>
ADD root /
ENTRYPOINT ["/bin/fedpa"]
CMD ["proxy", "-p", "80"]
EXPOSE 80
