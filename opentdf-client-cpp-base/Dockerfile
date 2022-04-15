# This image does nothing but serve as a base image containing the OpenTDF client-cpp SDK
# This is to work around artifact access issues.
#
# Alpine would be ideal - but the client-cpp C lib complicates that
# as we don't have a `musl` build yet
FROM conanio/gcc8-ubuntu16.04:1.43.0 AS builder

WORKDIR /build

COPY --chmod=777 client-cpp client-cpp

WORKDIR client-cpp/src/

RUN conan install . --build=missing

RUN sh build-all.sh

FROM golang:1.17-buster

COPY --chown=0:0 --from=builder /build/client-cpp/src/build/lib /build/tdf-client/lib
COPY --chown=0:0 --from=builder /build/client-cpp/src/lib/include /build/tdf-client/include
