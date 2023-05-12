FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY . /app

ENV CGO_ENABLED=0

RUN go mod download && \
    go build -o /dockerheal .

FROM scratch
COPY --from=builder --chown=0:0 --chmod=755 /dockerheal /dockerheal
ENTRYPOINT ["/dockerheal"]
