# Build
FROM golang:1.9-alpine as build

ARG TZ=America/Chicago

RUN apk add --no-cache --update tzdata git \
 && go get -u github.com/alecthomas/gometalinter \
 && gometalinter --install \
 && cp /usr/share/zoneinfo/$TZ /etc/localtime \
 && echo $TZ > /etc/timezone
ADD . /go/src/github.com/umputun/rlb-stats
WORKDIR /go/src/github.com/umputun/rlb-stats

RUN gometalinter --disable-all --vendor --deadline=300s --enable=vet --enable=vetshadow --enable=golint  --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert --enable=deadcode  --enable=gosimple --enable=gas -tests ./app/... && \
    CGO_ENABLED=0 GOOS=linux go build -o rlb-stats -ldflags "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S)" ./app

# Run
FROM umputun/baseimage:micro-latest

RUN apk add --update ca-certificates && update-ca-certificates

COPY --from=build /go/src/github.com/umputun/rlb-stats/rlb-stats /srv/

RUN chown -R umputun:umputun /srv

USER umputun

WORKDIR /srv
ENTRYPOINT ./rlb-stats
