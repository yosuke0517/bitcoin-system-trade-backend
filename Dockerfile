FROM golang:1.15-alpine3.13 as build

LABEL maintainer="https://github.com/yosuke0517"

WORKDIR /go/app

COPY . .

ENV GO111MODULE=off
ENV CGO_ENABLED=0

RUN set -eux && \
  apk update && \
  apk add --no-cache git curl gcc alpine-sdk && \
  go get -u github.com/labstack/echo/... && \
  go get golang.org/x/tools/cmd/godoc && \
  go get -v github.com/rubenv/sql-migrate/...

ENV GO111MODULE on

RUN set -eux && \
  go build -o bitcoin-system-trade-backend ./main.go

FROM alpine:3.11

WORKDIR /app

COPY --from=build /go/app/bitcoin-system-trade-backend .

RUN set -x && \
  addgroup go && \
  adduser -D -G go go && \
  chown -R go:go /app/bitcoin-system-trade-backend

#CMD ["./bitcoin-system-trade-backend"]
CMD ["/startup.sh"]