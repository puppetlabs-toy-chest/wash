# Run with: docker run --device /dev/fuse --cap-add SYS_ADMIN -v /var/run/docker.sock:/var/run/docker.sock -v /proc:/proc -it <name>
# Warning: these docker options give the container complete access to the host system. This
# container is designed for convenience, not security.
FROM golang:alpine AS build_base

RUN apk update && apk add --no-cache git
WORKDIR /workdir

COPY go.mod .
COPY go.sum .
RUN go mod download

FROM build_base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

FROM alpine AS wash
RUN apk update && apk add --no-cache fuse ca-certificates
COPY --from=builder /workdir/wash /bin/wash

ENTRYPOINT ["wash"]
