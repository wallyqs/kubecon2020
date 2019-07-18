FROM golang:1.12-alpine3.10 AS builder

MAINTAINER Derek Collison <derek@nats.io>

WORKDIR $GOPATH/src/github.com/connecteverything/

RUN apk add -U --no-cache git binutils

# Force the go compiler to use modules
ENV GO111MODULE=on

RUN go get github.com/nats-io/nsc

# We want to populate the module cache based on the go.{mod,sum} files.
RUN mkdir chat
COPY ./chat/go.mod ./chat
COPY ./chat/go.sum ./chat
RUN cd chat && go mod download

RUN mkdir util
COPY ./nats-util/go.mod ./util
COPY ./nats-util/go.sum ./util
RUN cd chat && go mod download

COPY . .
RUN cd chat && go install
RUN cd nats-util && go install

RUN strip /go/bin/*

FROM alpine:3.10

RUN apk add -U --no-cache ca-certificates figlet

COPY --from=builder /go/bin/* /usr/local/bin/
RUN cd /usr/local/bin/ && ln -s nats-util nats-pub && ln -s nats-util nats-sub && ln -s nats-util nats-req

WORKDIR /root

COPY .profile $WORKDIR
COPY .creds $WORKDIR
COPY README.md $WORKDIR

USER root

ENTRYPOINT ["/bin/sh", "-l"]