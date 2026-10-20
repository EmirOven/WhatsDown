#! /bin/bash
#
# Usage:
#   ./run.sh                  Dev mode with live reload (default):
#                               - ui/*.blp, *.ui, *.css changes apply in the running app
#                               - Go changes trigger a rebuild + restart
#   WHATSDOWN_DEV=0 ./run.sh  Plain run using the embedded UI assets.

set -u
cd "$(dirname "$0")" || exit 1

compile_ui() {
	blueprint-compiler batch-compile ui ui ui/*.blp ui/components/*.blp
}

export WHATSDOWN_DEV="${WHATSDOWN_DEV:-1}"

if [[ "$WHATSDOWN_DEV" != "1" ]]; then
	compile_ui || exit 1
	exec go run .
fi

DEV_DIR=.dev
BIN="$DEV_DIR/whatsdown"
STAMP="$DEV_DIR/stamp"
mkdir -p "$DEV_DIR"
app_pid=""

stop_app() {
	if [[ -n "$app_pid" ]] && kill -0 "$app_pid" 2>/dev/null; then
		kill "$app_pid"
		wait "$app_pid" 2>/dev/null
	fi
	app_pid=""
}
trap stop_app EXIT
trap 'exit 0' INT TERM HUP

build_and_start() {
	# Stamp before building so edits made during the build trigger another one.
	touch "$STAMP"
	compile_ui || echo "[dev] blueprint compile failed; building with the previous .ui"
	echo "[dev] building..."
	if go build -o "$BIN.new" .; then
		stop_app
		mv "$BIN.new" "$BIN"
		"./$BIN" &
		app_pid=$!
		echo "[dev] started (pid $app_pid)"
	else
		echo "[dev] build failed; keeping the previous version running"
	fi
}

go_changed() {
	find . \( -path ./subprojects -o -path ./.git -o -path "./$DEV_DIR" \) -prune -o \
		\( -name '*.go' -o -name go.mod -o -name go.sum \) -newer "$STAMP" -print -quit | grep -q .
}

build_and_start
echo "[dev] watching Go files for changes (Ctrl+C to quit)"

while true; do
	sleep 1

	if go_changed; then
		echo "[dev] Go change detected"
		build_and_start
	fi

	if [[ -n "$app_pid" ]] && ! kill -0 "$app_pid" 2>/dev/null; then
		wait "$app_pid"
		code=$?
		app_pid=""
		if [[ $code -eq 0 ]]; then
			echo "[dev] app closed"
			exit 0
		fi
		echo "[dev] app exited with code $code; waiting for changes..."
	fi
done