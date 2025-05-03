FROM stagex/pallet-go AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go vet ./...
RUN go build -trimpath -ldflags "-s -w -buildid=" .
RUN install -Dm755 -t /rootfs/usr/bin proxy64

FROM stagex/core-filesystem
COPY --from=builder /rootfs/ /
ENTRYPOINT [ "proxy64" ]
