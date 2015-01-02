FROM golang:onbuild

RUN apt-get update && apt-get install -y whois \
		--no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*
