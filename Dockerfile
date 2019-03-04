FROM lumas/lumas-base-ffmpeg as build

ENV LD_LIBRARY_PATH=/opt/ffmpeg/lib
ENV PKG_CONFIG_PATH=/opt/ffmpeg/lib/pkgconfig

RUN apk add --update git pkgconfig gcc libc-dev \
    openssl lame libogg libvpx libvorbis libass \
    freetype libtheora opus libwebp x264 x264-libs x265 && \
    go get -u github.com/3d0c/gmf && \
    go get -u github.com/golang/protobuf/proto && \
    go get -u github.com/golang/protobuf/ptypes/struct && \
    go get -u google.golang.org/grpc && \
    go get -u github.com/lumas-ai/lumas-provider-onvif && \
    cd / && go build /go/src/github.com/lumas-ai/lumas-provider-onvif/cmd/onvif/onvif-server.go


FROM alpine:3.9

ENV LD_LIBRARY_PATH=/opt/ffmpeg/lib
ENV PKG_CONFIG_PATH=/opt/ffmpeg/lib/pkgconfig

COPY --from=build /opt/ffmpeg /opt/ffmpeg
COPY --from=build /onvif-server /onvif-server

RUN apk add --update openssl lame libogg libvpx libvorbis libass \
    freetype libtheora opus libwebp x264 x264-libs x265

ENTRYPOINT ["/onvif-server"]
