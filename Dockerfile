# Build
FROM umputun/baseimage:buildgo-latest as build

ARG COVERALLS_TOKEN
ARG CI
ARG GIT_BRANCH

ENV GOFLAGS="-mod=vendor" GO111MODULE=on

ADD . /app
WORKDIR /app

# run tests
RUN go test -v ./...

# linters
RUN golangci-lint run --disable-all --deadline=300s --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=goconst --enable=errcheck --enable=unconvert \
    --enable=deadcode --enable=gosimple ./...

# coverage report
RUN mkdir -p target && /script/coverage.sh

# submit coverage to coverals if COVERALLS_TOKEN in env
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo "coverall not enabled" ; \
    else goveralls -coverprofile=.cover/cover.out -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi && \
    cat .cover/cover.out

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(/script/git-rev.sh); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    go build -o rlb-stats -ldflags "-X main.revision=${version}" ./app && \
    ls -la /app/rlb-stats

# Run
FROM umputun/baseimage:app-latest

RUN apk add --update ca-certificates && update-ca-certificates

COPY --from=build /app/rlb-stats /srv/
# timezone setter
COPY init.sh /srv/
ADD webapp /srv/webapp

RUN chown -R app:app /srv
USER app
WORKDIR /srv

CMD ["/srv/rlb-stats"]
ENTRYPOINT ["/init.sh", "/srv/rlb-stats"]
