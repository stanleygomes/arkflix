# arkflix-server: um binário + ffmpeg. Roda do lado do Jellyfin/Navidrome/RomM
# lendo as mesmas pastas (montadas só leitura) até eles saírem.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /arkflix-server ./cmd/arkflix-server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates tzdata \
  && rm -rf /var/lib/apt/lists/* && useradd -r -u 1000 -m arkflix
COPY --from=build /arkflix-server /usr/local/bin/arkflix-server
USER arkflix
ENV ARKFLIX_SERVER_ADDR=:8097 ARKFLIX_SERVER_DATA=/data
EXPOSE 8097
VOLUME /data
HEALTHCHECK CMD ["/bin/sh", "-c", "exec 3<>/dev/tcp/127.0.0.1/8097"]
ENTRYPOINT ["arkflix-server"]
