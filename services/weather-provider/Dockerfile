# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy ONLY the provider service (it is self-contained)
COPY . .

# Build the provider service
RUN go mod tidy
RUN go build -o /app/weather-provider cmd/server/main.go

# Stage 2: Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/weather-provider .

# Cloud Run uses the PORT environment variable
ENV PORT=50051

CMD ["./weather-provider"]
