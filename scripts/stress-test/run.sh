#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.."
state=${GORO_STRESS_STATE:-/tmp/goro-stress}
data=${GORO_STRESS_DATA_DIR:?Set GORO_STRESS_DATA_DIR to your Ragnarok data directory}
count=${1:-12}
[[ $count =~ ^([1-9]|[1-4][0-9]|5[0-7])$ ]] || { echo 'Usage: run.sh [1-57]' >&2; exit 1; }
# Check the entire batch before opening any clients.
[[ -x $state/goro ]] || { echo 'Missing stress-test client.' >&2; exit 1; }
for ((i=1; i<=count; i++)); do
	printf -v account 'stress%02d' "$i"
	[[ -r $state/$account.ini ]] || { echo "Missing config: $account.ini" >&2; exit 1; }
done
if [[ ${GORO_STRESS_SCOPED:-0} != 1 ]]; then
	# Established bots reached 340–360 MiB RSS in the local OOM snapshot.
	# Reserve some growth and isolate the entire batch from desktop applications.
	if ((count * 400 > 6656)); then
		echo 'Estimated memory exceeds the 6.5 GiB batch limit. Use at most 16 bots.' >&2
		exit 1
	fi
	available_kib=$(awk '/^MemAvailable:/ {print $2}' /proc/meminfo)
	if ((available_kib < 8704 * 1024)); then
		echo 'Need 8.5 GiB available for the 6.5 GiB bot limit plus desktop headroom.' >&2
		exit 1
	fi
	exec systemd-run --user --scope --quiet --unit=goro-stress-bots \
		--property=MemoryMax=6656M --property=MemorySwapMax=0 --property=OOMPolicy=kill \
		env GORO_STRESS_SCOPED=1 "$0" "$count"
fi
exec 9>"$state/bots.lock"
flock -n 9 || { echo 'Stress bots are already running.' >&2; exit 1; }

pids=()
cleanup() {
	trap - EXIT INT TERM
	if ((${#pids[@]})); then
		kill "${pids[@]}" 2>/dev/null || true
		wait "${pids[@]}" 2>/dev/null || true
	fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

for ((i=1; i<=count; i++)); do
	printf -v account 'stress%02d' "$i"
	GORO_STRESS_INDEX=$i GOMAXPROCS=1 GOMEMLIMIT=192MiB nice -n 10 "$state/goro" \
		--headless --data-dir "$data" --config "$state/$account.ini" \
		--width 800 --height 600 --script scripts/stress-test/combat.lua \
		--log-level "${GORO_STRESS_LOG_LEVEL:-info}" >"$state/$account.log" 2>&1 &
	pids+=("$!")
	# rAthena's connection flood guard resets after three seconds of quiet.
	sleep 4
done
echo "$count bots started. Watch at: @warp prt_fild08 216 255"
echo "Logs: $state/stress*.log. Ctrl+C stops this batch."
wait -n || true
echo 'A bot exited; stopping the batch. Check its log.' >&2
exit 1
