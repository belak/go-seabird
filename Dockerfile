# 99% of this Dockerfile is from golang:onbuild
# however, this allows us to inject all the deps
# from the local repo which makes it easier to run locally
FROM golang

RUN mkdir -p /go/src/github.com/belak/seabird

# this will ideally be built by the ONBUILD below ;)
CMD ["/go/bin/seabird"]

COPY . /go/src/github.com/belak/seabird
RUN go get -d github.com/belak/seabird \
	&& go install github.com/belak/seabird \
	&& rm -rf /go/src
