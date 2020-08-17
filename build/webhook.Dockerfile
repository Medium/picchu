FROM alpine:3.8

RUN apk add --no-cache tzdata

ENV OPERATOR=/usr/local/bin/picchu-webhook \
    USER_UID=1001 \
    USER_NAME=picchu

# install operator binary
COPY build/_output/bin/picchu-webhook ${OPERATOR}

COPY build/bin /usr/local/bin
RUN apk add --no-cache ca-certificates && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
