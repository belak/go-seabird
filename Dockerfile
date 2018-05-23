FROM golang:1.10-alpine

ENV SEABIRD_CONFIG /data/seabird.toml
VOLUME /data

RUN apk add --update iputils git
RUN go get -u golang.org/x/vgo

# Add the files and switch to that dir
ADD . /src
WORKDIR /src

RUN vgo install -v ./cmd/seabird

ENTRYPOINT ["/go/bin/seabird"]
