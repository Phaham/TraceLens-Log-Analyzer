#!/usr/bin/env bash
#
# Simulates a live production incident by appending log lines to a file
# over time. Run the monitor in another terminal first:
#
#   go run ./cmd/log-analyzer -file=live.log -watch
#
# Then start this script:
#
#   bash scripts/simulate-logs.sh live.log

set -euo pipefail

LOGFILE="${1:-live.log}"
> "$LOGFILE"

now() { date -u +"%Y-%m-%d %H:%M:%S"; }

log() {
  local level="$1" msg="$2"
  echo "$(now) [$level] $msg" >> "$LOGFILE"
  echo "  → [$level] $msg"
}

echo "Writing logs to $LOGFILE — Ctrl-C to stop"
echo

# Phase 1: Normal startup
echo "--- Phase 1: Normal startup ---"
log INFO  "Application started successfully"
sleep 1
log INFO  "Database connection pool established (max=50)"
sleep 1
log INFO  "Cache warmed: 12,840 keys loaded in 1.2s"
sleep 1
log INFO  "Listening on :8080"
sleep 2

# Phase 2: Steady traffic
echo "--- Phase 2: Steady traffic ---"
for i in $(seq 1 5); do
  log INFO "Request processed in ${i}2ms — GET /api/v1/users"
  sleep 0.5
done
sleep 2

# Phase 3: Warnings start
echo "--- Phase 3: Warnings ---"
log WARN  "Slow query detected: SELECT * FROM orders took 3.8s"
sleep 1
log WARN  "Database connection pool utilization at 80% (40/50)"
sleep 1
log WARN  "High memory usage detected: 78%"
sleep 2

# Phase 4: Errors escalate
echo "--- Phase 4: Error spike ---"
for i in $(seq 1 6); do
  log ERROR "Database connection pool exhausted - request queued"
  sleep 0.4
done
sleep 1
log ERROR "Request timeout after 30s for endpoint /api/v1/orders"
sleep 0.5
log ERROR "Request timeout after 30s for endpoint /api/v1/users"
sleep 0.5
log ERROR "Failed to send email notification to ops@company.com: SMTP connection refused"
sleep 1

# Phase 5: Critical failure
echo "--- Phase 5: Critical failure ---"
log ERROR "Panic recovered: runtime error: index out of range [3] with length 2"
sleep 0.5
log ERROR "Healthcheck failed: database ping timeout"
sleep 0.5
log FATAL "Database connection lost: dial tcp 10.0.1.20:5432: connection refused"
sleep 1
log FATAL "Out of memory error - heap alloc 3.8GB exceeds 4GB limit"
sleep 1
log INFO  "Graceful shutdown initiated"
sleep 0.5
log WARN  "187 in-flight requests will be dropped"
sleep 1
log INFO  "Application shutdown complete"
sleep 3

# Phase 6: Recovery
echo "--- Phase 6: Recovery ---"
log INFO  "Application started successfully (restart #1)"
sleep 1
log INFO  "Connected to standby database at 10.0.1.21:5432"
sleep 1
log WARN  "Replication lag: 2.1s behind primary"
sleep 1
log INFO  "Request processed in 45ms — GET /api/v1/health"
sleep 1
log INFO  "System stabilized — all health checks passing"

echo
echo "Done. $LOGFILE has $(wc -l < "$LOGFILE" | tr -d ' ') lines."
