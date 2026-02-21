FROM golang:1.25.6-alpine AS builder

WORKDIR /app

COPY shared/ ./shared/
COPY pollen-provider/ ./pollen-provider/

WORKDIR /app/pollen-provider
RUN go mod tidy
RUN go build -o /app/pollen-provider/bin cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/pollen-provider/bin ./pollen-provider

ENV PORT=50052

CMD ["./pollen-provider"]