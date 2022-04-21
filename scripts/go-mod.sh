#!/usr/bin/env sh
OTDF_PREFIX="[OpenTDF Client Install]"
GO_CLIENT=github.com/opentdf/client-go
PKG=opentdf-client
[[ -z $QUIET ]] && echo "$OTDF_PREFIX Searching for $PKG CPP library in Conan..."

if [[ -z $(which conan) ]]; then
  echo "$OTDF_PREFIX Error: Conan is required (e.g. brew install conan)"
  exit 0
fi

M_OS=$(uname -s)
if [[ $M_OS == "Darwin" ]]; then
  OS="Macos"
elif [[ $M_OS == "Linux" ]]; then
  OS="Linux"
else
  echo "$OTDF_PREFIX Error: OS $M_OS is not supported"
  exit 0
fi
OS_ARCH=$(uname -a | rev | cut -d" " -f1 | rev)

VER=$(conan search $PKG -r all | awk 'END{print}')
[[ -z $QUIET ]] && echo "$OTDF_PREFIX Found $VER"

PKGID=$(conan search "$VER@" -q "os=$OS AND (arch=$OS_ARCH)" | grep "Package_ID: " | rev | cut -d" " -f1 | rev)
[[ -z $QUIET ]] && echo "$OTDF_PREFIX Found Package ID $PKGID"

[[ -z $QUIET ]] && echo "$OTDF_PREFIX Downloading $VER:$PKGID..."
conan download "$VER:$PKGID"

[[ -z $QUIET ]] && echo "$OTDF_PREFIX Installing go mod $GO_CLIENT with CPP Flags"
CGO_ENABLED=1 \
CGO_LDFLAGS="-L$HOME/.conan/data/$VER/_/_/package/$PKGID/lib" \
CGO_CFLAGS="-I$HOME/.conan/data/$VER/_/_/package/$PKGID/include" \
go get "$GO_CLIENT"
