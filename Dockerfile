FROM docker.io/golang:1.23-alpine AS builder

COPY . /work
WORKDIR /work
RUN go build

FROM docker.io/alpine:latest

COPY --from=builder /work/inbound /inbound

ENTRYPOINT ["/inbound"]