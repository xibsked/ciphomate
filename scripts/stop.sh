#!/bin/bash
set -u

pid_file="${PID_FILE:-/tmp/ciphomate.pid}"
result_file="${STOP_RESULT_FILE:-/run/ciphomate-stop-result}"
graceful_wait_seconds="${STOP_GRACE_SECONDS:-12}"
app_binary="${CIPHOMATE_BIN:-/app/ciphomate}"
app_dir="${CIPHOMATE_DIR:-/app}"
expected_executable="${CIPHOMATE_EXE_PATH:-$(readlink -f "$app_binary")}"

log_event() {
  local level="$1"
  local event="$2"
  shift 2
  printf 'time=%s level=%s component=cron event=%s %s\n' \
    "$(date --iso-8601=seconds)" "$level" "$event" "$*"
}

if [[ ! "$graceful_wait_seconds" =~ ^[1-9][0-9]*$ ]]; then
  log_event ERROR invalid_grace_period
  exit 2
fi

run_off_fallback() {
  log_event INFO off_fallback_started
  if (cd "$app_dir" && "$app_binary" --off); then
    log_event INFO off_fallback_accepted_by_tuya
    return 0
  fi
  log_event ERROR off_fallback_failed
  return 1
}

process_running() {
  local stat rest state
  kill -0 "$1" 2>/dev/null || return 1
  [ -r "/proc/$1/stat" ] || return 0
  IFS= read -r stat < "/proc/$1/stat" || return 0
  rest="${stat##*) }"
  state="${rest%% *}"
  [ "$state" != Z ]
}

finish() {
  local app_pid="${1:-}"
  local do_fallback="$2"
  local fallback_ok=0

  if [ "$do_fallback" -eq 1 ]; then
    run_off_fallback || fallback_ok=1
  fi

  if [ -n "$app_pid" ] && process_running "$app_pid"; then
    if kill -KILL "$app_pid" 2>/dev/null; then
      for ((tick = 0; tick < 20; tick++)); do
        process_running "$app_pid" || break
        sleep 0.1
      done
      if process_running "$app_pid"; then
        log_event ERROR app_force_termination_not_observed "pid=$app_pid"
      else
        log_event WARN app_force_terminated "pid=$app_pid"
      fi
    else
      if process_running "$app_pid"; then
        log_event ERROR app_force_termination_failed "pid=$app_pid"
      fi
    fi
  fi

  rm -f "$pid_file"
  if [ "$fallback_ok" -eq 1 ]; then
    log_event ERROR scheduled_stop_not_confirmed_by_tuya
    return 1
  fi
  return 0
}

if [ ! -f "$pid_file" ]; then
  log_event WARN app_pid_missing
  finish "" 1
  exit $?
fi

pid=$(cat "$pid_file")
if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
  log_event ERROR invalid_pid_file "path=$pid_file"
  finish "" 1
  exit $?
fi

if ! process_running "$pid"; then
  log_event WARN app_pid_stale "pid=$pid"
  finish "" 1
  exit $?
fi

process_executable=$(readlink "/proc/$pid/exe" 2>/dev/null || true)
if [ "$process_executable" != "$expected_executable" ]; then
  log_event ERROR pid_process_mismatch "pid=$pid"
  finish "" 1
  exit $?
fi

rm -f "$result_file" "$result_file.tmp"
if ! kill -TERM "$pid"; then
  log_event WARN graceful_signal_failed "pid=$pid"
  finish "$pid" 1
  exit $?
fi
log_event INFO graceful_shutdown_requested "pid=$pid"

wait_ticks=$((graceful_wait_seconds * 10))
for ((tick = 0; tick < wait_ticks; tick++)); do
  if ! process_running "$pid"; then
    if [ -f "$result_file" ] && [ "$(cat "$result_file")" = accepted ]; then
      log_event INFO graceful_off_accepted_by_tuya "pid=$pid"
      rm -f "$pid_file" "$result_file"
      exit 0
    fi
    log_event WARN graceful_off_not_confirmed "pid=$pid"
    finish "" 1
    exit $?
  fi
  sleep 0.1
done

log_event ERROR graceful_shutdown_timed_out "pid=$pid" "wait_seconds=$graceful_wait_seconds"
finish "$pid" 1
exit $?
