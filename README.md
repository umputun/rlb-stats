# rlb-stats [![Build Status](https://github.com/umputun/rlb-stats/workflows/CI%20Build/badge.svg)](https://github.com/umputun/rlb-stats/actions?query=workflow%3A%22CI+Build%22) [![Coverage Status](https://coveralls.io/repos/github/umputun/rlb-stats/badge.svg?branch=master)](https://coveralls.io/github/umputun/rlb-stats?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/rlb-stats)](https://goreportcard.com/report/github.com/umputun/rlb-stats) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rlb-stats/)

Download stats collector with REST interface and WEB UI. Two ingest modes:

- **Nginx tail mode**: tails an nginx access log, parses each line, counts downloads and bandwidth per vhost, file and country.
- **RLB mode** (legacy): accepts individual `POST /api/insert` payloads from [RLB](https://github.com/umputun/rlb). Still supported for existing setups.

## Nginx tail mode

Configure nginx with the required `log_format` (name it anything you like — this example uses `rlbstats`):

```nginx
log_format rlbstats '$remote_addr - - [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_host" '
                    '"$http_referer" "$http_user_agent" $scheme '
                    '"$http_cf_connecting_ip" "$http_cf_ipcountry" "$http_range"';

access_log /var/log/nginx/access.log rlbstats;
```

Then point rlb-stats at the log file:

```sh
docker run --rm \
    -p 8080:8080 \
    -v /var/log/nginx/access.log:/var/log/nginx/access.log:ro \
    -v /tmp:/tmp \
    -e NGINX_LOG=/var/log/nginx/access.log \
    umputun/rlb-stats:latest
```

Open http://127.0.0.1:8080/ for the dashboard.

**Field notes:**

- `$http_cf_connecting_ip` is Cloudflare's real-client-IP header. Without CF in front it stays empty and rlb-stats falls back to `$remote_addr`. Behind CF this field is essential — otherwise every "client IP" is a Cloudflare edge IP and per-IP dedup breaks.
- `$http_cf_ipcountry` is Cloudflare's two-letter country code. Requires CF "IP Geolocation" to be enabled (default on all plans). Without CF the top-countries panel stays empty.
- `$http_range` is logged but not yet used; it is included so the format does not need another change when probe-vs-play analysis is added later.

**Known behaviour change vs RLB mode:** one RLB redirect used to be one download regardless of play duration. Tailing nginx, a browser `<audio>` streaming for an hour produces up to ~60 unique file+IP+minute pairs = up to 60 "downloads". Apple Podcasts fetches in a few large ranges up front and inflates less. Historical RLB-era numbers are **not comparable** to nginx-tail-mode numbers.

### Dashboard

Server-side rendered using HTMX, Picocss and ECharts. Loads JS/CSS from `cdn.jsdelivr.net`.

- Summary cards: total downloads for 1 hour, 24 hours, 1 week, 1 month and all time, with total bandwidth
- Downloads histogram (ECharts bar chart) with selectable periods (1h, 12h, 24h, 10d, 30d, all)
- Top files with download count and bandwidth
- Per-vhost (node) breakdown with downloads and bandwidth
- Top countries (nginx tail mode only, requires Cloudflare)
- Peak hours heatmap (24h x 7 days)

Period switching uses HTMX without full page reloads. Dark mode is handled by Picocss.

The UI is compiled into the binary, so no files are needed on disk. To alter it without
rebuilding, put a copy of [app/web/templates](app/web/templates) and
[app/web/static](app/web/static) into a directory and point `--webapp` at it, for example
`docker run -v ./my-ui:/srv/webapp ...`, since the container's default is `/srv/webapp`.
Each subdirectory is optional: static files fall back to the embedded copy one by one,
and the templates are taken from the directory only if the whole set parses, since they
reference each other. Symlinks leading out of the directory are not followed.

## RLB mode (legacy)

If you still run RLB in front, point its `STATS` env var at `http://rlb-stats:8080/api/insert` as before. No code change on the RLB side is needed.

## API

### `GET /api/candle`

Parameters: `?from=<RFC3339_date>&to=<RFC3339_date>&aggregate=<duration>`

- `from` (required) is the beginning of the interval, format is RFC3339, for example `2026-01-02T15:04:05+07:00`
- `to` (optional) is the end of the interval
- `max_points` (optional, default `100`) unsigned integer, sets aggregate interval to return not more than specified amount of candles; ignored when `aggregate` is set
- `aggregate` (optional, overwrites `max_points`) is the aggregation interval (must be positive, truncated to minute), format examples are `5m`, `600s`, `1h`

### `POST /api/insert`

Still supported for RLB. Expects:

```json
{
    "from_ip": "172.21.0.1",
    "ts": "2026-03-24T08:20:00Z",
    "file_name": "rtfiles/rt_podcast659.mp3",
    "dest": "n3.radio-t.com"
}
```

The multi-value `--nginx-*` flags take one value per occurrence on the command line (`--nginx-status=200 --nginx-status=206`); their matching environment variables accept comma-separated values (`NGINX_STATUS=200,206`).

## Parameters

| Command line     | Environment    | Default              | Description                                           |
|------------------|----------------|----------------------|-------------------------------------------------------|
| `--port`         | `PORT`         | `8080`               | Web server port                                       |
| `--bolt`         | `BOLT_FILE`    | `/tmp/rlb-stats.bd`  | boltdb file path                                      |
| `--webapp`       | `WEBAPP`       | `webapp`             | override embedded UI directory (used if present)      |
| `--nginx-log`    | `NGINX_LOG`    | *(empty)*            | nginx access log to tail; empty = feature disabled    |
| `--nginx-method` | `NGINX_METHOD` | `GET`                | allowed HTTP methods                                  |
| `--nginx-status` | `NGINX_STATUS` | `200,206`            | allowed status codes                                  |
| `--nginx-path`   | `NGINX_PATH`   | `*.mp3`              | path globs matched on basename                        |
| `--nginx-host`   | `NGINX_HOST`   | *(empty = all)*      | nginx `$http_host` allowlist                          |
| `--dbg`          | `DEBUG`        | `false`              | debug mode (adds parse-skip DEBUG logs)               |
|                  | `TIME_ZONE`    | `America/Chicago`    | container timezone                                    |

## Running from source

```sh
go build -o rlb-stats ./app
./rlb-stats --nginx-log /var/log/nginx/access.log --port 8080
```

Or with docker-compose:

```sh
docker-compose up
```

Edit `docker-compose.yml` to uncomment the `NGINX_LOG` env var and the access-log volume mount.
