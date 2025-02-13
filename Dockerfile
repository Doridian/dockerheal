FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . /app

ENV CGO_ENABLED=0

RUN go mod download && \
    go build -o /dockerheal .

FROM scratch
COPY LICENSE /LICENSE

COPY --from=builder --chown=0:0 --chmod=755 /dockerheal /dockerheal

ENTRYPOINT ["/dockerheal"]
