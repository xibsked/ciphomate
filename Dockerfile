FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go build -o ciphomate main.go

FROM debian:bookworm-slim

# Install cron and tzdata for timezone support
RUN apt-get update && \
    apt-get install -y cron tzdata && \
    apt-get clean

# Set the default timezone — can be overridden by ENV
ENV TZ=Asia/Kolkata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/ciphomate /app/ciphomate
COPY scripts/start.sh /app/start.sh
COPY scripts/stop.sh /app/stop.sh
COPY scripts/entrypoint.sh /app/entrypoint.sh
COPY scripts/write-cron-env.sh /app/write-cron-env.sh

RUN chmod +x /app/start.sh /app/stop.sh /app/entrypoint.sh /app/write-cron-env.sh

CMD ["/app/entrypoint.sh"]
