FROM --platform=${BUILDPLATFORM} golang:1.23-alpine AS builder
LABEL maintainer="Tom Helander <thomas.helander@gmail.com>"

RUN apk add --no-cache make curl git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS TARGETARCH
RUN make GOOS=$TARGETOS GOARCH=$TARGETARCH build

FROM alpine:3.22
LABEL maintainer="Tom Helander <thomas.helander@gmail.com>"

WORKDIR /app
RUN set -eux; \
    apk update; \
    apk upgrade -v --no-cache; \
    apk cache purge

COPY --from=builder --chmod=0755 /src/output/openmeteo_exporter .

EXPOSE 9812

ENTRYPOINT ["/app/openmeteo_exporter"]
