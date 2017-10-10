FROM scratch
MAINTAINER Jamie Hannaford <jamie@limetree.org>

ADD . /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/canary"]
