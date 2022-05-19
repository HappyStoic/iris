FROM golang:latest AS build

ENV APP_DIR /usr/iris
WORKDIR ${APP_DIR}

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg

RUN go build cmd/peercli.go


FROM debian:bullseye-slim as final

ENV GOLOG_LOG_LEVEL="error,iris=info"
ENV APP_DIR /usr/iris
WORKDIR ${APP_DIR}

COPY config.yaml ./
COPY --from=build ${APP_DIR}/peercli ./
CMD ["./peercli", "--conf", "config.yaml"]
