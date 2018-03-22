FROM golang:1.10-alpine3.7
RUN apk add --no-cache git

ADD . /go/src/github.com/GoogleCloudPlatform/k8s-service-catalog/catalog-oauth
RUN go get -v github.com/GoogleCloudPlatform/k8s-service-catalog/catalog-oauth
RUN go install github.com/GoogleCloudPlatform/k8s-service-catalog/catalog-oauth


FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=0 /go/bin/catalog-oauth .
ENTRYPOINT ["/catalog-oauth"]
CMD ["-v=0", "-logtostderr=true"]
