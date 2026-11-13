#!/bin/bash

pid_file="${PID_FILE:-/tmp/ciphomate.pid}"

if [ ! -f "$pid_file" ]; then
  echo "time=$(date --iso-8601=seconds) level=INFO component=cron event=pid_not_found"
  exit 0
fi

pid=$(cat "$pid_file")
if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
  echo "time=$(date --iso-8601=seconds) level=ERROR component=cron event=invalid_pid_file path=$pid_file"
  rm -f "$pid_file"
  exit 1
fi

if ! kill -0 "$pid" 2>/dev/null; then
  echo "time=$(date --iso-8601=seconds) level=INFO component=cron event=stale_pid pid=$pid"
  rm -f "$pid_file"
  exit 0
fi

process_executable=$(readlink "/proc/$pid/exe" 2>/dev/null)
if [ "$process_executable" != "/app/ciphomate" ]; then
  echo "time=$(date --iso-8601=seconds) level=WARN component=cron event=pid_process_mismatch pid=$pid"
  rm -f "$pid_file"
  exit 1
fi

if kill "$pid"; then
  rm -f "$pid_file"
  echo "time=$(date --iso-8601=seconds) level=INFO component=cron event=app_stopped pid=$pid"
else
  echo "time=$(date --iso-8601=seconds) level=ERROR component=cron event=app_stop_failed pid=$pid"
  rm -f "$pid_file"
  exit 1
fi
