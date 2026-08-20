#!/usr/bin/env bash
#
# Drive real Terraform against the mock API.
#
#   ./run.sh                 full cycle: every stack under every quirk profile
#   ./run.sh test <stack>    one stack under every profile
#   ./run.sh up              start the mock and leave it running
#   ./run.sh down            stop the mock
#   ./run.sh apply <stack>   apply one stack against a running mock, keep it
#   ./run.sh destroy <stack> destroy one stack
#   ./run.sh clean           remove terraform state and the mock's logs
#
# The mock keeps everything in memory, so its state lives and dies with the
# process. That is fine for a full cycle, which starts it, applies, and destroys
# in one go. For interactive work use `up`, then `apply`/`destroy` as often as
# you like -- just do not restart the mock while a terraform state file still
# refers to objects inside it, or the next plan will show everything as deleted.
# `./run.sh clean` resets both sides together.

set -euo pipefail

QA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$QA_DIR/.." && pwd)"
SRC_DIR="$REPO_ROOT/src"
MOCK_DIR="$REPO_ROOT/mockapi"

MOCK_HOST="${MOCK_HOST:-127.0.0.1}"
MOCK_PORT="${MOCK_PORT:-8080}"
MOCK_URL="http://${MOCK_HOST}:${MOCK_PORT}"

BIN_DIR="$QA_DIR/.bin"
PID_FILE="$QA_DIR/.mock.pid"
LOG_FILE="$QA_DIR/.mock.log"
TFRC="$QA_DIR/.dev.tfrc"

STACKS=(tags core)

# Quirk profiles. A correct provider must survive every one of them: the docs
# do not say which the live API implements, so passing under only the default
# proves nothing.
#
#   strict     the conservative reading of every ambiguous behaviour
#   preserve   updates that omit a field leave the stored value alone
#   normalise  the API folds category names to lower case
PROFILES=(strict preserve normalise)

# Stack/profile pairs where the apply is expected to stop with a provider error,
# given as stack:profile:message-fragment.
#
# Under `normalise` the server rewrites tag category names, and `name` is a
# Required attribute. Terraform has no way to reconcile a value the user wrote
# with a different one the server insists on: whichever side state records, the
# other disagrees forever. The provider's contract is to detect that and fail
# loudly rather than leave a plan that proposes the same change on every run, so
# here a *clean* apply would be the bug.
EXPECT_APPLY_ERROR=("tags:normalise:Tenable.io Stored a Different Value")

expected_error() {
  local prefix="$1:$2:" entry
  for entry in "${EXPECT_APPLY_ERROR[@]}"; do
    if [[ "$entry" == "$prefix"* ]]; then printf '%s' "${entry#"$prefix"}"; return 0; fi
  done
  return 1
}

profile_env() {
  case "$1" in
    strict)    echo "" ;;
    preserve)  echo "MOCK_OMITTED_DESCRIPTION=preserves MOCK_OMITTED_FILTERS=preserves" ;;
    normalise) echo "MOCK_LOWERCASE_CATEGORY_NAMES=1" ;;
    *) echo "unknown profile: $1" >&2; exit 2 ;;
  esac
}

# Terraform's own logging is set to INFO in the devcontainer, which drowns
# everything here.
export TF_LOG=
export TF_IN_AUTOMATION=1
export TF_CLI_CONFIG_FILE="$TFRC"
export TENABLEIO_BASE_URL="$MOCK_URL"
export TENABLEIO_ACCESS_KEY="${TENABLEIO_ACCESS_KEY:-qa-access-key}"
export TENABLEIO_SECRET_KEY="${TENABLEIO_SECRET_KEY:-qa-secret-key}"

RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; BOLD=$'\033[1m'; OFF=$'\033[0m'
info() { printf '%s==>%s %s\n' "$BOLD" "$OFF" "$*"; }
pass() { printf '%s  ok%s %s\n' "$GREEN" "$OFF" "$*"; }
warn() { printf '%s  ..%s %s\n' "$YELLOW" "$OFF" "$*"; }
fail() { printf '%s FAIL%s %s\n' "$RED" "$OFF" "$*" >&2; }

# --- provider --------------------------------------------------------------

build_provider() {
  info "building provider"
  mkdir -p "$BIN_DIR"
  (cd "$SRC_DIR" && go build -o "$BIN_DIR/terraform-provider-tenableio")

  # dev_overrides points Terraform straight at the freshly built binary, so
  # there is no registry download and no `terraform init` step at all.
  cat > "$TFRC" <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/lamda-systems/tenableio" = "$BIN_DIR"
  }
  direct {}
}
EOF
}

# --- mock lifecycle --------------------------------------------------------

mock_running() {
  curl -sf "$MOCK_URL/__mock/health" >/dev/null 2>&1
}

venv_python() {
  if [[ ! -x "$MOCK_DIR/.venv/bin/python" ]]; then
    info "creating the mock's virtualenv"
    (cd "$MOCK_DIR" && python3 -m venv .venv && .venv/bin/pip install -q -r requirements.txt)
  fi
  echo "$MOCK_DIR/.venv/bin/python"
}

# Anything listening on the mock's port, however it got there.
port_pids() {
  ss -lptnH "sport = :$MOCK_PORT" 2>/dev/null | grep -oP 'pid=\K[0-9]+' | sort -u || true
}

start_mock() {
  local env_vars="$1"
  stop_mock

  local py; py="$(venv_python)"
  info "starting mock${env_vars:+ with $env_vars}"

  # The subshell is backgrounded as a unit and `exec` replaces it with the
  # interpreter, so $! really is the server's pid. Writing it any other way
  # ("cmd && cmd & echo $!") records the wrapper instead, and stop_mock then
  # leaves the real process running -- which silently leaks state between runs.
  # shellcheck disable=SC2086 # env_vars is a deliberate list of assignments
  (
    cd "$MOCK_DIR" || exit 1
    exec env $env_vars "$py" -m tenableio_mock \
      --host "$MOCK_HOST" --port "$MOCK_PORT" --log-level warning
  ) >"$LOG_FILE" 2>&1 &
  echo $! > "$PID_FILE"

  local up=0
  for _ in $(seq 1 60); do
    if mock_running; then up=1; break; fi
    sleep 0.25
  done
  if (( up == 0 )); then
    fail "mock did not become healthy; see $LOG_FILE"
    tail -20 "$LOG_FILE" >&2 || true
    return 1
  fi

  # Guard against reaching a server that outlived a previous run: if the port
  # was already taken, uvicorn would have exited and health would be answered by
  # the stale process, whose quirks and contents are both wrong.
  verify_quirks "$env_vars" || return 1
  curl -sf -X POST "$MOCK_URL/__mock/reset" >/dev/null
  pass "mock up at $MOCK_URL"
}

# Confirm the running server actually has the quirks we asked for.
verify_quirks() {
  local env_vars="$1" settings
  settings="$(curl -sf "$MOCK_URL/__mock/settings")" || {
    fail "could not read $MOCK_URL/__mock/settings"; return 1; }

  local want_desc=clears want_filters=clears want_lower=False
  [[ "$env_vars" == *"MOCK_OMITTED_DESCRIPTION=preserves"* ]] && want_desc=preserves
  [[ "$env_vars" == *"MOCK_OMITTED_FILTERS=preserves"* ]]     && want_filters=preserves
  [[ "$env_vars" == *"MOCK_LOWERCASE_CATEGORY_NAMES=1"* ]]    && want_lower=True

  local got
  got="$(printf '%s' "$settings" | python3 -c '
import json, sys
q = json.load(sys.stdin)["quirks"]
print(q["on_omitted_description"], q["on_omitted_filters"], q["lowercase_category_names"])')"

  if [[ "$got" != "$want_desc $want_filters $want_lower" ]]; then
    fail "mock quirks are '$got', expected '$want_desc $want_filters $want_lower'"
    fail "a server from an earlier run is probably still on port $MOCK_PORT"
    return 1
  fi
}

stop_mock() {
  local pids=()
  if [[ -f "$PID_FILE" ]]; then pids+=("$(cat "$PID_FILE")"); fi
  while read -r pid; do [[ -n "$pid" ]] && pids+=("$pid"); done < <(port_pids)
  rm -f "$PID_FILE"

  local pid
  for pid in "${pids[@]:-}"; do
    [[ -n "$pid" ]] || continue
    kill "$pid" 2>/dev/null || true
  done

  for _ in $(seq 1 40); do mock_running || return 0; sleep 0.25; done

  # Still answering: escalate rather than let a stale server poison the run.
  while read -r pid; do [[ -n "$pid" ]] && kill -9 "$pid" 2>/dev/null; done < <(port_pids)
  sleep 0.5
}

require_mock() {
  mock_running || { fail "no mock at $MOCK_URL -- run './run.sh up' first"; exit 1; }
}

# --- terraform -------------------------------------------------------------

tf() {
  local stack="$1"; shift
  terraform -chdir="$QA_DIR/stacks/$stack" "$@"
}

# A stack is healthy when it applies, and then plans clean. The second plan is
# the assertion that matters: an empty diff is what proves the provider wrote
# back exactly what it planned, which is the failure mode this whole harness
# exists to catch.
test_stack() {
  local stack="$1" profile="$2"
  info "stack '$stack' under profile '$profile'"

  local out="$QA_DIR/.last-run.log"

  # Capture on the first attempt. Re-running a failed apply just to show its
  # output would report "already exists" for whatever the first pass created,
  # burying the error that actually mattered.
  local want_error=""
  want_error="$(expected_error "$stack" "$profile")" || true

  if ! tf "$stack" apply -auto-approve -input=false -no-color >"$out" 2>&1; then
    if [[ -n "$want_error" ]] && grep -qF "$want_error" "$out"; then
      warn "apply stopped with the expected guard: \"$want_error\""
      # Best effort tidy-up; the mock is replaced for the next profile anyway.
      tf "$stack" destroy -auto-approve -input=false -no-color >/dev/null 2>&1 || true
      return 0
    fi
    fail "$stack/$profile: apply failed"
    tail -40 "$out" >&2
    return 1
  fi

  if [[ -n "$want_error" ]]; then
    fail "$stack/$profile: apply succeeded, but the guard \"$want_error\" was expected to fire"
    return 1
  fi
  pass "applied"

  # -detailed-exitcode: 0 = no changes, 1 = error, 2 = changes pending.
  local code=0
  tf "$stack" plan -detailed-exitcode -input=false -no-color >"$out" 2>&1 || code=$?
  case "$code" in
    0) pass "re-plan is empty (idempotent)" ;;
    2) fail "$stack/$profile: re-plan is NOT empty -- the provider did not persist what it planned"
       grep -E '^\s+[~+-]|will be|must be replaced' "$out" | head -30 >&2 || true
       return 1 ;;
    *) fail "$stack/$profile: re-plan errored"; tail -40 "$out" >&2; return 1 ;;
  esac

  # Re-apply on top of an unchanged state: catches an Update path that breaks
  # even when nothing changed.
  if ! tf "$stack" apply -auto-approve -input=false -no-color >"$out" 2>&1; then
    fail "$stack/$profile: second apply failed"
    tail -40 "$out" >&2
    return 1
  fi
  pass "second apply is a no-op"

  check_outputs "$stack" || return 1

  if ! tf "$stack" destroy -auto-approve -input=false -no-color >"$out" 2>&1; then
    fail "$stack/$profile: destroy failed"
    tail -40 "$out" >&2
    return 1
  fi
  pass "destroyed"
}

# Outputs double as assertions about values that must survive a round trip.
check_outputs() {
  local stack="$1" out
  out="$(tf "$stack" output -json)" || { fail "could not read outputs"; return 1; }

  local failed=0
  expect() { # name, expected
    local got; got="$(echo "$out" | python3 -c \
      "import json,sys; print(json.load(sys.stdin).get('$1',{}).get('value'))")"
    if [[ "$got" != "$2" ]]; then
      fail "output $1 = '$got', want '$2'"; failed=1
    fi
  }

  case "$stack" in
    tags)
      expect category_count 2
      expect value_count 5
      expect dynamic_value_count 2
      expect location_description ""
      expect staging_description ""
      ;;
    core)
      expect folder_count 3   # two seeded system folders plus ours
      expect network_count 3  # the seeded default plus two
      expect agent_group_count 1
      expect exclusion_count 2
      expect policy_count 2
      expect scan_count 2
      expect scanner_count 2
      expect asset_count 2
      expect minimal_network_description ""
      expect minimal_network_ttl 180
      expect minimal_policy_visibility private
      expect minimal_policy_description ""
      ;;
  esac
  (( failed == 0 )) && pass "outputs match"
  return $failed
}

# Configurations under guards/ are deliberately invalid: each one must be
# rejected by `terraform validate`. They pin the guards that stop a bad value
# before it ever reaches the API, where the resulting error would be far less
# legible.
check_guards() {
  local failed=0 dir name out="$QA_DIR/.last-run.log"
  for dir in "$QA_DIR"/guards/*/; do
    [[ -d "$dir" ]] || continue
    name="$(basename "$dir")"
    if terraform -chdir="$dir" validate -no-color >"$out" 2>&1; then
      fail "guard '$name': terraform validate accepted a configuration it should reject"
      failed=1
    else
      pass "guard '$name' rejected as expected"
    fi
  done
  return $failed
}

clean_state() {
  info "clearing terraform state"
  for stack in "${STACKS[@]}"; do
    rm -rf "$QA_DIR/stacks/$stack/.terraform" \
           "$QA_DIR/stacks/$stack"/terraform.tfstate*
  done
}

# --- commands --------------------------------------------------------------

cmd_full() {
  local only="${1:-}"
  local stacks=("${STACKS[@]}")
  [[ -n "$only" ]] && stacks=("$only")

  build_provider

  local failures=0
  printf '\n%s──── guards ────%s\n' "$BOLD" "$OFF"
  check_guards || failures=$((failures + 1))

  for profile in "${PROFILES[@]}"; do
    printf '\n%s──── profile: %s ────%s\n' "$BOLD" "$profile" "$OFF"
    # A fresh mock per profile: quirks are read at startup, and every stack
    # should begin from an empty container.
    start_mock "$(profile_env "$profile")"
    clean_state
    for stack in "${stacks[@]}"; do
      test_stack "$stack" "$profile" || failures=$((failures + 1))
    done
  done

  stop_mock
  clean_state

  printf '\n'
  if (( failures == 0 )); then
    pass "all stacks passed under all profiles"
  else
    fail "$failures stack/profile combination(s) failed"
  fi
  return $((failures > 0))
}

main() {
  case "${1:-full}" in
    full)    cmd_full ;;
    test)    cmd_full "${2:?usage: run.sh test <stack>}" ;;
    up)      build_provider; start_mock "$(profile_env "${2:-strict}")" ;;
    down)    stop_mock; pass "mock stopped" ;;
    apply)   require_mock; build_provider
             tf "${2:?usage: run.sh apply <stack>}" apply -auto-approve -input=false ;;
    destroy) require_mock
             tf "${2:?usage: run.sh destroy <stack>}" destroy -auto-approve -input=false ;;
    clean)   stop_mock; clean_state; rm -f "$TFRC" "$LOG_FILE" "$QA_DIR/.last-run.log"; rm -rf "$BIN_DIR"; pass "cleaned" ;;
    *)       sed -n '3,20p' "${BASH_SOURCE[0]}"; exit 2 ;;
  esac
}

main "$@"
