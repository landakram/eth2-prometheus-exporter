FROM golang:latest as builder
MAINTAINER Mark Hudnall <me@markhudnall.com>
WORKDIR /app

ENV GO111MODULE on
ENV CGO_ENABLED 0
ENV GOOS linux

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -a -o eth2-prometheus-exporter cmd/eth2-prometheus-exporter/main.go

FROM alpine:latest
MAINTAINER Mark Hudnall <me@markhudnall.com>
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/eth2-prometheus-exporter .

EXPOSE 8080
ENTRYPOINT ["./eth2-prometheus-exporter"]
