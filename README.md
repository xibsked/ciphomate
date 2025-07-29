# Ciphomate Smart Socket Pump Automator (Go + Docker + Cron)

This utility is a **Go-based application** that uses a **2-stage Docker build** and is managed using **Docker Compose**. It automates turning a **Wipro smart socket (Tuya-based)** on and off to control a water pump, based on scheduling and real-time power usage.

---

## How It Works

- The application reads `START_TIME` and `STOP_TIME` from a `.env` file.
- At `START_TIME`, a **cron job** is triggered to **start the app** (inside the container).
- The core Go application uses the **Tuya API** to:
  - Fetch the **current status** of the Wipro socket.
  - Turn the socket **on** at the scheduled time.
  - Automatically **turn it off** either:
    - At `STOP_TIME` (cron kills the app and triggers a final off signal), or
    - If it detects that the pump is not drawing enough **current**.

---

## Core Logic

- If the **current (in mA)** drawn by the socket is **below** the configured `CURRENT_THRESHOLD` **for longer than** `LOW_CURRENT_MINUTES`, the app turns off the socket.
- This logic is designed to detect **dry run scenarios**, i.e., when the pump is powered but not pulling water.

---

## Retry Mechanism

- If the socket fails to activate within the inching time window, the app retries after:
  - `RETRY_DELAY_1` (first retry)
  - `RETRY_DELAY_2` (second retry)

This ensures robustness while preventing rapid repeated triggering.

---

## Docker Architecture

- **Stage 1:** Builds the Go binary inside a lightweight container.
- **Stage 2:** Runs the built binary with minimal dependencies using Docker Compose.

---

## Scheduling (via Cron)

- Controlled externally by cron using `START_TIME` and `STOP_TIME` from `.env`.
- `STOP_TIME` not only stops the app but also triggers a **final off** signal to ensure the pump doesn't stay on.

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
