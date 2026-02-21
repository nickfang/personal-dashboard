# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy shared module first (changes less often → better layer caching)
COPY shared/ ./shared/

# Copy service code
COPY dashboard-api/ ./dashboard-api/

WORKDIR /app/dashboard-api
RUN go mod tidy
RUN go build -o /app/dashboard-api/bin cmd/server/main.go

# Stage 2: Final image
FROM alpine:3
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/dashboard-api/bin ./dashboard-api

# Cloud Run uses the PORT environment variable
ENV PORT=8080

CMD ["./dashboard-api"]
