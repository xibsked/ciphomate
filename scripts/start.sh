#!/bin/bash
set -euo pipefail

pid_file="${PID_FILE:-/tmp/ciphomate.pid}"

echo "time=$(date --iso-8601=seconds) level=INFO component=cron event=app_start"
/app/ciphomate &
echo "$!" > "$pid_file"
