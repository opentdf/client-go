FROM golang:1.15 as build
WORKDIR /src
COPY . .

WORKDIR /src/wrapper
RUN make clean && make

WORKDIR /src
RUN go build -x -o /out/main .

RUN cp /src/wrapper/libwrapper.so /usr/lib
RUN cp /src/virtru-tdf3-lib-cpp/lib/libvirtru_tdf3_static_combined.a /usr/lib

ENTRYPOINT ["/out/main"]
