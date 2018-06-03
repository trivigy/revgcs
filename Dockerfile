FROM golang:alpine AS build-env

ARG pkg=revgcs

COPY . $GOPATH/src/$pkg

RUN set -ex \
      && apk add --no-cache --virtual .build-deps \
              git \
              ca-certificates \
      && go get -v $pkg/... \
      && apk del .build-deps

RUN go install $pkg/...

FROM alpine
RUN set -ex \
      && apk add --no-cache ca-certificates

COPY --from=build-env /go/bin/revgcs /usr/bin/

CMD echo "Use the revgcs commands."; exit 1