#!/bin/sh
# Install and verify one project-pinned Loaf release for an Amp Orb.
# POSIX sh only: this script intentionally has no source-checkout or language
# toolchain dependency.
set -eu

fail() {
  printf '%s\n' "loaf-orb-bootstrap: $*" >&2
  exit 1
}

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
printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?(\+[0-9A-Za-z][0-9A-Za-z.-]*)?$' || fail "invalid release version"
printf '%s\n' "$checksum_x64" | grep -Eq '^[0-9a-f]{64}$' || fail "invalid linux-x64 checksum"
printf '%s\n' "$checksum_arm64" | grep -Eq '^[0-9a-f]{64}$' || fail "invalid linux-arm64 checksum"
printf '%s\n' "$project_id" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._:-]*$' || fail "invalid project ID"
printf '%s\n' "$git_remote" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$' || fail "git.remote must be an unauthenticated owner/repository name"
printf '%s\n' "$tracker_provider" | grep -Eq '^[a-z][a-z0-9-]*$' || fail "invalid tracker provider"
printf '%s\n' "$tracker_scope" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._:/#+-]*$' || fail "invalid tracker scope"
case "$tracker_scope" in
  *://*|[A-Za-z]:/*) fail "tracker scope must not be a URL or machine path" ;;
esac

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
    existing_target=$(resolve_path "$loaf_link") || fail "cannot resolve existing $loaf_link symlink"
    resolved_prefix=$(CDPATH='' cd "$orb_prefix" && pwd -P) || fail "cannot resolve Orb-local user prefix"
    case "$existing_target" in
      "$resolved_prefix"/releases/*/bin/loaf) ;;
      *) fail "$loaf_link points outside this project's Orb-local prefix" ;;
    esac
  fi

  temporary_link=$loaf_user_bin_dir/.loaf.new.$$
  [ ! -e "$temporary_link" ] && [ ! -L "$temporary_link" ] || fail "temporary user-bin link already exists"
  ln -s "$loaf_bin" "$temporary_link" || fail "cannot create user-bin activation symlink"
  mv -f "$temporary_link" "$loaf_link" || fail "cannot activate pinned loaf in user bin"
}

user_bin_is_active() {
  loaf_link=$loaf_user_bin_dir/loaf
  [ -L "$loaf_link" ] || return 1
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

run_readiness() {
  installed_release_is_verified || return 1
  user_bin_is_active || return 1
  prepare_runtime_environment
  (
    cd "$project_root"
    "$loaf_bin" harness readiness --target amp --pin "$pin_argument" --archive-sha256 "$verified_archive_sha256"
  )
}

cleanup_dir=
lock_dir=$orb_prefix/.bootstrap.lock
lock_acquired=0
cleanup() {
  if [ -n "$cleanup_dir" ] && [ -d "$cleanup_dir" ]; then
    rm -rf "$cleanup_dir"
  fi
  if [ "$lock_acquired" -eq 1 ]; then
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
    return 0
  fi
  fail "another setup or resume holds $lock_dir"
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
    curl --fail --location --silent --show-error --proto '=https' --tlsv1.2 --output "$archive_path" "$archive_url" || fail "release download failed"
  else
    curl --fail --location --silent --show-error --output "$archive_path" "$archive_url" || fail "isolated-test release download failed"
  fi

  case "$checksum_command" in
    shasum) actual_sha256=$(shasum -a 256 "$archive_path") ;;
    sha256sum) actual_sha256=$(sha256sum "$archive_path") ;;
  esac
  actual_sha256=${actual_sha256%% *}
  [ "$actual_sha256" = "$archive_sha256" ] || fail "archive checksum does not match the project pin"

  tar -xzf "$archive_path" -C "$unpack_dir" || fail "release extraction failed"
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
  if run_readiness; then
    return 0
  fi
  install_verified_release
  installed_release_is_verified || fail "published release failed provenance verification"
  activate_user_bin
  prepare_runtime_environment
  (
    cd "$project_root"
    LOAF_PROJECT_ENV=1 "$loaf_bin" install --to amp --yes
    "$loaf_bin" harness readiness --target amp --pin "$pin_argument" --archive-sha256 "$verified_archive_sha256"
  )
}

case "${1:-}" in
  setup)
    if run_readiness; then
      exit 0
    fi
    repair_and_install
    ;;
  resume)
    if run_readiness; then
      exit 0
    fi
    repair_and_install
    ;;
  *)
    fail "usage: $0 setup|resume"
    ;;
esac
