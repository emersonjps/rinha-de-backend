FROM golang:1.22-alpine AS builder-dataset
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/converter/ ./cmd/converter/
COPY resources/ ./resources/
RUN go run ./cmd/converter/main.go -out /app/references.bin

FROM golang:1.22-alpine AS builder-api
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rinha-api ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder-api /app/rinha-api ./rinha-api
COPY --from=builder-dataset /app/references.bin /data/references.bin
EXPOSE 8080
ENV GOMEMLIMIT=150MiB
ENV GOGC=1000
ENTRYPOINT ["./rinha-api"]
