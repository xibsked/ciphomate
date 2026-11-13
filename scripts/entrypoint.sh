#!/bin/bash
set -euo pipefail

: "${START_TIME:?START_TIME must be set}"
: "${STOP_TIME:?STOP_TIME must be set}"

time_pattern='^([01]?[0-9]|2[0-3]):[0-5][0-9]$'
if [[ ! "$START_TIME" =~ $time_pattern ]]; then
  printf 'time=%s level=ERROR component=cron event=invalid_schedule variable=START_TIME value=%s expected=HH:MM\n' \
    "$(date --iso-8601=seconds)" "$START_TIME" >&2
  exit 1
fi
START_HOUR=${BASH_REMATCH[1]}
START_MIN=${START_TIME#*:}

if [[ ! "$STOP_TIME" =~ $time_pattern ]]; then
  printf 'time=%s level=ERROR component=cron event=invalid_schedule variable=STOP_TIME value=%s expected=HH:MM\n' \
    "$(date --iso-8601=seconds)" "$STOP_TIME" >&2
  exit 1
fi
STOP_HOUR=${BASH_REMATCH[1]}
STOP_MIN=${STOP_TIME#*:}

/app/write-cron-env.sh /run/ciphomate-cron-env.sh

{
  printf 'SHELL=/bin/bash\n'
  printf 'BASH_ENV=/run/ciphomate-cron-env.sh\n'
  printf '%s %s * * * root /app/start.sh >> /proc/1/fd/1 2>> /proc/1/fd/2\n' "$START_MIN" "$START_HOUR"
  printf '%s %s * * * root /app/stop.sh >> /proc/1/fd/1 2>> /proc/1/fd/2\n' "$STOP_MIN" "$STOP_HOUR"
} > /etc/crontab

ln -fs "/usr/share/zoneinfo/$TZ" /etc/localtime
dpkg-reconfigure -f noninteractive tzdata
printf 'time=%s level=INFO component=cron event=cron_started timezone=%s\n' \
  "$(date --iso-8601=seconds)" "$TZ"

exec cron -f
