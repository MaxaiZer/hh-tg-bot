FROM alpine:latest AS base

FROM golang:1.23 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/main cmd/main.go

FROM base AS final
WORKDIR /app
RUN mkdir ./configs & mkdir ./logs
COPY configs/config.yaml ./configs
COPY --from=build /app/build .
CMD ["./main"]