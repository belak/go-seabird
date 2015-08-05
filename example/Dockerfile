FROM golang:1.4-wheezy

RUN apt-get update && apt-get install -y whois traceroute \
		--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

COPY . /go/src/app
RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run"]
