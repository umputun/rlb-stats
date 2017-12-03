# Build
FROM golang:1.9-alpine as build

RUN apk add --no-cache --update tzdata git && cp /usr/share/zoneinfo/America/Chicago /etc/localtime 
ADD . /go/src/github.com/umputun/rlb-stats
WORKDIR /go/src/github.com/umputun/rlb-stats

RUN go test ./app/... && \
    CGO_ENABLED=0 GOOS=linux go build -o rlb-stats -ldflags "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S)" ./app

# Run
FROM umputun/baseimage:micro-latest

RUN apk add --update ca-certificates && update-ca-certificates

COPY --from=build /go/src/github.com/umputun/rlb-stats/rlb-stats /srv/

RUN chown -R umputun:umputun /srv

USER umputun

WORKDIR /srv
ENTRYPOINT ./rlb-stats
