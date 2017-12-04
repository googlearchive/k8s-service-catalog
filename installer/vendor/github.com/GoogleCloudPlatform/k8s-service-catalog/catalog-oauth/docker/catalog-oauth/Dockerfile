FROM debian:stretch
# This image assumes that the binary you want to run is already in ./bin

RUN apt update && apt install -y ca-certificates

COPY ./bin/catalog-oauth /bin
ENTRYPOINT ["/bin/catalog-oauth"]
CMD ["-v=0", "-logtostderr=true"]
