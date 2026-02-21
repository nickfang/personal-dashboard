# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy shared module first (changes less often → better layer caching)
COPY shared/ ./shared/

# Copy service code
COPY weather-collector/ ./weather-collector/

WORKDIR /app/weather-collector
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/weather-collector/bin main.go

# Stage 2: Final image
FROM alpine:3
RUN apk add --no-cache ca-certificates

WORKDIR /root/
COPY --from=builder /app/weather-collector/bin ./weather-collector

CMD ["./weather-collector"]
