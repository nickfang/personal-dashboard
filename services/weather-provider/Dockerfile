# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy shared module first (changes less often → better layer caching)
COPY shared/ ./shared/

# Copy service code
COPY weather-provider/ ./weather-provider/

WORKDIR /app/weather-provider
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/weather-provider/bin cmd/server/main.go

# Stage 2: Final image
FROM alpine:3
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/weather-provider/bin ./weather-provider

# Cloud Run uses the PORT environment variable
ENV PORT=50051

CMD ["./weather-provider"]
