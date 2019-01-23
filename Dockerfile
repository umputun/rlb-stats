# Build
FROM umputun/baseimage:buildgo-latest as build

ADD . /go/src/github.com/umputun/rlb-stats
WORKDIR /go/src/github.com/umputun/rlb-stats

RUN go test -v $(go list -e ./... | grep -v vendor)

RUN gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode --enable=gosimple -tests ./...

RUN /script/checkvendor.sh
RUN mkdir -p target && /script/coverage.sh

RUN go build -o rlb-stats -ldflags "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S)" ./app

# Run
FROM umputun/baseimage:micro-latest

RUN apk add --update ca-certificates && update-ca-certificates

COPY --from=build /go/src/github.com/umputun/rlb-stats/rlb-stats /srv/
ADD webapp /srv/webapp

RUN chown -R umputun:umputun /srv

USER umputun

WORKDIR /srv
CMD ["/srv/rlb-stats"]
ENTRYPOINT ["/init.sh", "/srv/rlb-stats"]
