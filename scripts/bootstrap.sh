#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${PERFASSESS_REPO_URL:-https://github.com/E8A281E6ACA2/perfassess.git}"
WORK_DIR="${PERFASSESS_BOOTSTRAP_DIR:-$HOME/perfassess}"
GO_VERSION="${PERFASSESS_GO_VERSION:-1.25.3}"
OUTPUT_DIR="${PERFASSESS_AUTO_DIR:-/tmp/perfassess-auto}"
RUN_TESTS="${PERFASSESS_BOOTSTRAP_TESTS:-1}"
ACTION="run"

usage() {
  cat <<EOF
Usage: scripts/bootstrap.sh [options]

Options:
  --dir PATH       Source directory to clone or use. Default: $WORK_DIR
  --go VERSION    Go version to install when missing or too old. Default: $GO_VERSION
  --clean         Remove build output and auto-test output.
  --clean-all     Remove build output, auto-test output, and the cloned source directory.
  -h, --help      Show this help.

Environment:
  PERFASSESS_BOOTSTRAP_TESTS=0   Skip go test ./...
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

run_all() {
  install_packages
  install_go
  checkout_repo

  info "downloading Go modules"
  go mod download

  if [[ "$RUN_TESTS" == "1" ]]; then
    info "running unit tests"
    go test ./...
  fi

  info "running full non-interactive benchmark and acceptance"
  scripts/perfassess-auto.sh

  echo ""
  success "bootstrap completed"
  echo "Source: $WORK_DIR"
  echo "Binary: $WORK_DIR/build/perfassess"
  echo "Summary: $OUTPUT_DIR/summary.md"
  echo ""
  cat "$OUTPUT_DIR/summary.md"
}

case "$ACTION" in
  run)
    run_all
    ;;
  clean|clean-all)
    clean_outputs
    ;;
esac
