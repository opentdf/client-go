# OpenTDF Golang Client

This is a light Go wrapper around the OpenTDF C++ client SDK (https://github.com/opentdf/client-cpp), via that library's C interop.

## Install as a Go module

Since `opentdf/client-go` depends on the [opentdf/client-cpp](https://github.com/opentdf/client-cpp) binary, the library binaries and include files of that library
must be present in your Go environment before you can `go build` this client, or anything that depends on it, and `CGO_CFLAGS` and `CGO_LDFLAGS` must be set accordingly.

See [Dockerfile.example](./Dockerfile.example) for a complete, working example of what steps are required to install `client-cpp` and then build a Go project that uses this library in a clean environment.

Nominally, the steps are: 
1. Install conan (i.e. `brew install conan`)
1. Install a [`client-cpp`](https://github.com/opentdf/client-cpp) release (`conan install opentdf-client/1.1.3@ --build=missing -g deploy -if /my-workdir/client-cpp`)
1. ```sh
    export CGO_LDFLAGS="-L/my-workdir/client-cpp/opentdf-client/lib"
    export CGO_CFLAGS="-I/my-workdir/client-cpp/opentdf-client/include"
    export CGO_ENABLED=1```
1. Install `client-go` module normally (`go get opentdf/client-go`)
1. Since `client-go` depends on a C++ library, platform support is not the same as a pure Go program, and is constrained by whatever platforms the underlying C++ `client-cpp` library supports (currently linux and macOS amd64+arm64)

## Caveats

1. The OpenTDF C interop only supports encrypting files and strings, so everything has to be passed as strings (or file paths) - no streaming.

1. Go is very fast - but Go->C calls are 9X slower than pure Go calls, due to memory copying - to preserve safety, Go does not share memory space with C code. The Go interop is faster than the Python wrapper/JS SDK, but far slower than a pure Go SDK, or direct use of the C++ SDK.

1. Go code can easily be compiled/cross compiled to over a dozen different platforms and architectures out of the box with no extra work or extra tooling, and dependencies are always dynamically compiled when fetched - C cannot support any of this, so by inference Go code that depends on C code loses the ability to be easily cross-compiled and distributed.

## Highly unscientific performance numbers

    {"level":"info","ts":1614204786.0663092,"caller":"opentdf-client/opentdfclient.go:83","msg":"Initializing OpenTDF C SDK"}
    2021/02/24 17:13:07
    Operation encrypt #1: 1.551678274s
    2021/02/24 17:13:08
    Operation decrypt #1: 634.32529ms
    Round trip decrypted: holla at ya boi2021/02/24 17:13:08
    Operation encrypt #2: 603.196413ms
    2021/02/24 17:13:09
    Operation decrypt #2: 490.061242ms
    Round trip decrypted: holla at ya boi2021/02/24 17:13:09
    Operation encrypt #3: 606.305416ms
    2021/02/24 17:13:11
    Operation decrypt #3: 1.161271025s
    Round trip decrypted: holla at ya boi2021/02/24 17:13:11
    Operation encrypt #4: 485.090529ms
    2021/02/24 17:13:12
    Operation decrypt #4: 617.335109ms
    Round trip decrypted: holla at ya boi2021/02/24 17:13:12
    Operation encrypt #5: 570.543769ms
    2021/02/24 17:13:13
    Operation decrypt #5: 511.418639ms
    Round trip decrypted: holla at ya boi2021/02/24 17:13:13
  
## Testing

For now there's a simple wrapper exerciser binary you can build in `cmd/wrapper`

1. `cd cmd/wrappertest`
1. `go build`
This is more practically useful than native Go unit tests given the minimal amount of Go logic.
OIDC auth is the only auth mechanism supported, and currently requires setting additional environment variables, see `sequentialOIDC()` in [cmd/wrappertest/main.go](cmd/wrappertest/main.go)

The env vars required for the exerciser binary in OIDC Client Credentials mode (assuming locally-hosted services) are:

```shell
export TDF_OIDC_URL="http://localhost:8080"
export TDF_KAS_URL="http://localhost:8000"
export TDF_ORGNAME="tdf"
export TDF_CLIENTID="tdf-client"
export TDF_CLIENTSECRET="123-456"
# Currently TDF_USER is...unused, but the underlying C++ SDK expects it, so have to pass it
export TDF_USER="dan@gmail.com"
# If you are using OIDC Token Exchange, you may additionally set:
export TDF_EXTERNALTOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJle..."
```

## Building this library locally

Since `opentdf/client-go` depends on the [opentdf/client-cpp](https://github.com/opentdf/client-cpp) binary, the library binaries and include files of that library
must be present in your Go environment before you can `go build` this client, or anything that depends on it, and `CGO_CFLAGS` and `CGO_LDFLAGS` must be set accordingly.

See [Dockerfile.example](./Dockerfile.example) for a complete, working example of what steps are required to install `client-cpp` and then build a Go project that uses this library in a clean environment,
but nominally the steps are: 
1. Install conan (i.e. `brew install conan`)
1. Install a [`client-cpp`](https://github.com/opentdf/client-cpp) release (`conan install opentdf-client/1.1.3@ --build=missing -g deploy -if /my-workdir/client-cpp`)
1. ```sh
    export CGO_LDFLAGS="-L/my-workdir/client-cpp/opentdf-client/lib"
    export CGO_CFLAGS="-I/my-workdir/client-cpp/opentdf-client/include"
    export CGO_ENABLED=1```
1. Build from the repo root like normal (`go build .`)

## Example Code

See [cmd/wrappertest/main.go](./cmd/wrappertest/main.go)
