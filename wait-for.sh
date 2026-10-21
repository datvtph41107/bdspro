#!/bin/sh
# Wait for a TCP service before starting the requested command.
set -eu

if [ "$#" -lt 1 ]; then
	echo "usage: wait-for.sh host:port [-- command args...]" >&2
	exit 2
fi
target="$1"
shift
host="${target%:*}"
port="${target##*:}"
if [ "${1:-}" = "--" ]; then
	shift
fi
attempt=0
until nc -z -w 1 "$host" "$port" >/dev/null 2>&1; do
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 30 ]; then
		echo "connection timed out: $target" >&2
		exit 1
	fi
	sleep 1
done
if [ "$#" -gt 0 ]; then
	exec "$@"
fi
