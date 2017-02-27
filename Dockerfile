FROM golang:1.7

ENV SEABIRD_CONFIG /data/seabird.toml
VOLUME /data

ADD . /go/src/github.com/belak/go-seabird

RUN go get -v -d github.com/belak/go-seabird/cmd/seabird
RUN go install github.com/belak/go-seabird/cmd/seabird

ENTRYPOINT ["/go/bin/seabird"]
