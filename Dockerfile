FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o import-gitlab-commits

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add git ca-certificates

COPY --from=builder /app/import-gitlab-commits /usr/local/bin/import-gitlab-commits

RUN chmod +x /usr/local/bin/import-gitlab-commits

ENTRYPOINT ["import-gitlab-commits"]
