#!/usr/bin/env bash
set -eEuo pipefail

# Build Disco binaries for Linux, macOS and Windows, packaging them in their
# platform-native formats.
# Based on https://github.com/grafana/k6/blob/v0.50.0/build-release.sh


eval "$(go env)"

_bin_name="disco"
_proj_dir="$(git rev-parse --show-toplevel)"

# Arguments
_platforms="${1:-linux macos windows}"
_version="${2:-$(git describe --tags --abbrev=10 --always --dirty)}"
_dest_dir="${3-${_proj_dir}/dist}"

build() {
  declare _env="$1"  # comma-separated environment variables
  declare -n _path="$2"

  # Backup original environment, and export the received one.
  declare -A _orig_env
  declare _old_ifs=$IFS
  IFS=','
  for _env_pair in $_env; do
    IFS='=' read -r _key _value <<< "$_env_pair"
    _orig_env["$_key"]="$(eval echo "\$$_key")"
    export "${_key}=${_value}"
  done
  IFS=$_old_ifs

  declare _os="$GOOS"
  if [ "$GOOS" = "darwin" ]; then
    _os="macos"
  fi

  declare _name="${_bin_name}-${_version}-${_os}-${GOARCH}"
  # Set return value
  _path="${_dest_dir}/${_name}"
  mkdir -p "$_path"

  declare _suffix=""
  if [ "$GOOS" = "windows" ]; then
    _suffix=".exe"
  fi

  declare _build_args=(
    -o "${_path}/${_bin_name}${_suffix}" -trimpath
  )

  if [ -n "$_version" ]; then
    _build_args+=(-ldflags "-X go.hackfix.me/disco/app/context.vcsVersion=${_version}")
  fi

  log "Building\t${_name}"
  go build "${_build_args[@]}" "${_proj_dir}/cmd/${_bin_name}"

  # Restore original environment
  for _key in "${!_orig_env[@]}"; do
    export "$_key=${_orig_env[$_key]}"
  done
}

package() {
  declare _path="$1" _fmt="$2" _name="$(basename "$1")"

  log "Packaging\t${_name}.${_fmt}"

  case $_fmt in
  deb|rpm)
    # nfpm can't substitute env vars in file paths, so we have to cd...
    cd "$_path"
    set -x
    nfpm package --config ../../packaging/nfpm.yaml --packager "${_fmt}" \
      --target "../${_name}.${_fmt}"
    set +x
    cd -
    ;;
  tgz)
    tar -C "${_dest_dir}" -zcf "${_path}.tar.gz" "$_name"
    ;;
  zip)
    (cd "${_dest_dir}" && zip -rq9 "${_path}.zip" "$_name")
    ;;
  *)
    quit "Unknown format: $_fmt"
    ;;
  esac
}

log() {
  echo -e "$(date -Iseconds)" "$*"
}

err() {
  echo -e "ERROR:" "$*"
}

quit() {
  err "$*"
  exit 1
}

cleanup() {
  find "$_dest_dir" -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} \;
  log "Cleaned ${_dest_dir}"
}
[ -z "${NO_CLEANUP-}" ] && trap cleanup EXIT

build_release() {
  log "Building release ${_version} in ${_dest_dir}"
  export CGO_ENABLED=0

  for _pf in $_platforms; do
    IFS='-' read -r _goos _goarch <<< "$_pf"
    case "$_pf" in
      linux*)
        if [ "$_goarch" = "amd64" ] || [ "$_pf" = "linux" ]; then
          build GOOS=linux,GOARCH=amd64 _build_path
          package "$_build_path" "tgz"
        fi
        if [ "$_goarch" = "arm64" ] || [ "$_pf" = "linux" ]; then
          build GOOS=linux,GOARCH=arm64 _build_path
          package "$_build_path" "tgz"
        fi
        ;;
      macos*)
        if [ "$_goarch" = "arm64" ] || [ "$_pf" = "macos" ]; then
          build GOOS=darwin,GOARCH=arm64 _build_path
          package "$_build_path" "zip"
        fi
        ;;
      windows*)
        build GOOS=windows,GOARCH=amd64 _build_path
        package "$_build_path" "zip"
        ;;
      *)
        quit "invalid platform: ${_pf}"
        ;;
    esac
  done
}

build_release "$_platforms"
