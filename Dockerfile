FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /fluxcache .

FROM alpine:3.22
RUN apk add --no-cache tzdata ca-certificates
COPY --from=builder /fluxcache /fluxcache
VOLUME /data
EXPOSE 6379 8081
ENTRYPOINT ["/fluxcache"]
