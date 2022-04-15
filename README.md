# OpenTDF Golang Client

This is a light Go wrapper around the OpenTDF C++ client SDK (https://github.com/opentdf/client-cpp), via that library's C interop.

## Caveats

1. The OpenTDF C interop only supports encrypting files and strings, so everything has to be passed as strings (or file paths) - no streaming.

1. Go is very fast - but Go->C calls are 9X slower than pure Go calls, due to memory copying - to preserve safety, Go does not share memory space with C code. The Go interop is faster than the Python wrapper/JS SDK, but far slower than a pure Go SDK, or direct use of the C++ SDK.

1. Go code can easily be compiled/cross compiled to over a dozen different platforms and architectures out of the box with no extra work or extra tooling, and dependencies are always dynamically compiled when fetched - C cannot support any of this, so by inference Go code that depends on C code loses the ability to be easily cross-compiled and distributed.

1. Right now this only builds if the OpenTDF CPP static library and header files are in the right spots - you will need to set that up yourself if you plan to do builds outside of the provided Docker build environment, see [opentdf-client-cpp-base/Dockerfile](opentdf-client-cpp-base/Dockerfile) for an example of how that's done.

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
  
## Building

1. `git clone` this repo
1. Run `make dockerbuild` (Builds sample Go programs against the `client-go` wrapper, using a special container preloaded with the `client-cpp` static libraries and headers)

Since dev builds of the OpenTDF C++ client SDK are not published at the time of this writing (soon), we manually clone, build, and pack the SDK into a base image in [./opentdf-client-cpp-base](./opentdf-client-cpp-base), publish that to our internal repo, and use that as the base image for Go builds that depend on the C++ SDK, for now.

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

## Using as a "normal" Go dependency

Since this is not a pure Go library and depends on the OpenTDF C wrapper (which ships with OpenTDF's C++ SDK), when fetching and building the Go
package as a dependency you must tell `cgo` where to find the OpenTDF C header files and OpenTDF static library.

We hide some of this by generating a Docker image with the correct headers and libraries (see [./opentdf-client-cpp-base](./opentdf-client-cpp-base)), but if you don't want to use that as your app's base image, or want to build this outside of docker, it gets a little more complicated - you have to fetch the headers _and correct C libraries for your platform_, and use the

- `CGO_LDFLAGS`
- `CGO_CFLAGS`

environment variables to tell `cgo` where to find those things.

### Example

1. `mkdir my-go-project`
1. `cd my-go-project && go mod init github.com/myorg/my-go-project`
1. At this point you have a normal Go project, but you wanna bring in `opentdf/client-go` as a dependency
1. `go get github.com/opentdf/client-go`
1. If you `go build` at this point, your Go program will rightly complain that `opentdf/client-go` is looking for C headers and libraries, and it can't locate them.
1. Download/obtain the OpenTDF C++ SDK **for your OS/architecture** (public release zip is fine, or you can use a dev build)
1. `mkdir client-cpp`
1. `cp $CPP_LIBRARY/src/include tdf-cpp/include`
1. `cp $CPP_LIBRARY/src/build/lib tdf-cpp/lib`
1. Now run `CGO_ENABLED=1 CGO_LDFLAGS="-L./tdf-cpp/lib" CGO_CFLAGS="-I./tdf-cpp/include" go build`
1. Your Go app should build

> An example Dockerfile demonstrating a complete build environment can be found [here](./Dockerfile).
>
> You can build this Docker image by running `make dockerbuild`

## Example Code

See [cmd/wrappertest/main.go](./cmd/wrappertest/main.go)
