# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

# Set the working directory to the app root
WORKDIR /app

# Copy the shared library
COPY services/gen/go services/gen/go

# Copy the provider service
COPY services/weather-provider services/weather-provider

# Build the provider
WORKDIR /app/services/weather-provider

# Use go mod edit to point to the local shared module
RUN go mod edit -replace github.com/nickfang/personal-dashboard/services/gen/go=../gen/go
RUN go mod tidy
RUN go build -o /app/weather-provider ./cmd/server/main.go

# Stage 2: Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/weather-provider .

# Cloud Run uses the PORT environment variable
ENV PORT=50051

CMD ["./weather-provider"]
