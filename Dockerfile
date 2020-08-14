FROM alpine:latest

COPY nats-test .

ENTRYPOINT ["/nats-test"]