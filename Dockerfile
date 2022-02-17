# This is to work around SDK artifact access issues.
FROM opentdf/client-cpp-base:0.5.1-dcr AS sdkbase

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    CGO_LDFLAGS="-L/build/tdf-client/lib" \
    CGO_CFLAGS="-I/build/tdf-client/include"

WORKDIR /build

COPY . .

# Let's create a /dist folder containing just the files necessary for runtime.
# Later, it will be copied as the / (root) of the output image.
RUN mkdir /dist

#Build the demo executable - can be skipped if not needed
RUN go build  -o /dist/wrappertest ./cmd/wrappertest

# Build the library - Library consumers will build this library during dependency resolution,
# but doing it here as well for the sake of example
RUN go build -o /dist/client-go
