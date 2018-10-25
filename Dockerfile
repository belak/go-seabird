FROM golang:1.11-alpine

# Add any runtime dependencies
RUN apk add -U --no-cache iputils

# Copy the built seabird into the container
ADD dist/seabird /bin/seabird
ADD dist/seabird-migrate /bin/seabird-migrate

VOLUME /data
ENV SEABIRD_CONFIG /data/seabird.toml

ENTRYPOINT ["/bin/seabird"]
