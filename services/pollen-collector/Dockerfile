# Stage 1: Build
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

# Copy shared module first (changes less often → better layer caching)
COPY shared/ ./shared/

# Copy service code
COPY pollen-collector/ ./pollen-collector/

WORKDIR /app/pollen-collector
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/pollen-collector/bin main.go

# Stage 2: Final image
FROM alpine:3
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/pollen-collector/bin ./pollen-collector

CMD ["./pollen-collector"]