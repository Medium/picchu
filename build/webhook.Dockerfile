FROM golang:1.17

WORKDIR /go/src/go.medium.engineering/picchu

COPY pkg ./pkg
COPY cmd ./cmd
COPY version ./version

COPY go.mod go.sum ./

RUN ls

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook ./cmd/webhook

FROM alpine:3.8

RUN apk add --no-cache tzdata

ENV OPERATOR=/usr/local/bin/picchu-webhook \
    USER_UID=1001 \
    USER_NAME=picchu

# install operator binary
COPY --from=0 /go/src/go.medium.engineering/picchu/webhook ${OPERATOR}

COPY build/bin /usr/local/bin
RUN apk add --no-cache ca-certificates && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
