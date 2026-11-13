#!/bin/bash
set -euo pipefail

pid_file="${PID_FILE:-/tmp/ciphomate.pid}"
app_binary="${CIPHOMATE_BIN:-/app/ciphomate}"
app_dir="${CIPHOMATE_DIR:-/app}"

rm -f /run/ciphomate-stop-result /run/ciphomate-stop-result.tmp
echo "time=$(date --iso-8601=seconds) level=INFO component=cron event=app_start"
cd "$app_dir"
"$app_binary" &
echo "$!" > "$pid_file"
