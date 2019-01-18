# Run with: docker run --device /dev/fuse --cap-add SYS_ADMIN -v /var/run/docker.sock:/var/run/docker.sock
# Then enter: docker exec -it <name> sh

FROM golang:alpine AS build_base

RUN apk update && apk add --no-cache git
WORKDIR /workdir

COPY go.mod .
COPY go.sum .
RUN go mod download

FROM build_base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"
RUN cd cmd/meta && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s"

FROM alpine AS wash
RUN apk update && apk add --no-cache fuse ca-certificates
COPY --from=builder /workdir/wash /bin/wash
COPY --from=builder /workdir/cmd/meta/meta /bin/meta
WORKDIR /mnt

# Eventually move to entrypoint with #!/bin/sh\nwash /mnt 2>/var/log/wash.log &\nsh
# Challenging now because /mnt is a different inode after wash finishes initializing.
ENTRYPOINT ["wash", "/mnt"]
