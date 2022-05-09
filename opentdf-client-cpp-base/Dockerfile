# This image does nothing but serve as a base image containing the OpenTDF client-cpp SDK
# This is to work around
# - artifact access issues
# - to easily generate a arm64 and amd64 build for use in containers (ubuntu base has both arm64 and amd64 variants).
#
# Alpine would be ideal - but the client-cpp C lib complicates that
# as we don't have a `musl` build yet
FROM ubuntu:20.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y build-essential cmake pip

RUN pip install conan

WORKDIR /build

COPY --chmod=777 client-cpp client-cpp

WORKDIR client-cpp/src/

RUN conan install . --build=missing

RUN sh build-all.sh

FROM golang:1.17-bullseye

COPY --chown=0:0 --from=builder /build/client-cpp/src/build/lib /build/tdf-client/lib
COPY --chown=0:0 --from=builder /build/client-cpp/src/lib/include /build/tdf-client/include
