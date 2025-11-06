FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o import-gitlab-commits .

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add git ca-certificates

COPY --from=builder /app/import-gitlab-commits /usr/local/bin/import-gitlab-commits
RUN chmod +x /usr/local/bin/import-gitlab-commits

ENTRYPOINT ["/bin/sh", "-c", "git config --global user.name \"${COMMITTER_NAME}\" && git config --global user.email \"${COMMITTER_EMAIL}\" && import-gitlab-commits"]

