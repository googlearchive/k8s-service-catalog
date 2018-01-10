FROM debian:stretch

RUN apt update && apt install -y golang-go git ca-certificates

WORKDIR /go/src/plori/catalog-oauth
ENV GOPATH /go
COPY watcher/*.go ./watcher/
COPY auth/*.go ./auth/
COPY main.go .

RUN go get -v ./...
RUN mkdir bin
ENTRYPOINT ["go", "build", "-o", "bin/catalog-oauth"]
