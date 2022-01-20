FROM golang:latest AS build

ENV APP_DIR /usr/p2pnetwork
WORKDIR ${APP_DIR}

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg

RUN go build cmd/peercli.go


FROM debian:bullseye-slim as final

ENV GOLOG_LOG_LEVEL="error,p2pnetwork=info"
ENV APP_DIR /usr/p2pnetwork
WORKDIR ${APP_DIR}

COPY config.yaml ./
COPY --from=build ${APP_DIR}/peercli ./
CMD ["./peercli", "--conf", "config.yaml"]
