#!/bin/sh
# Install and verify one project-pinned Loaf release for an Amp Orb.
# POSIX sh only: this script intentionally has no source-checkout or language
# toolchain dependency.
set -eu

inherited_path=${PATH:-}
inherited_project_env=${LOAF_PROJECT_ENV:-}

fail() {
  printf '%s\n' "loaf-orb-bootstrap: $*" >&2
  exit 1
}

safe_pin_value() {
  safe_value=$1
  [ "${#safe_value}" -le 256 ] || return 1
  printf '%s\n' "$safe_value" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._:/+#-]{0,255}$' || return 1
  case "$safe_value" in
    *\\*|*@*|/*|~*|*..*|*://*) return 1 ;;
  esac
  safe_lower=$(printf '%s\n' "$safe_value" | tr '[:upper:]' '[:lower:]')
  case "$safe_lower" in
    *password*|*passwd*|*token*|*secret*|*credential*|*authorization*|*bearer*) return 1 ;;
  esac
}

valid_semver() (
  semver_value=$1
  [ "$semver_value" != 0.0.0 ] || return 1
  printf '%s\n' "$semver_value" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$' || return 1
  semver_without_build=${semver_value%%+*}
  semver_core=${semver_without_build%%-*}
  semver_major=${semver_core%%.*}
  semver_remainder=${semver_core#*.}
  semver_minor=${semver_remainder%%.*}
  semver_patch=${semver_remainder#*.}
  [ "${#semver_major}" -le 9 ] && [ "${#semver_minor}" -le 9 ] && [ "${#semver_patch}" -le 9 ] || return 1
  case "$semver_without_build" in
    *-*) semver_prerelease=${semver_without_build#*-} ;;
    *) return 0 ;;
  esac
  while :; do
    semver_identifier=${semver_prerelease%%.*}
    case "$semver_identifier" in
      *[!0-9]*) ;;
      0|[1-9]|[1-9][0-9]*) ;;
      *) return 1 ;;
    esac
    case "$semver_prerelease" in
      *.*) semver_prerelease=${semver_prerelease#*.} ;;
      *) break ;;
    esac
  done
)

agents_dir=$(CDPATH='' cd "$(dirname "$0")" && pwd)
project_root=$(CDPATH='' cd "$agents_dir/.." && pwd)
pin_path="$agents_dir/loaf-orb.pin"
pin_argument=.agents/loaf-orb.pin

[ -f "$pin_path" ] && [ ! -L "$pin_path" ] || fail "$pin_argument must be a non-symlink regular file"
[ "$(wc -l < "$pin_path")" -eq 8 ] || fail "pin must contain exactly eight newline-terminated lines"

schema=
version=
checksum_x64=
checksum_arm64=
project_id=
git_remote=
tracker_provider=
tracker_scope=
line_count=0

while IFS= read -r line || [ -n "$line" ]; do
  line_count=$((line_count + 1))
  case "$line_count" in
    1) expected_key=schema ;;
    2) expected_key=version ;;
    3) expected_key=archive.linux-x64.sha256 ;;
    4) expected_key=archive.linux-arm64.sha256 ;;
    5) expected_key=project.id ;;
    6) expected_key=git.remote ;;
    7) expected_key=tracker.provider ;;
    8) expected_key=tracker.scope ;;
    *) fail "pin must contain exactly eight lines" ;;
  esac
  case "$line" in
    "$expected_key"=*) ;;
    *) fail "pin line $line_count must define $expected_key" ;;
  esac
  case "$line" in
    schema=*) [ -z "$schema" ] || fail "duplicate pin key: schema"; schema=${line#schema=} ;;
    version=*) [ -z "$version" ] || fail "duplicate pin key: version"; version=${line#version=} ;;
    archive.linux-x64.sha256=*)
      [ -z "$checksum_x64" ] || fail "duplicate pin key: archive.linux-x64.sha256"
      checksum_x64=${line#archive.linux-x64.sha256=}
      ;;
    archive.linux-arm64.sha256=*)
      [ -z "$checksum_arm64" ] || fail "duplicate pin key: archive.linux-arm64.sha256"
      checksum_arm64=${line#archive.linux-arm64.sha256=}
      ;;
    project.id=*) [ -z "$project_id" ] || fail "duplicate pin key: project.id"; project_id=${line#project.id=} ;;
    git.remote=*) [ -z "$git_remote" ] || fail "duplicate pin key: git.remote"; git_remote=${line#git.remote=} ;;
    tracker.provider=*)
      [ -z "$tracker_provider" ] || fail "duplicate pin key: tracker.provider"
      tracker_provider=${line#tracker.provider=}
      ;;
    tracker.scope=*)
      [ -z "$tracker_scope" ] || fail "duplicate pin key: tracker.scope"
      tracker_scope=${line#tracker.scope=}
      ;;
    *=*) fail "unknown pin key: ${line%%=*}" ;;
    *) fail "malformed pin line $line_count" ;;
  esac
done < "$pin_path"

[ "$line_count" -eq 8 ] || fail "pin must contain exactly eight lines"
[ "$schema" = 1 ] || fail "pin schema must be 1"
valid_semver "$version" || fail "invalid release version"
printf '%s\n' "$checksum_x64" | grep -Eq '^[0-9a-f]{64}$' || fail "invalid linux-x64 checksum"
printf '%s\n' "$checksum_arm64" | grep -Eq '^[0-9a-f]{64}$' || fail "invalid linux-arm64 checksum"
safe_pin_value "$project_id" || fail "unsafe project ID"
printf '%s\n' "$project_id" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._:-]*$' || fail "invalid project ID"
safe_pin_value "$git_remote" || fail "unsafe git remote"
printf '%s\n' "$git_remote" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._-]{0,99}/[A-Za-z0-9][A-Za-z0-9._-]{0,99}$' || fail "git.remote must be an unauthenticated owner/repository name"
safe_pin_value "$tracker_provider" || fail "unsafe tracker provider"
printf '%s\n' "$tracker_provider" | grep -Eq '^[a-z][a-z0-9-]{0,63}$' || fail "invalid tracker provider"
safe_pin_value "$tracker_scope" || fail "unsafe tracker scope"

command -v git >/dev/null 2>&1 || fail "git is required"
(
  cd "$project_root"
  git cat-file -e "HEAD:$pin_argument" >/dev/null 2>&1 &&
    git diff --quiet HEAD -- "$pin_argument"
) || fail "$pin_argument must match committed bytes"

case "$(uname -s)" in
  Linux) ;;
  *) fail "Amp Orb bootstrap supports Linux only" ;;
esac

case "$(uname -m)" in
  x86_64|amd64)
    platform=linux-x64
    archive_sha256=$checksum_x64
    ;;
  aarch64|arm64)
    platform=linux-arm64
    archive_sha256=$checksum_arm64
    ;;
  *) fail "unsupported Linux architecture: $(uname -m)" ;;
esac

if [ -n "${LOAF_ORB_HOME:-}" ]; then
  orb_prefix=$LOAF_ORB_HOME
else
  if [ -n "${XDG_DATA_HOME:-}" ]; then
    orb_data_home=$XDG_DATA_HOME
  else
    [ -n "${HOME:-}" ] || fail "HOME or XDG_DATA_HOME is required"
    orb_data_home=$HOME/.local/share
  fi
  orb_prefix=$orb_data_home/loaf/orb/$project_id
fi
case "$orb_prefix" in
  /*) ;;
  *) fail "Orb-local user prefix must be an absolute path" ;;
esac
[ "$orb_prefix" != / ] || fail "refusing root as the Orb-local user prefix"

if [ -n "${LOAF_BIN_DIR:-}" ]; then
  loaf_user_bin_dir=$LOAF_BIN_DIR
else
  [ -n "${HOME:-}" ] || fail "HOME or LOAF_BIN_DIR is required"
  loaf_user_bin_dir=$HOME/.local/bin
fi
case "$loaf_user_bin_dir" in
  /*) ;;
  *) fail "LOAF_BIN_DIR must be an absolute path" ;;
esac
[ "$loaf_user_bin_dir" != / ] || fail "refusing root as LOAF_BIN_DIR"

release_dir=$orb_prefix/releases/$version
release_bin_dir=$release_dir/bin
loaf_bin=$release_bin_dir/loaf
manifest_path=$release_dir/loaf-release-manifest.json
receipt_path=$release_dir/.loaf-archive.sha256
platform_path=$release_dir/.platform
verified_archive_sha256=

resolve_path() {
  resolve_candidate=$1
  resolve_depth=0
  while [ -L "$resolve_candidate" ]; do
    resolve_depth=$((resolve_depth + 1))
    [ "$resolve_depth" -le 16 ] || return 1
    resolve_target=$(readlink "$resolve_candidate") || return 1
    case "$resolve_target" in
      /*) resolve_candidate=$resolve_target ;;
      *)
        resolve_parent=$(CDPATH='' cd "$(dirname "$resolve_candidate")" && pwd -P) || return 1
        resolve_candidate=$resolve_parent/$resolve_target
        ;;
    esac
  done
  resolve_parent=$(CDPATH='' cd "$(dirname "$resolve_candidate")" && pwd -P) || return 1
  printf '%s/%s\n' "$resolve_parent" "$(basename "$resolve_candidate")"
}

activate_user_bin() {
  mkdir -p "$loaf_user_bin_dir"
  loaf_link=$loaf_user_bin_dir/loaf
  if [ -e "$loaf_link" ] || [ -L "$loaf_link" ]; then
    [ -L "$loaf_link" ] || fail "$loaf_link exists and is not a project-managed symlink"
    existing_raw_target=$(readlink "$loaf_link") || fail "cannot read existing $loaf_link symlink"
    existing_is_owned=0
    case "$existing_raw_target" in
      "$orb_prefix"/releases/*/bin/loaf)
        existing_version=${existing_raw_target#"$orb_prefix"/releases/}
        existing_version=${existing_version%/bin/loaf}
        case "$existing_version" in
          */*) ;;
          *) valid_semver "$existing_version" && existing_is_owned=1 ;;
        esac
        ;;
    esac
    if [ "$existing_is_owned" -ne 1 ] && [ -e "$loaf_link" ]; then
      existing_target=$(resolve_path "$loaf_link") || fail "cannot resolve existing $loaf_link symlink"
      resolved_prefix=$(CDPATH='' cd "$orb_prefix" && pwd -P) || fail "cannot resolve Orb-local user prefix"
      case "$existing_target" in
        "$resolved_prefix"/releases/*/bin/loaf) existing_is_owned=1 ;;
      esac
    fi
    [ "$existing_is_owned" -eq 1 ] || fail "$loaf_link points outside this project's Orb-local prefix"
  fi

  temporary_link=$loaf_user_bin_dir/.loaf.new.$$
  [ ! -e "$temporary_link" ] && [ ! -L "$temporary_link" ] || fail "temporary user-bin link already exists"
  ln -s "$loaf_bin" "$temporary_link" || fail "cannot create user-bin activation symlink"
  mv -f "$temporary_link" "$loaf_link" || fail "cannot activate pinned loaf in user bin"
}

user_bin_is_active() {
  loaf_link=$loaf_user_bin_dir/loaf
  [ -L "$loaf_link" ] || return 1
  [ "$(readlink "$loaf_link")" = "$loaf_bin" ] || return 1
  resolved_link=$(resolve_path "$loaf_link") || return 1
  resolved_pinned=$(resolve_path "$loaf_bin") || return 1
  [ "$resolved_link" = "$resolved_pinned" ]
}

prepare_runtime_environment() {
  [ -n "${PATH:-}" ] || fail "PATH is required"
  PATH=$loaf_user_bin_dir:$PATH
  export PATH
  LOAF_PROJECT_ENV=1
  export LOAF_PROJECT_ENV
  resolved_command=$(command -v loaf) || fail "loaf is not available through PATH"
  resolved_command=$(resolve_path "$resolved_command") || fail "cannot resolve loaf from PATH"
  resolved_pinned=$(resolve_path "$loaf_bin") || fail "cannot resolve pinned loaf executable"
  [ "$resolved_command" = "$resolved_pinned" ] || fail "PATH loaf does not resolve to the pinned release"
}

installed_release_is_verified() {
  [ -f "$loaf_bin" ] && [ ! -L "$loaf_bin" ] && [ -x "$loaf_bin" ] || return 1
  [ -f "$manifest_path" ] && [ ! -L "$manifest_path" ] || return 1
  [ -f "$receipt_path" ] && [ ! -L "$receipt_path" ] || return 1
  [ "$(wc -l < "$receipt_path")" -eq 1 ] || return 1
  [ "$(wc -c < "$receipt_path")" -eq 65 ] || return 1
  [ -f "$platform_path" ] && [ ! -L "$platform_path" ] || return 1
  verified_archive_sha256=$(sed -n '1p' "$receipt_path")
  printf '%s\n' "$verified_archive_sha256" | grep -Eq '^[0-9a-f]{64}$' || return 1
  [ "$verified_archive_sha256" = "$archive_sha256" ] || return 1
  [ "$(sed -n '1p' "$platform_path")" = "$platform" ] || return 1
}

run_internal_readiness() {
  installed_release_is_verified || return 1
  user_bin_is_active || return 1
  (
    prepare_runtime_environment
    cd "$project_root"
    "$loaf_bin" harness readiness --target amp --pin "$pin_argument" --archive-sha256 "$verified_archive_sha256"
  )
}

run_agent_check() {
  [ "$inherited_project_env" = 1 ] || fail "Amp project environment must set LOAF_PROJECT_ENV=1"
  [ -n "$inherited_path" ] || fail "Amp project environment PATH is missing"
  installed_release_is_verified || fail "pinned Loaf release is missing or unverified"
  user_bin_is_active || fail "project-owned Loaf user-bin activation is missing or stale"
  (
    PATH=$inherited_path
    export PATH
    LOAF_PROJECT_ENV=$inherited_project_env
    export LOAF_PROJECT_ENV
    inherited_command=$(command -v loaf) || fail "Amp project environment PATH does not resolve loaf"
    [ "$inherited_command" = "$loaf_user_bin_dir/loaf" ] || fail "Amp project environment PATH does not select the project-owned loaf activation"
    resolved_inherited=$(resolve_path "$inherited_command") || fail "Amp project environment loaf activation cannot be resolved"
    resolved_pinned=$(resolve_path "$loaf_bin") || fail "pinned Loaf executable cannot be resolved"
    [ "$resolved_inherited" = "$resolved_pinned" ] || fail "Amp project environment loaf does not resolve to the pinned release"
    cd "$project_root"
    if ! readiness_output=$("$loaf_bin" harness readiness --target amp --pin "$pin_argument" --archive-sha256 "$verified_archive_sha256" 2>&1); then
      fail "pinned Loaf readiness check failed"
    fi
    [ -z "$readiness_output" ] || printf '%s\n' "$readiness_output"
  )
}

cleanup_dir=
lock_dir=$orb_prefix/.bootstrap.lock
lock_owner_path=$lock_dir/owner.pid
lock_acquired=0
cleanup() {
  if [ -n "$cleanup_dir" ] && [ -d "$cleanup_dir" ]; then
    rm -rf "$cleanup_dir"
  fi
  if [ "$lock_acquired" -eq 1 ]; then
    rm -f "$lock_owner_path"
    rmdir "$lock_dir" 2>/dev/null || printf '%s\n' "loaf-orb-bootstrap: warning: could not remove $lock_dir" >&2
    lock_acquired=0
  fi
}
trap cleanup 0
trap 'cleanup; exit 1' 1 2 15

acquire_bootstrap_lock() {
  mkdir -p "$orb_prefix"
  if mkdir "$lock_dir" 2>/dev/null; then
    lock_acquired=1
    printf '%s\n' "$$" > "$lock_owner_path" || fail "cannot record bootstrap lock owner"
    return 0
  fi
  [ -f "$lock_owner_path" ] && [ ! -L "$lock_owner_path" ] || fail "bootstrap lock has no valid PID owner: $lock_dir"
  [ "$(wc -l < "$lock_owner_path")" -eq 1 ] || fail "bootstrap lock has malformed PID owner: $lock_dir"
  lock_owner=$(sed -n '1p' "$lock_owner_path")
  printf '%s\n' "$lock_owner" | grep -Eq '^[1-9][0-9]*$' || fail "bootstrap lock has malformed PID owner: $lock_dir"
  if kill -0 "$lock_owner" 2>/dev/null; then
    fail "another setup or resume process $lock_owner holds $lock_dir"
  fi
  rm -f "$lock_owner_path" || fail "cannot retire stale bootstrap lock owner"
  rmdir "$lock_dir" 2>/dev/null || fail "cannot retire stale bootstrap lock"
  if mkdir "$lock_dir" 2>/dev/null; then
    lock_acquired=1
    printf '%s\n' "$$" > "$lock_owner_path" || fail "cannot record bootstrap lock owner"
    return 0
  fi
  fail "another setup or resume acquired $lock_dir"
}

install_verified_release() {
  command -v curl >/dev/null 2>&1 || fail "curl is required"
  command -v tar >/dev/null 2>&1 || fail "tar is required"
  if command -v shasum >/dev/null 2>&1; then
    checksum_command=shasum
  elif command -v sha256sum >/dev/null 2>&1; then
    checksum_command=sha256sum
  else
    fail "shasum or sha256sum is required"
  fi

  archive_name=loaf_${version}_${platform}.tar.gz
  official_base=https://github.com/levifig/loaf/releases
  release_base=$official_base
  if [ -n "${LOAF_ORB_TEST_RELEASE_BASE_URL:-}" ]; then
    [ "${LOAF_ORB_ISOLATED_TESTING:-}" = 1 ] || fail "release URL override requires LOAF_ORB_ISOLATED_TESTING=1"
    release_base=${LOAF_ORB_TEST_RELEASE_BASE_URL%/}
  fi
  archive_url=$release_base/download/v$version/$archive_name

  mkdir -p "$orb_prefix/releases"
  cleanup_dir=$(mktemp -d "${TMPDIR:-/tmp}/loaf-orb-bootstrap.XXXXXX") || fail "cannot create temporary directory"
  archive_path=$cleanup_dir/$archive_name
  unpack_dir=$cleanup_dir/unpack
  mkdir -p "$unpack_dir"

  if [ "$release_base" = "$official_base" ]; then
    curl --fail --location --silent --show-error --connect-timeout 10 --max-time 120 --retry 2 --retry-delay 1 --proto '=https' --tlsv1.2 --output "$archive_path" "$archive_url" || fail "release download failed"
  else
    curl --fail --location --silent --show-error --connect-timeout 10 --max-time 120 --retry 2 --retry-delay 1 --output "$archive_path" "$archive_url" || fail "isolated-test release download failed"
  fi

  case "$checksum_command" in
    shasum) actual_sha256=$(shasum -a 256 "$archive_path") ;;
    sha256sum) actual_sha256=$(sha256sum "$archive_path") ;;
  esac
  actual_sha256=${actual_sha256%% *}
  [ "$actual_sha256" = "$archive_sha256" ] || fail "archive checksum does not match the project pin"

  tar -xpzf "$archive_path" -C "$unpack_dir" || fail "release extraction failed"
  unpacked_release=$unpack_dir/loaf_${version}_${platform}
  [ -d "$unpacked_release" ] || fail "archive is missing the expected release directory"
  [ -f "$unpacked_release/loaf-release-manifest.json" ] || fail "archive is missing loaf-release-manifest.json"
  [ -x "$unpacked_release/bin/loaf" ] || fail "archive is missing executable bin/loaf"

  staged_release=$orb_prefix/releases/.${version}.new.$$
  rm -rf "$staged_release"
  mv "$unpacked_release" "$staged_release" || fail "cannot stage the verified release"
  printf '%s\n' "$actual_sha256" > "$staged_release/.loaf-archive.sha256"
  printf '%s\n' "$platform" > "$staged_release/.platform"
  if [ -e "$release_dir" ] || [ -L "$release_dir" ]; then
    rm -rf "$release_dir"
  fi
  mv "$staged_release" "$release_dir" || fail "cannot activate the verified release"
}

repair_and_install() {
  acquire_bootstrap_lock
  if run_internal_readiness; then
    run_agent_check
    return
  fi
  install_verified_release
  installed_release_is_verified || fail "published release failed provenance verification"
  activate_user_bin
  prepare_runtime_environment
  (
    cd "$project_root"
    LOAF_PROJECT_ENV=1 "$loaf_bin" install --to amp --yes
  )
  run_internal_readiness || fail "installed Loaf content did not become internally ready"
  run_agent_check
}

case "${1:-}" in
  setup)
    if run_internal_readiness; then
      run_agent_check
      exit
    fi
    repair_and_install
    ;;
  resume)
    if run_internal_readiness; then
      run_agent_check
      exit
    fi
    repair_and_install
    ;;
  agent-check)
    run_agent_check
    ;;
  *)
    fail "usage: loaf-orb-bootstrap.sh setup|resume|agent-check"
    ;;
esac
