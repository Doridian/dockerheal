FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.24-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download

COPY . /src
RUN go build -ldflags='-s -w' -trimpath -o /dockerheal .

FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch

COPY LICENSE /LICENSE
COPY --from=builder --chown=0:0 --chmod=755 /dockerheal /dockerheal

ENTRYPOINT ["/dockerheal"]
