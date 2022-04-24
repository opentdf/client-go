#!/usr/bin/env sh

# This script will install the OpenTDF CPP library and prepare the
# environment for use with `go mod get github.com/opentdf/client-go`
set -u

abort() {
  printf "%s\n" "$@" >&2
  exit 1
}

log() {
  printf "[OpenTDF Client Install] %s\n" "$@" >&2
}

GO_CLIENT=github.com/opentdf/client-go
PKG=opentdf-client
log "Searching for $PKG CPP library in Conan..."

if [ -z "$(which conan)" ]; then
  abort "Conan is required (e.g. brew install conan)"
fi

M_OS=$(uname -s)
if [ $M_OS = "Darwin" ]; then
  OS="Macos"
elif [ $M_OS = "Linux" ]; then
  OS="Linux"
else
  abort "OS $M_OS is not supported"
fi
OS_ARCH=$(uname -a | rev | cut -d" " -f1 | rev)

VER=$(conan search $PKG -r all | awk 'END{print}')
log "Found $VER"

PKGID=$(
  conan search "$VER@" -q "os=$OS AND (arch=$OS_ARCH)" |
  grep "Package_ID: " |
  rev |
  cut -d" " -f1 |
  rev
)
log "Found Package ID $PKGID"

log "Downloading $VER:$PKGID..."
conan download "$VER:$PKGID"

log "Installing go mod $GO_CLIENT with CPP Flags"
export CGO_ENABLED=1 \
export CGO_LDFLAGS="-L$HOME/.conan/data/$VER/_/_/package/$PKGID/lib" \
export CGO_CFLAGS="-I$HOME/.conan/data/$VER/_/_/package/$PKGID/include" 

log "CPP lib setup complete, you may now build your Go modules normally"
log ""
log "    go get github.com/opentdf/client-go"
