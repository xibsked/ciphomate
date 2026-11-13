# Ciphomate Smart Socket Pump Automator (Go + Docker + Cron)

This utility is a **Go-based application** that uses a **2-stage Docker build** and is managed using **Docker Compose**. It automates turning a **Wipro smart socket (Tuya-based)** on and off to control a water pump, based on scheduling and real-time power usage.

---

## Configuration

Copy `.env.example` to `.env` and set your Tuya API host, credentials, device IDs, and schedule:

```sh
cp .env.example .env
```

Docker Compose injects the variables from `.env` into the container at runtime. The file is not copied into the image or sent as part of the Docker build context. Compose's project `.env` file can also supply values for `${...}` interpolation in the Compose file, but that does not by itself pass variables into a container; the service's `env_file` setting does that.

The app reads `HOST`, `CLIENT_ID`, `SECRET`, `PUMP_DEVICE_ID`, `TANK_DEVICE_ID`, `MAX_RETRIES`, `RETRY_DELAY`, `PUMP_CURRENT_THRESHOLD`, `TANK_CURRENT_THRESHOLD`, `LOW_CURRENT_MINUTES`, and `MONITOR_INTERVAL`. Cron uses `START_TIME`, `STOP_TIME`, `TZ`, and `STOP_GRACE_SECONDS`. The stop grace period defaults to 12 seconds and must exceed the Tuya HTTP timeout. The container defaults to `TZ=Asia/Kolkata` if it is not set.

Logs use structured, human-readable text. Set `LOG_LEVEL` to `debug`, `info` (default), `warn`, or `error`; `debug` includes successful Tuya request status and duration. Log output deliberately excludes credentials, access tokens, signatures, and raw API response bodies.

The Compose service and cron-launched app use the same runtime environment. Cron receives it through a root-readable temporary file under `/run`; credentials are not embedded in the image. Environment variables are still visible to container administrators, so use a secrets manager when that level of protection is required.

### Viewing logs

Cron job output and errors are forwarded to the container's stdout and stderr, so they appear in the terminal running `docker compose up` and can be followed later with:

```sh
docker compose logs -f ciphomate
```

The scheduled start and stop scripts also print status messages; the start message includes a timestamp. Cron's own daemon messages are separate system log messages; the container forwards the scheduled jobs' output rather than running a separate syslog service.

## How It Works

- Docker Compose injects `START_TIME` and `STOP_TIME` from `.env` into the container.
- At `START_TIME`, a **cron job** is triggered to **start the app** (inside the container).
- The core Go application uses the **Tuya API** to:
  - Fetch the **current status** of the Wipro socket.
  - Turn the socket **on** at the scheduled time.
  - Automatically **turn it off** either:
    - At `STOP_TIME` after the app cancels monitoring, requests OFF from Tuya, and waits for Tuya's API response, or
    - If it detects that the pump is not drawing enough **current**.

---

## Core Logic

- If the **current (in mA)** drawn by the socket is **below** the configured `CURRENT_THRESHOLD` **for longer than** `LOW_CURRENT_MINUTES`, the app turns off the socket.
- This logic is designed to detect **dry run scenarios**, i.e., when the pump is powered but not pulling water.

---

## Retry Mechanism

- If the socket fails to activate within the inching time window, the app retries after:
  - `RETRY_DELAY` between retry attempts, up to `MAX_RETRIES`.

This ensures robustness while preventing rapid repeated triggering.

---

## Docker Architecture

- **Stage 1:** Builds the Go binary inside a lightweight container.
- **Stage 2:** Runs the built binary with minimal dependencies using Docker Compose.

---

## Scheduling (via Cron)

- Controlled externally by cron using `START_TIME` and `STOP_TIME` from `.env`.
- At `STOP_TIME`, cron requests graceful app shutdown. The app cancels monitoring/retries before sending OFF, then exits after Tuya responds.
- If the app is already gone, Tuya rejects the graceful OFF request, or the app misses the grace period, cron makes one independent OFF request before force-terminating any remaining app process. The result is successful only when Tuya accepts an OFF request; this does not confirm the physical device changed state.

---

## Roadmap

1. Improve retry mechanism robustness.
2. Allow remote configuration of **inching duration** (Wipro socket allows up to 1 hour).
3. Align inching and actual **pump run time** more precisely.
4. Add a **"Tank Full" detection** using another socket or sensor.
5. Consider using **power threshold** instead of current for pump detection (possibly more reliable).
6. Build a **web-based UI** for configuring thresholds, timers, and logs.

---

## License

**Open license** – free for personal, commercial, or experimental use.  
**Contributions are welcome.**

> **Disclaimer**: This code is dynamic and may contain bugs. Use at your own risk.
