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
COPY --from=builder /app/.env /app/.env

# Create start/stop scripts
RUN echo '#!/bin/bash\n/app/ciphomate --env /app/.env & echo $! > /tmp/ciphomate.pid' > /app/start.sh && \
    echo '#!/bin/bash\nif [ -f /tmp/ciphomate.pid ]; then kill $(cat /tmp/ciphomate.pid) && rm /tmp/ciphomate.pid; else echo "PID not found"; fi' > /app/stop.sh && \
    chmod +x /app/start.sh /app/stop.sh

# Run cron with dynamic times from ENV
CMD bash -c '\
  START_MIN=$(echo "$START_TIME" | cut -d: -f2); \
  START_HOUR=$(echo "$START_TIME" | cut -d: -f1); \
  STOP_MIN=$(echo "$STOP_TIME" | cut -d: -f2); \
  STOP_HOUR=$(echo "$STOP_TIME" | cut -d: -f1); \
  echo "$START_MIN $START_HOUR * * * root /app/start.sh >> /var/log/cron.log 2>&1" > /etc/crontab; \
  echo "$STOP_MIN $STOP_HOUR * * * root /app/stop.sh >> /var/log/cron.log 2>&1" >> /etc/crontab; \
  ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && dpkg-reconfigure -f noninteractive tzdata; \
  echo "Container timezone: $(date)"; \
  cron -f'
