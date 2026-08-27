FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=v2.1.0
ARG SOURCE=https://github.com/jormar/md2confluence.git
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION} -X main.source=${SOURCE}" -o /out/md2c ./cmd/md2c

FROM debian:bookworm-slim
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*
COPY --from=build /out/md2c /usr/local/bin/md2c
ENTRYPOINT ["md2c"]
