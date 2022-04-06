# Build
FROM umputun/baseimage:buildgo-latest as build

ARG CI
ARG GIT_BRANCH
ARG SKIP_TEST
ARG GITHUB_SHA

ENV GOFLAGS="-mod=vendor" GO111MODULE=on

ADD . /app
WORKDIR /app

# run tests
RUN \
    if [ -z "$SKIP_TEST" ] ; then \
        go test -timeout=30s  ./... && \
        golangci-lint run --config ./.golangci.yml ./... ; \
    else echo "skip tests and linter" ; fi


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
ADD webapp /srv/webapp

RUN chown -R app:app /srv
USER app
WORKDIR /srv

CMD ["/srv/rlb-stats"]
ENTRYPOINT ["/init.sh", "/srv/rlb-stats"]
