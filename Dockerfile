# Build
FROM umputun/baseimage:buildgo as build

ADD . /app
WORKDIR /app
ENV GOFLAGS="-mod=vendor" GO111MODULE=on

RUN go test -v ./...

RUN golangci-lint run --disable-all --deadline=300s --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode --enable=gosimple ./...

RUN mkdir -p target && /script/coverage.sh

RUN go build -o rlb-stats -ldflags "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S)" ./app

# Run
FROM umputun/baseimage:app

RUN apk add --update ca-certificates && update-ca-certificates

COPY --from=build /app/rlb-stats /srv/
ADD webapp /srv/webapp

RUN chown -R app:app /srv

USER app

WORKDIR /srv
CMD ["/srv/rlb-stats"]
ENTRYPOINT ["/init.sh", "/srv/rlb-stats"]
