FROM debian:12

COPY godnsd /usr/local/bin/godnsd

ENTRYPOINT ["/usr/local/bin/godnsd", "start"]
