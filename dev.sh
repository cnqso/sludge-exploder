#!/usr/bin/env bash
# Dev helper: spin up daemon + app together, make resets easy. See usage()
# below, or just run `./dev.sh` with no args.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO_ROOT"

DEV_DIR="$REPO_ROOT/.dev"
DAEMON_PID_FILE="$DEV_DIR/daemon.pid"
APP_PID_FILE="$DEV_DIR/app.pid"
DAEMON_LOG="$DEV_DIR/daemon.log"
APP_LOG="$DEV_DIR/app.log"

# Where the app/daemon keep their sockets, control token, and persisted
# state -- see shared/paths.go.
SUPPORT_DIR="$HOME/Library/Application Support/SludgeExploder"

mkdir -p "$DEV_DIR"

is_running() {
    local pid_file="$1"
    [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null
}

stop_one() {
    local name="$1" pid_file="$2"
    if is_running "$pid_file"; then
        local pid
        pid=$(cat "$pid_file")
        echo "Stopping $name (pid $pid)..."
        kill "$pid" 2>/dev/null || true
        for _ in $(seq 1 20); do
            kill -0 "$pid" 2>/dev/null || break
            sleep 0.2
        done
        kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$pid_file"
}

cmd_stop() {
    stop_one "app" "$APP_PID_FILE"
    stop_one "daemon" "$DAEMON_PID_FILE"
}

cmd_build() {
    echo "Building..."
    make build
}

cmd_start() {
    cmd_stop
    cmd_build

    echo "Starting daemon (log-only enforcement -- see note below)..."
    nohup ./bin/daemon > "$DAEMON_LOG" 2>&1 &
    echo $! > "$DAEMON_PID_FILE"
    sleep 0.5

    echo "Starting app..."
    nohup ./bin/app > "$APP_LOG" 2>&1 &
    echo $! > "$APP_PID_FILE"

    echo
    echo "daemon: pid $(cat "$DAEMON_PID_FILE"), log $DAEMON_LOG"
    echo "app:    pid $(cat "$APP_PID_FILE"), log $APP_LOG"
    echo
    echo "Note: this script never passes --enforce to the daemon on purpose --"
    echo "testing real browser-closing should stay a deliberate manual step"
    echo "('./bin/daemon --enforce ...' by hand), not something a generic"
    echo "start script could accidentally enable."
    echo
    echo "Tailing logs now (Ctrl-C stops watching, NOT the daemon/app -- they"
    echo "keep running in the background; use '$0 stop' to actually stop them)."
    echo
    cmd_logs
}

cmd_reset() {
    cmd_stop
    echo "Wiping daemon/app runtime state in: $SUPPORT_DIR"
    echo "  (sockets, control token, lock state -- NOT your saved preferences)"
    rm -f "$SUPPORT_DIR/heartbeat.sock" "$SUPPORT_DIR/control.sock" \
          "$SUPPORT_DIR/control.token" "$SUPPORT_DIR/lock_state.json"
    cmd_start
}

cmd_wipe_all() {
    cmd_stop
    echo "This wipes ALL app/daemon state, including saved preferences, at:"
    echo "  $SUPPORT_DIR"
    read -r -p "Are you sure? [y/N] " reply
    if [[ "$reply" =~ ^[Yy]$ ]]; then
        rm -rf "$SUPPORT_DIR"
        echo "Wiped."
    else
        echo "Cancelled."
    fi
}

cmd_status() {
    if is_running "$DAEMON_PID_FILE"; then
        echo "daemon: running (pid $(cat "$DAEMON_PID_FILE"))"
    else
        echo "daemon: not running"
    fi
    if is_running "$APP_PID_FILE"; then
        echo "app: running (pid $(cat "$APP_PID_FILE"))"
    else
        echo "app: not running"
    fi
}

cmd_logs() {
    tail -f "$DAEMON_LOG" "$APP_LOG"
}

usage() {
    cat <<EOF
Usage: $0 <command>

  start      Stop anything running, rebuild, start daemon + app in background
  restart    Alias for start
  stop       Stop daemon + app
  reset      Stop, wipe sockets/token/lock state (keeps saved preferences), start
  wipe-all   Stop, wipe EVERYTHING including saved preferences (asks to confirm)
  status     Show whether daemon/app are running
  logs       Tail both log files
EOF
}

case "${1:-}" in
    start|restart) cmd_start ;;
    stop) cmd_stop ;;
    reset) cmd_reset ;;
    wipe-all) cmd_wipe_all ;;
    status) cmd_status ;;
    logs) cmd_logs ;;
    *) usage; exit 1 ;;
esac
