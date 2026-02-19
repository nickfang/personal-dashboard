# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy ONLY the dashboard service (it is self-contained)
COPY . .

# Build the dashboard service
RUN go mod tidy
RUN go build -o /app/dashboard-api cmd/server/main.go

# Stage 2: Final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/dashboard-api .

# Cloud Run uses the PORT environment variable
ENV PORT=8080

CMD ["./dashboard-api"]
