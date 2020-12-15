# This image does nothing but serve as a base image containing the Virtru C++ SDK
# This is to work around artifact access issues.
#
# Alpine would be ideal - but the Virtru C lib complicates that
# as we don't have a `musl` build yet
FROM conanio/gcc8-ubuntu16.04:1.43.0 AS builder

WORKDIR /build

COPY --chmod=777 client-cpp client-cpp
# USER root
# RUN apt update && apt install openssh-client -y
# USER conan
# # Git creds come from host mount, but we need to do some setup to make sure fetching internal Virtru
# # private Git repos via Go works correctly
# # hadolint ignore=SC2174
# RUN git config --global url."git@github.com:".insteadOf "https://github.com/" \
#     && mkdir -p -m 0700 ~/.ssh \
#     && ssh-keyscan github.com >> ~/.ssh/known_hosts

# # Vendor modules here (that is, download them, using SSH creds if need be) so that subsequent steps don't
# # re-fetch, and just use the vendored versions
# RUN --mount=type=ssh GIT_SSH_COMMAND="ssh -vvv" git clone git@github.com:opentdf/client-cpp.git

WORKDIR client-cpp/src/

RUN conan install . --build=missing

RUN sh build-all.sh

# This assumes the Linux x86 build of the Virtru CPP SDK has been downloaded already
# We don't publish dev builds - you have to manually auth with AWS and do an S3 fetch from
# BuildKite
#
# If you're using a public release, you can just WGET it to this folder:
# wget -q -O virtru-tdf3-lib-cpp-linux-gnu-x86_64.tar.gz https://sdk.virtru.com/cpp/2.1.1/virtru-tdf3-lib-cpp-lin.tar.gz
# COPY virtru-tdf3-lib-cpp-linux-gnu-x86_64.tar.gz .

# RUN tar -xf virtru-tdf3-lib-cpp-linux-gnu-x86_64.tar.gz

FROM golang:1.17-buster

COPY --chown=0:0 --from=builder /build/client-cpp/src/build/lib /build/tdf-client/lib
COPY --chown=0:0 --from=builder /build/client-cpp/src/lib/include /build/tdf-client/include
