#!/usr/bin/env bash

set -e
set -o pipefail

error() {
  echo "ERROR: $@" 1>&2
  echo "Exiting installer" 1>&2
  exit 1
}

ARCH=`uname -m`

if [ "${ARCH}" != "x86_64" ]; then
  error "summon only works on 64-bit systems"
fi

DISTRO=`uname | tr "[:upper:]" "[:lower:]"`

if [ "${DISTRO}" != "linux" ] && [ "${DISTRO}" != "darwin"  ]; then
  error "This installer only supports Linux and OSX"
fi

tmp="/tmp"
if [ ! -z "$TMPDIR" ]; then
  tmp=$TMPDIR
fi

# secure-ish temp dir creation without having mktemp available (DDoS-able but not exploitable)
tmp_dir="$tmp/install.sh.$$"
(umask 077 && mkdir $tmp_dir) || exit 1

# do_download URL DIR
do_download() {
  echo "Downloading $1"
  if [[ $(command -v wget) ]]; then
    wget -q -O "$2" "$1" >/dev/null
  elif [[ $(command -v curl) ]]; then
    curl --fail -sSL -o "$2" "$1" &>/dev/null || true
  else
    error "Could not find wget or curl"
  fi
}

# get_latest_version
get_latest_version() {
  local LATEST_VERSION_URL="https://api.github.com/repos/slamdf150xc/summon-provider-ccp/releases"
  local latest_payload

  if [[ $(command -v wget) ]]; then
    latest_payload=$(wget -q -O - "$LATEST_VERSION_URL")
  elif [[ $(command -v curl) ]]; then
    latest_payload=$(curl --fail -sSL "$LATEST_VERSION_URL")
  else
    error "Could not find wget or curl"
  fi

  echo "$latest_payload" | # Get latest release from GitHub api
    grep '"tag_name":' | # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/' # Pluck JSON value
}

LATEST_VERSION=$(get_latest_version)

echo "Using version number: $LATEST_VERSION"

BASEURL="https://github.com/slamdf150xc/summon-provider-ccp/releases/download/"
URL=${BASEURL}"${LATEST_VERSION}/summon-provider-ccp-linux-amd64.tar.gz"

ZIP_PATH="${tmp_dir}/summon_ccp.tar.gz"
do_download ${URL} ${ZIP_PATH}

echo "Installing Summon CCP Provider ${LATEST_VERSION} into /usr/local/lib/summon"

PROVIDER_CCP_DIR=/usr/local/lib/summon
if [[ ! -e $PROVIDER_CCP_DIR ]]; then
  mkdir -p $PROVIDER_CCP_DIR
fi

if sudo -h >/dev/null 2>&1; then
  sudo tar -C /usr/local/lib/summon -o -zxvf ${ZIP_PATH} >/dev/null
else
  tar -C /usr/local/lib/summon -o -zxvf ${ZIP_PATH} >/dev/null
fi

echo "Success!"