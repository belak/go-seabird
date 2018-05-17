FROM golang:1.10-alpine

ENV SEABIRD_CONFIG /data/seabird.toml
VOLUME /data

RUN apk add --update libcap iputils git

ADD . /go/src/github.com/belak/go-seabird

RUN go get -v -d github.com/belak/go-seabird/cmd/seabird
RUN go install github.com/belak/go-seabird/cmd/seabird

RUN setcap cap_net_raw=+ep /go/bin/seabird

ENTRYPOINT ["/go/bin/seabird"]
