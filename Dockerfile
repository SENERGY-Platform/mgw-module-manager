FROM golang:1.20 AS builder

ARG VERSION=dev

COPY . /go/src/app
WORKDIR /go/src/app

RUN CGO_ENABLED=0 GOOS=linux go build -o manager -ldflags="-X 'main.version=$VERSION'" main.go

FROM alpine:latest

RUN mkdir -p /opt/module-manager
WORKDIR /opt/module-manager
RUN mkdir include
COPY --from=builder /go/src/app/manager manager
COPY --from=builder /go/src/app/include include

ENTRYPOINT ["./manager"]
