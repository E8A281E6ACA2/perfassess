#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${PERFASSESS_REPO_URL:-https://github.com/E8A281E6ACA2/perfassess.git}"
WORK_DIR="${PERFASSESS_BOOTSTRAP_DIR:-$HOME/perfassess}"
GO_VERSION="${PERFASSESS_GO_VERSION:-1.25.3}"
OUTPUT_DIR="${PERFASSESS_AUTO_DIR:-/tmp/perfassess-auto}"
BOOTSTRAP_TESTS_SET="${PERFASSESS_BOOTSTRAP_TESTS+x}"
RUN_TESTS="${PERFASSESS_BOOTSTRAP_TESTS:-1}"
START_WEB="${PERFASSESS_BOOTSTRAP_WEB:-0}"
WEB_PORT="${PERFASSESS_WEB_PORT:-8080}"
ACTION="run"
PROGRESS_SERVER_PID=""
LOW_MEMORY_THRESHOLD_MB="${PERFASSESS_LOW_MEMORY_THRESHOLD_MB:-768}"
BOOTSTRAP_SWAP_MODE="${PERFASSESS_BOOTSTRAP_SWAP:-auto}"
BOOTSTRAP_SWAP_SIZE_MB="${PERFASSESS_BOOTSTRAP_SWAP_SIZE_MB:-1024}"
BOOTSTRAP_SWAP_FILE="${PERFASSESS_BOOTSTRAP_SWAP_FILE:-$OUTPUT_DIR/bootstrap.swap}"
BOOTSTRAP_SWAP_ACTIVE="0"

usage() {
  cat <<EOF
Usage: scripts/bootstrap.sh [options]

Options:
  --dir PATH       Source directory to clone or use. Default: $WORK_DIR
  --go VERSION    Go version to install when missing or too old. Default: $GO_VERSION
  --web           Start the realtime Material Design progress page.
  --port PORT     Web report port when --web is used. Default: $WEB_PORT
  --clean         Remove build output and auto-test output.
  --clean-all     Remove build output, auto-test output, and the cloned source directory.
  -h, --help      Show this help.

Environment:
  PERFASSESS_BOOTSTRAP_TESTS=0   Skip go test ./...
  PERFASSESS_BOOTSTRAP_WEB=1     Start the realtime Material Design progress page.
  PERFASSESS_BOOTSTRAP_SWAP=auto Create temporary swap on low-memory Linux hosts. Set 0 to disable.
  PERFASSESS_LOW_MEMORY_THRESHOLD_MB=768  Memory threshold for low-memory mode.
  PERFASSESS_WEB_PORT=9090       Web report port.
  PERFASSESS_AUTO_OPTIONAL=auto  Let acceptance run optional checks when dependencies exist.
EOF
}

info() {
  echo "[INFO] $*"
}

success() {
  echo "[OK] $*"
}

fail() {
  echo "[ERROR] $*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)
      [[ $# -ge 2 ]] || fail "--dir requires a path"
      WORK_DIR="$2"
      shift 2
      ;;
    --go)
      [[ $# -ge 2 ]] || fail "--go requires a version"
      GO_VERSION="$2"
      shift 2
      ;;
    --web)
      START_WEB="1"
      shift
      ;;
    --port)
      [[ $# -ge 2 ]] || fail "--port requires a port"
      WEB_PORT="$2"
      shift 2
      ;;
    --clean)
      ACTION="clean"
      shift
      ;;
    --clean-all)
      ACTION="clean-all"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "Unknown option: $1"
      ;;
  esac
done

need_sudo() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    SUDO=()
  elif command -v sudo >/dev/null 2>&1; then
    SUDO=(sudo)
  else
    fail "This operation needs root privileges. Run as root or install sudo."
  fi
}

install_packages() {
  local packages=(git curl ca-certificates tar gzip make python3)
  local missing=()

  command -v git >/dev/null 2>&1 || missing+=(git)
  command -v curl >/dev/null 2>&1 || missing+=(curl)
  command -v make >/dev/null 2>&1 || missing+=(make)
  command -v python3 >/dev/null 2>&1 || missing+=(python3)
  command -v tar >/dev/null 2>&1 || missing+=(tar)

  if [[ ${#missing[@]} -eq 0 ]]; then
    success "base tools are present"
    return
  fi

  need_sudo
  info "installing missing base tools: ${missing[*]}"

  if command -v apt-get >/dev/null 2>&1; then
    "${SUDO[@]}" apt-get update
    "${SUDO[@]}" apt-get install -y "${packages[@]}"
  elif command -v dnf >/dev/null 2>&1; then
    "${SUDO[@]}" dnf install -y "${packages[@]}"
  elif command -v yum >/dev/null 2>&1; then
    "${SUDO[@]}" yum install -y "${packages[@]}"
  elif command -v apk >/dev/null 2>&1; then
    "${SUDO[@]}" apk add --no-cache "${packages[@]}"
  elif command -v pacman >/dev/null 2>&1; then
    "${SUDO[@]}" pacman -Sy --noconfirm "${packages[@]}"
  else
    fail "No supported package manager found. Install these first: ${packages[*]}"
  fi
}

version_ge() {
  local current="$1"
  local required="$2"
  local current_major current_minor current_patch required_major required_minor required_patch
  IFS=. read -r current_major current_minor current_patch <<<"$current"
  IFS=. read -r required_major required_minor required_patch <<<"$required"
  current_minor="${current_minor:-0}"
  current_patch="${current_patch:-0}"
  required_minor="${required_minor:-0}"
  required_patch="${required_patch:-0}"

  (( current_major > required_major )) && return 0
  (( current_major < required_major )) && return 1
  (( current_minor > required_minor )) && return 0
  (( current_minor < required_minor )) && return 1
  (( current_patch >= required_patch ))
}

current_go_version() {
  if command -v go >/dev/null 2>&1; then
    go version | sed -n 's/^go version go\([0-9][^ ]*\).*/\1/p'
  fi
}

install_go() {
  local current
  current="$(current_go_version || true)"

  if [[ -n "$current" ]] && version_ge "$current" "$GO_VERSION"; then
    success "Go $current is present"
    return
  fi

  need_sudo

  local os arch archive url tmp
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) fail "Unsupported architecture: $(uname -m)" ;;
  esac

  case "$os" in
    linux|darwin) ;;
    *) fail "Unsupported OS for automatic Go install: $os" ;;
  esac

  archive="go${GO_VERSION}.${os}-${arch}.tar.gz"
  url="https://go.dev/dl/${archive}"
  tmp="$(mktemp -d)"

  info "installing Go $GO_VERSION from $url"
  curl -fsSL "$url" -o "$tmp/$archive"
  "${SUDO[@]}" rm -rf /usr/local/go
  "${SUDO[@]}" tar -C /usr/local -xzf "$tmp/$archive"
  rm -rf "$tmp"

  if [[ -d /etc/profile.d ]]; then
    printf '%s\n' 'export PATH=/usr/local/go/bin:$PATH' | "${SUDO[@]}" tee /etc/profile.d/perfassess-go.sh >/dev/null
  fi

  export PATH="/usr/local/go/bin:$PATH"
  success "Go installed: $(go version)"
}

memory_total_mb() {
  if [[ -r /proc/meminfo ]]; then
    awk '/MemTotal:/ { printf "%d", $2 / 1024 }' /proc/meminfo
  else
    echo 0
  fi
}

swap_total_mb() {
  if [[ -r /proc/meminfo ]]; then
    awk '/SwapTotal:/ { printf "%d", $2 / 1024 }' /proc/meminfo
  else
    echo 0
  fi
}

configure_low_memory_mode() {
  local mem_mb swap_mb
  mem_mb="$(memory_total_mb)"
  swap_mb="$(swap_total_mb)"

  if [[ "$mem_mb" -le 0 || "$mem_mb" -ge "$LOW_MEMORY_THRESHOLD_MB" ]]; then
    return
  fi

  info "low-memory host detected: ${mem_mb}MB RAM, ${swap_mb}MB swap"
  export GOMAXPROCS="${GOMAXPROCS:-1}"
  if [[ "${GOFLAGS:-}" != *"-p="* && "${GOFLAGS:-}" != *"-p "* ]]; then
    export GOFLAGS="${GOFLAGS:-} -p=1"
    GOFLAGS="${GOFLAGS# }"
  fi
  export PERFASSESS_LOW_MEMORY="1"

  if [[ -z "$BOOTSTRAP_TESTS_SET" ]]; then
    RUN_TESTS="0"
    info "skipping unit tests on low-memory host; set PERFASSESS_BOOTSTRAP_TESTS=1 to force them"
  fi

  if [[ "$BOOTSTRAP_SWAP_MODE" == "0" || "$BOOTSTRAP_SWAP_MODE" == "false" ]]; then
    return
  fi
  if [[ "$swap_mb" -ge 512 ]]; then
    return
  fi
  if [[ "$(uname -s)" != "Linux" ]]; then
    return
  fi
  if ! command -v mkswap >/dev/null 2>&1 || ! command -v swapon >/dev/null 2>&1; then
    info "swap tools are unavailable; continuing with reduced Go build parallelism"
    return
  fi

  need_sudo
  mkdir -p "$(dirname "$BOOTSTRAP_SWAP_FILE")"
  info "creating temporary ${BOOTSTRAP_SWAP_SIZE_MB}MB swap file for bootstrap build"
  if command -v fallocate >/dev/null 2>&1; then
    if ! "${SUDO[@]}" fallocate -l "${BOOTSTRAP_SWAP_SIZE_MB}M" "$BOOTSTRAP_SWAP_FILE"; then
      info "temporary swap allocation failed; continuing with reduced Go build parallelism"
      "${SUDO[@]}" rm -f "$BOOTSTRAP_SWAP_FILE" >/dev/null 2>&1 || true
      return
    fi
  else
    if ! "${SUDO[@]}" dd if=/dev/zero of="$BOOTSTRAP_SWAP_FILE" bs=1M count="$BOOTSTRAP_SWAP_SIZE_MB" status=none; then
      info "temporary swap allocation failed; continuing with reduced Go build parallelism"
      "${SUDO[@]}" rm -f "$BOOTSTRAP_SWAP_FILE" >/dev/null 2>&1 || true
      return
    fi
  fi
  "${SUDO[@]}" chmod 600 "$BOOTSTRAP_SWAP_FILE"
  if ! "${SUDO[@]}" mkswap "$BOOTSTRAP_SWAP_FILE" >/dev/null; then
    info "temporary swap setup failed; continuing with reduced Go build parallelism"
    "${SUDO[@]}" rm -f "$BOOTSTRAP_SWAP_FILE" >/dev/null 2>&1 || true
    return
  fi
  if ! "${SUDO[@]}" swapon "$BOOTSTRAP_SWAP_FILE"; then
    info "temporary swap activation failed; continuing with reduced Go build parallelism"
    "${SUDO[@]}" rm -f "$BOOTSTRAP_SWAP_FILE" >/dev/null 2>&1 || true
    return
  fi
  BOOTSTRAP_SWAP_ACTIVE="1"
  trap 'stop_progress_server; cleanup_bootstrap_swap' EXIT
}

cleanup_bootstrap_swap() {
  if [[ "$BOOTSTRAP_SWAP_ACTIVE" == "1" ]]; then
    "${SUDO[@]}" swapoff "$BOOTSTRAP_SWAP_FILE" >/dev/null 2>&1 || true
    "${SUDO[@]}" rm -f "$BOOTSTRAP_SWAP_FILE" >/dev/null 2>&1 || true
    BOOTSTRAP_SWAP_ACTIVE="0"
  fi
}

checkout_repo() {
  if [[ -f go.mod ]] && grep -q '^module github.com/E8A281E6ACA2/perfassess$' go.mod; then
    WORK_DIR="$(pwd)"
    success "using current source directory: $WORK_DIR"
    return
  fi

  if [[ -d "$WORK_DIR/.git" ]]; then
    info "updating existing source directory: $WORK_DIR"
    git -C "$WORK_DIR" fetch origin main
    git -C "$WORK_DIR" checkout main
    git -C "$WORK_DIR" pull --ff-only origin main
  elif [[ -e "$WORK_DIR" ]]; then
    fail "$WORK_DIR exists but is not a git repository"
  else
    info "cloning source to $WORK_DIR"
    git clone "$REPO_URL" "$WORK_DIR"
  fi

  cd "$WORK_DIR"
}

clean_outputs() {
  local source_dir="$WORK_DIR"
  if [[ -f go.mod ]] && grep -q '^module github.com/E8A281E6ACA2/perfassess$' go.mod; then
    source_dir="$(pwd)"
  fi

  info "removing build and auto-test output"
  rm -rf "$source_dir/build" "$source_dir/coverage.out" "$source_dir/coverage.html" "$OUTPUT_DIR"
  success "cleaned build output and $OUTPUT_DIR"

  if [[ "$ACTION" == "clean-all" ]]; then
    [[ -d "$source_dir/.git" ]] || fail "Refusing to remove non-git directory: $source_dir"
    info "removing source directory: $source_dir"
    cd /
    rm -rf "$source_dir"
    success "removed $source_dir"
  fi
}

start_progress_server() {
  [[ "$START_WEB" == "1" ]] || return

  mkdir -p "$OUTPUT_DIR"
  info "starting realtime web progress on port $WEB_PORT"
  echo "Listening on all interfaces: http://0.0.0.0:$WEB_PORT"
  echo "Open http://<server-public-ip>:$WEB_PORT in your browser, or use SSH port forwarding."
  if public_ip="$(detect_public_ip)"; [[ -n "$public_ip" ]]; then
    echo "Detected public IP: http://$(format_url_host "$public_ip"):$WEB_PORT"
  fi
  echo "The page will update during the benchmark and expose generated report files."

  python3 "$WORK_DIR/scripts/perfassess-progress-server.py" \
    --dir "$OUTPUT_DIR" \
    --progress-file "$OUTPUT_DIR/progress.json" \
    --port "$WEB_PORT" &
  PROGRESS_SERVER_PID="$!"

  trap 'stop_progress_server; cleanup_bootstrap_swap' EXIT
  trap 'stop_progress_server; cleanup_bootstrap_swap; exit 130' INT TERM
  sleep 1
  if ! kill -0 "$PROGRESS_SERVER_PID" >/dev/null 2>&1; then
    fail "Realtime web progress failed to start. Check whether port $WEB_PORT is already in use."
  fi
}

detect_public_ip() {
  local ip
  ip="$(curl -fsS --max-time 3 https://api.ipify.org 2>/dev/null || true)"
  if [[ "$ip" =~ ^[0-9a-fA-F:.]+$ ]]; then
    echo "$ip"
  fi
}

format_url_host() {
  local host="$1"
  if [[ "$host" == *:* && "$host" != \[*\] ]]; then
    printf '[%s]' "$host"
  else
    printf '%s' "$host"
  fi
}

stop_progress_server() {
  if [[ -n "$PROGRESS_SERVER_PID" ]] && kill -0 "$PROGRESS_SERVER_PID" >/dev/null 2>&1; then
    kill "$PROGRESS_SERVER_PID" >/dev/null 2>&1 || true
    wait "$PROGRESS_SERVER_PID" >/dev/null 2>&1 || true
  fi
}

run_all() {
  install_packages
  install_go
  configure_low_memory_mode
  checkout_repo

  info "downloading Go modules"
  go mod download

  if [[ "$RUN_TESTS" == "1" ]]; then
    info "running unit tests"
    go test ./...
  fi

  start_progress_server

  info "running full non-interactive benchmark and acceptance"
  PERFASSESS_PROGRESS_FILE="$OUTPUT_DIR/progress.json" scripts/perfassess-auto.sh

  echo ""
  success "bootstrap completed"
  echo "Source: $WORK_DIR"
  echo "Binary: $WORK_DIR/build/perfassess"
  echo "Summary: $OUTPUT_DIR/summary.md"
  echo ""
  cat "$OUTPUT_DIR/summary.md"

  if [[ "$START_WEB" == "1" ]]; then
    echo ""
    info "realtime web progress remains available until this script exits"
    echo "Listening on all interfaces: http://0.0.0.0:$WEB_PORT"
    echo "Open http://<server-public-ip>:$WEB_PORT in your browser, or use SSH port forwarding."
    if public_ip="$(detect_public_ip)"; [[ -n "$public_ip" ]]; then
      echo "Detected public IP: http://$(format_url_host "$public_ip"):$WEB_PORT"
    fi
    echo "Press Ctrl+C to stop the progress server."
    wait "$PROGRESS_SERVER_PID"
  fi
}

case "$ACTION" in
  run)
    run_all
    ;;
  clean|clean-all)
    clean_outputs
    ;;
esac
