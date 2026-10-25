FROM --platform=$BUILDPLATFORM golang:1.26.8-alpine AS build
WORKDIR /src

COPY go.mod ./
COPY *.go ./
COPY web/ ./web/

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/flipflash .

FROM scratch
COPY --from=build /out/flipflash /flipflash
USER 65532:65532
EXPOSE 8081
ENTRYPOINT ["/flipflash"]
CMD ["--dir", "/photos"]
