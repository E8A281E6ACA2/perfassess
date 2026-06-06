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
WEB_TTL_SECONDS="${PERFASSESS_WEB_TTL_SECONDS:-0}"
AUTO_PROFILE="${PERFASSESS_AUTO_PROFILE:-auto}"
QUALITY_PROFILE="${PERFASSESS_QUALITY_PROFILE:-auto}"
NETWORK_PROFILE="${PERFASSESS_NETWORK_PROFILE:-auto}"
BOOTSTRAP_INTERACTIVE="${PERFASSESS_BOOTSTRAP_INTERACTIVE:-auto}"
ACTION="run"
PROGRESS_SERVER_PID=""
BOOTSTRAP_CLEANUP_AFTER_RUN="${PERFASSESS_BOOTSTRAP_CLEANUP_AFTER_RUN:-0}"
BOOTSTRAP_DESTROY_AFTER_WEB="${PERFASSESS_BOOTSTRAP_DESTROY_AFTER_WEB:-0}"
LOW_MEMORY_THRESHOLD_MB="${PERFASSESS_LOW_MEMORY_THRESHOLD_MB:-768}"
BOOTSTRAP_SWAP_MODE="${PERFASSESS_BOOTSTRAP_SWAP:-auto}"
BOOTSTRAP_SWAP_SIZE_MB="${PERFASSESS_BOOTSTRAP_SWAP_SIZE_MB:-1024}"
BOOTSTRAP_SWAP_FILE="${PERFASSESS_BOOTSTRAP_SWAP_FILE:-$OUTPUT_DIR/bootstrap.swap}"
BOOTSTRAP_SWAP_ACTIVE="0"
IPERF3_SERVER="${PERFASSESS_IPERF3_SERVER:-}"
IPERF3_SERVERS="${PERFASSESS_IPERF3_SERVERS:-}"
IPERF3_SERVER_FILE="${PERFASSESS_IPERF3_SERVER_FILE:-}"

usage() {
  cat <<EOF
Usage: scripts/bootstrap.sh [options]

Options:
  --dir PATH       Source directory to clone or use. Default: $WORK_DIR
  --go VERSION    Go version to install when missing or too old. Default: $GO_VERSION
  --web           Start the realtime Material Design progress page.
  --port PORT     Web report port when --web is used. Default: $WEB_PORT
  --web-ttl SEC   Stop the Web page SEC seconds after benchmark completion. Default: $WEB_TTL_SECONDS.
  --profile NAME  Auto benchmark profile: auto, basic, standard, full. Default: $AUTO_PROFILE
  --quality NAME  Benchmark backend quality: auto, builtin, mainstream. Default: $QUALITY_PROFILE
  --network-profile NAME
                   Network test profile: auto, quick, standard, full. Default: $NETWORK_PROFILE
  --cleanup-after-run
                   Remove build artifacts after benchmark. Reports remain in $OUTPUT_DIR.
  --destroy-after-web
                   After Web viewing ends, remove source, build artifacts, and $OUTPUT_DIR.
  --clean         Remove build output and auto-test output.
  --clean-all     Remove build output, auto-test output, and the cloned source directory.
  -h, --help      Show this help.

Environment:
  PERFASSESS_BOOTSTRAP_TESTS=0   Skip go test ./...
  PERFASSESS_BOOTSTRAP_WEB=1     Start the realtime Material Design progress page.
  PERFASSESS_WEB_TTL_SECONDS=600 Stop Web server 10 minutes after completion.
  PERFASSESS_BOOTSTRAP_SWAP=auto Create temporary swap on low-memory Linux hosts. Set 0 to disable.
  PERFASSESS_BOOTSTRAP_CLEANUP_AFTER_RUN=1  Remove build artifacts after benchmark.
  PERFASSESS_BOOTSTRAP_DESTROY_AFTER_WEB=1  Remove source and reports after Web viewing ends.
  PERFASSESS_LOW_MEMORY_THRESHOLD_MB=768  Memory threshold for low-memory mode.
  PERFASSESS_WEB_PORT=9090       Web report port.
  PERFASSESS_AUTO_PROFILE=standard  Auto profile: auto, basic, standard, or full.
  PERFASSESS_QUALITY_PROFILE=mainstream  Backend quality: auto, builtin, or mainstream.
  PERFASSESS_NETWORK_PROFILE=standard  Network profile: auto, quick, standard, or full.
  PERFASSESS_STREAMING_PROFILE=full  Streaming profile: auto, quick, standard, or full.
  PERFASSESS_AUTO_STRESS=1          Include stress test when profile is standard.
  PERFASSESS_AUTO_OPTIONAL=auto  Let acceptance run optional checks when dependencies exist.
  PERFASSESS_IPERF3_SERVER=1.2.3.4:5201  Use iperf3 network backend with this server.
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
    --web-ttl)
      [[ $# -ge 2 ]] || fail "--web-ttl requires seconds"
      WEB_TTL_SECONDS="$2"
      shift 2
      ;;
    --profile)
      [[ $# -ge 2 ]] || fail "--profile requires a value"
      AUTO_PROFILE="$2"
      shift 2
      ;;
    --quality)
      [[ $# -ge 2 ]] || fail "--quality requires a value"
      QUALITY_PROFILE="$2"
      shift 2
      ;;
    --network-profile)
      [[ $# -ge 2 ]] || fail "--network-profile requires a value"
      NETWORK_PROFILE="$2"
      shift 2
      ;;
    --cleanup-after-run)
      BOOTSTRAP_CLEANUP_AFTER_RUN="1"
      shift
      ;;
    --destroy-after-web)
      START_WEB="1"
      BOOTSTRAP_DESTROY_AFTER_WEB="1"
      shift
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
  elif command -v brew >/dev/null 2>&1; then
    brew install "${packages[@]}"
  else
    fail "No supported package manager found. Install these first: ${packages[*]}"
  fi
}

install_named_packages() {
  local packages=("$@")
  [[ ${#packages[@]} -gt 0 ]] || return 0

  need_sudo
  info "installing profile dependencies: ${packages[*]}"

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
  elif command -v brew >/dev/null 2>&1; then
    brew install "${packages[@]}"
  else
    fail "No supported package manager found. Install these first: ${packages[*]}"
  fi
}

install_profile_packages() {
  local packages=()

  case "$AUTO_PROFILE" in
    standard|full)
      if [[ "$(uname -s)" == "Linux" ]] && ! command -v traceroute >/dev/null 2>&1; then
        packages+=(traceroute)
      fi
      ;;
  esac

  if [[ "$QUALITY_PROFILE" == "mainstream" ]]; then
    command -v sysbench >/dev/null 2>&1 || packages+=(sysbench)
    command -v fio >/dev/null 2>&1 || packages+=(fio)
    if [[ -n "$IPERF3_SERVER" || -n "$IPERF3_SERVERS" || -n "$IPERF3_SERVER_FILE" ]]; then
      command -v iperf3 >/dev/null 2>&1 || packages+=(iperf3)
    fi
  fi

  if [[ ${#packages[@]} -eq 0 ]]; then
    success "profile dependencies are present"
    return
  fi

  install_named_packages "${packages[@]}"
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

cpu_threads() {
  if command -v getconf >/dev/null 2>&1; then
    getconf _NPROCESSORS_ONLN 2>/dev/null || echo 0
  elif [[ -r /proc/cpuinfo ]]; then
    awk '/^processor[[:space:]]*:/ { count++ } END { print count + 0 }' /proc/cpuinfo
  else
    echo 0
  fi
}

disk_available_mb() {
  df -Pm . 2>/dev/null | awk 'NR == 2 { print $4 }'
}

can_prompt() {
  [[ "$BOOTSTRAP_INTERACTIVE" != "0" && "$BOOTSTRAP_INTERACTIVE" != "false" && -t 1 && -r /dev/tty ]]
}

read_tty() {
  local var_name="$1"
  IFS= read -r "$var_name" </dev/tty || printf -v "$var_name" ''
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

recommended_profile() {
  local mem_mb="$1"
  local disk_mb="$2"
  local _cpu_count="$3"

  if [[ "$mem_mb" -gt 0 && "$mem_mb" -lt 1024 ]]; then
    echo "basic"
  elif [[ "$disk_mb" -gt 0 && "$disk_mb" -lt 2048 ]]; then
    echo "basic"
  else
    echo "standard"
  fi
}

recommended_quality_profile() {
  local mem_mb="$1"
  local disk_mb="$2"
  local cpu_count="$3"

  if [[ "$mem_mb" -gt 0 && "$mem_mb" -lt 2048 ]]; then
    echo "builtin"
  elif [[ "$disk_mb" -gt 0 && "$disk_mb" -lt 4096 ]]; then
    echo "builtin"
  elif [[ "$cpu_count" -gt 0 && "$cpu_count" -lt 2 ]]; then
    echo "builtin"
  else
    echo "mainstream"
  fi
}

recommended_network_profile() {
  case "$1" in
    full)
      echo "full"
      ;;
    standard)
      echo "standard"
      ;;
    *)
      echo "quick"
      ;;
  esac
}

profile_description() {
  case "$1" in
    basic)
      echo "基础测评：CPU、内存、磁盘、网络，最稳，适合低配或 512MB 机器。"
      ;;
    full)
      echo "全量测评：标准报告 + 压力测试，耗时更长且会明显占用 CPU、内存和磁盘。"
      ;;
    standard)
      echo "标准报告：基础测评 + 路由、流媒体、AI、IP 质量、安全体检，不跑压力测试。"
      ;;
  esac
}

quality_description() {
  case "$1" in
    builtin)
      echo "内置后端：依赖少、最稳，适合快速验收；报告置信度通常为 medium。"
      ;;
    mainstream)
      echo "主流后端：安装并使用 sysbench/fio；有 iperf3 服务端时使用真实上传，报告更可比。"
      ;;
  esac
}

network_profile_description() {
  case "$1" in
    quick)
      echo "轻量网络：少量公共目标，耗时短，只作出站路径参考。"
      ;;
    full)
      echo "全量网络：更多全球和国内方向目标，耗时更长；国内方向仅作参考，不等同真实回程。"
      ;;
    standard)
      echo "标准网络：公共目标 + 国内三网方向参考；不等同真实回程。"
      ;;
  esac
}

choose_auto_profile() {
  local fixed_profile="0"

  case "$AUTO_PROFILE" in
    stress)
      AUTO_PROFILE="full"
      info "using requested auto profile: $AUTO_PROFILE"
      fixed_profile="1"
      ;;
    basic|standard|full)
      info "using requested auto profile: $AUTO_PROFILE"
      fixed_profile="1"
      ;;
    auto) ;;
    *)
      fail "--profile must be auto, basic, standard, or full"
      ;;
  esac

  case "$QUALITY_PROFILE" in
    auto|builtin|mainstream) ;;
    *)
      fail "--quality must be auto, builtin, or mainstream"
      ;;
  esac

  case "$NETWORK_PROFILE" in
    auto|quick|standard|full) ;;
    *)
      fail "--network-profile must be auto, quick, standard, or full"
      ;;
  esac

  local mem_mb swap_mb disk_mb cpu_count recommended recommended_quality recommended_network selected selected_quality selected_network
  mem_mb="$(memory_total_mb)"
  swap_mb="$(swap_total_mb)"
  disk_mb="$(disk_available_mb)"
  disk_mb="${disk_mb:-0}"
  cpu_count="$(cpu_threads)"
  cpu_count="${cpu_count:-0}"
  recommended="$(recommended_profile "$mem_mb" "$disk_mb" "$cpu_count")"
  recommended_quality="$(recommended_quality_profile "$mem_mb" "$disk_mb" "$cpu_count")"
  if [[ "$fixed_profile" == "1" ]]; then
    recommended_network="$(recommended_network_profile "$AUTO_PROFILE")"
  else
    recommended_network="$(recommended_network_profile "$recommended")"
  fi

  echo ""
  info "machine probe before benchmark"
  echo "CPU threads: ${cpu_count}"
  echo "Memory: ${mem_mb}MB RAM, ${swap_mb}MB swap"
  echo "Available disk: ${disk_mb}MB"
  if [[ "$fixed_profile" == "1" ]]; then
    echo "Benchmark profile: ${AUTO_PROFILE} - $(profile_description "$AUTO_PROFILE")"
  else
    echo "Recommended profile: ${recommended} - $(profile_description "$recommended")"
  fi
  echo "Recommended quality: ${recommended_quality} - $(quality_description "$recommended_quality")"
  echo "Recommended network: ${recommended_network} - $(network_profile_description "$recommended_network")"

  if can_prompt; then
    if [[ "$fixed_profile" != "1" ]]; then
      echo ""
      echo "Choose benchmark profile:"
      echo "  1) basic    - $(profile_description basic)"
      echo "  2) standard - $(profile_description standard)"
      echo "  3) full     - $(profile_description full)"
      printf "Selection [recommended: %s]: " "$recommended"
      read_tty selected
      case "${selected:-$recommended}" in
        1|basic) AUTO_PROFILE="basic" ;;
        2|standard) AUTO_PROFILE="standard" ;;
        3|full|stress) AUTO_PROFILE="full" ;;
        *) AUTO_PROFILE="$recommended" ;;
      esac
    fi

    if [[ "$QUALITY_PROFILE" == "auto" ]]; then
      echo ""
      echo "Choose backend quality:"
      echo "  1) builtin    - $(quality_description builtin)"
      echo "  2) mainstream - $(quality_description mainstream)"
      printf "Selection [recommended: %s]: " "$recommended_quality"
      read_tty selected_quality
      case "${selected_quality:-$recommended_quality}" in
        1|builtin) QUALITY_PROFILE="builtin" ;;
        2|mainstream) QUALITY_PROFILE="mainstream" ;;
        *) QUALITY_PROFILE="$recommended_quality" ;;
      esac
    fi

    if [[ "$NETWORK_PROFILE" == "auto" ]]; then
      echo ""
      echo "Choose network profile:"
      echo "  1) quick    - $(network_profile_description quick)"
      echo "  2) standard - $(network_profile_description standard)"
      echo "  3) full     - $(network_profile_description full)"
      printf "Selection [recommended: %s]: " "$recommended_network"
      read_tty selected_network
      case "${selected_network:-$recommended_network}" in
        1|quick) NETWORK_PROFILE="quick" ;;
        2|standard) NETWORK_PROFILE="standard" ;;
        3|full) NETWORK_PROFILE="full" ;;
        *) NETWORK_PROFILE="$recommended_network" ;;
      esac
    fi
  else
    if [[ "$fixed_profile" != "1" ]]; then
      AUTO_PROFILE="$recommended"
    fi
    if [[ "$QUALITY_PROFILE" == "auto" ]]; then
      QUALITY_PROFILE="$recommended_quality"
    fi
    if [[ "$NETWORK_PROFILE" == "auto" ]]; then
      NETWORK_PROFILE="$recommended_network"
    fi
    info "non-interactive mode selected profile: $AUTO_PROFILE"
    info "non-interactive mode selected quality: $QUALITY_PROFILE"
    info "non-interactive mode selected network: $NETWORK_PROFILE"
  fi

  export PERFASSESS_AUTO_PROFILE="$AUTO_PROFILE"
  export PERFASSESS_QUALITY_PROFILE="$QUALITY_PROFILE"
  export PERFASSESS_NETWORK_PROFILE="$NETWORK_PROFILE"
  if [[ "$AUTO_PROFILE" == "full" ]]; then
    export PERFASSESS_AUTO_STRESS="1"
  fi
  success "auto profile selected: $AUTO_PROFILE"
  success "quality profile selected: $QUALITY_PROFILE"
  success "network profile selected: $NETWORK_PROFILE"
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

cleanup_after_run() {
  [[ "$BOOTSTRAP_CLEANUP_AFTER_RUN" == "1" || "$BOOTSTRAP_CLEANUP_AFTER_RUN" == "true" ]] || return 0

  info "cleaning build artifacts after benchmark"
  rm -rf "$WORK_DIR/build" "$WORK_DIR/coverage.out" "$WORK_DIR/coverage.html"
  success "build artifacts cleaned; reports remain in $OUTPUT_DIR"
}

destroy_after_web() {
  [[ "$BOOTSTRAP_DESTROY_AFTER_WEB" == "1" || "$BOOTSTRAP_DESTROY_AFTER_WEB" == "true" ]] || return 0

  [[ -d "$WORK_DIR/.git" ]] || fail "Refusing to remove non-git source directory: $WORK_DIR"
  info "destroying benchmark source and report output"
  cd /
  rm -rf "$WORK_DIR" "$OUTPUT_DIR"
  success "destroyed $WORK_DIR and $OUTPUT_DIR"
}

start_progress_server() {
  [[ "$START_WEB" == "1" ]] || return 0

  mkdir -p "$OUTPUT_DIR"
  local requested_port="$WEB_PORT"
  local candidate_port
  [[ "$requested_port" =~ ^[0-9]+$ ]] || fail "Web port must be a number: $requested_port"
  rm -f "$OUTPUT_DIR/progress-server.log"
  for candidate_port in $(seq "$requested_port" $((requested_port + 20))); do
    info "starting realtime web progress on port $candidate_port"
    python3 "$WORK_DIR/scripts/perfassess-progress-server.py" \
      --dir "$OUTPUT_DIR" \
      --progress-file "$OUTPUT_DIR/progress.json" \
      --port "$candidate_port" >>"$OUTPUT_DIR/progress-server.log" 2>&1 &
    PROGRESS_SERVER_PID="$!"
    sleep 1
    if kill -0 "$PROGRESS_SERVER_PID" >/dev/null 2>&1; then
      WEB_PORT="$candidate_port"
      if [[ "$WEB_PORT" != "$requested_port" ]]; then
        info "requested port $requested_port is unavailable; using $WEB_PORT instead"
      fi
      print_web_access
      echo "The page will update during the benchmark and expose generated report files."
      break
    fi
    wait "$PROGRESS_SERVER_PID" >/dev/null 2>&1 || true
    PROGRESS_SERVER_PID=""
  done

  trap 'stop_progress_server; cleanup_bootstrap_swap' EXIT
  trap 'stop_progress_server; cleanup_bootstrap_swap; exit 130' INT TERM
  if [[ -z "$PROGRESS_SERVER_PID" ]] || ! kill -0 "$PROGRESS_SERVER_PID" >/dev/null 2>&1; then
    fail "Realtime web progress failed to start. Tried ports $requested_port-$((requested_port + 20)). See $OUTPUT_DIR/progress-server.log"
  fi
}

print_web_access() {
  echo "Listening on all interfaces: http://0.0.0.0:$WEB_PORT"
  echo "Open http://<server-public-ip>:$WEB_PORT in your browser, or use SSH port forwarding."
  if public_ip="$(detect_public_ip)"; [[ -n "$public_ip" ]]; then
    echo "Detected public IP: http://$(format_url_host "$public_ip"):$WEB_PORT"
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

  choose_auto_profile
  install_profile_packages
  start_progress_server

  info "running selected benchmark profile and acceptance"
  PERFASSESS_PROGRESS_FILE="$OUTPUT_DIR/progress.json" scripts/perfassess-auto.sh

  echo ""
  success "bootstrap completed"
  echo "Source: $WORK_DIR"
  echo "Binary: $WORK_DIR/build/perfassess"
  echo "Console: $OUTPUT_DIR/console.txt"
  echo "Summary: $OUTPUT_DIR/summary.md"
  echo "Archive: $OUTPUT_DIR/perfassess-report.zip"
  echo ""
  if [[ -t 1 && -f "$OUTPUT_DIR/console.ansi" && -z "${NO_COLOR:-}" && "${PERFASSESS_NO_COLOR:-0}" != "1" ]]; then
    cat "$OUTPUT_DIR/console.ansi"
  elif [[ -f "$OUTPUT_DIR/console.txt" ]]; then
    cat "$OUTPUT_DIR/console.txt"
  else
    cat "$OUTPUT_DIR/summary.md"
  fi

  cleanup_after_run

  if [[ "$START_WEB" == "1" ]]; then
    echo ""
    info "realtime web progress remains available until this script exits"
    print_web_access
    if [[ "$WEB_TTL_SECONDS" =~ ^[0-9]+$ && "$WEB_TTL_SECONDS" -gt 0 ]]; then
      echo "Web page will stop automatically after ${WEB_TTL_SECONDS}s."
      sleep "$WEB_TTL_SECONDS"
      stop_progress_server
    else
      echo "Press Ctrl+C to stop the progress server."
      wait "$PROGRESS_SERVER_PID"
    fi
    destroy_after_web
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
