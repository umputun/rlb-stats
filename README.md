# rlb-stats [![Build Status](https://github.com/umputun/rlb-stats/workflows/CI%20Build/badge.svg)](https://github.com/umputun/rlb-stats/actions?query=workflow%3A%22CI+Build%22) [![Coverage Status](https://coveralls.io/repos/github/umputun/rlb-stats/badge.svg?branch=master)](https://coveralls.io/github/umputun/rlb-stats?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/rlb-stats)](https://goreportcard.com/report/github.com/umputun/rlb-stats) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rlb-stats/)

Stats collector for [RLB](https://github.com/umputun/rlb) with REST interface and WEB UI.

## Run instructions

Run `docker-compose up` in order to start rlb-stats and rlb (for data generation).

### Generate data

Run following command in the terminal in order to generate traffic for rlb-stats:

```sh
while true; do
  curl "http://127.0.0.1:7070/api/v1/jump/test1?url=/mp3files/test_file_$(((RANDOM % 10) + 1)).mp3" >/dev/null 2>&1
  curl "http://localhost:7070/api/v1/jump/test1?url=/mp3files/test_file_$(((RANDOM % 10) + 1)).mp3" >/dev/null 2>&1
  sleep $(((RANDOM % 10) + 1))
done
```


### API

Open [http://127.0.0.1:8080/api/candle](http://127.0.0.1:8080/api/candle?from=2018-02-18T15:35:00-00:00&to=2032-02-18T15:38:00-00:00&aggregate=2m)
endpoint from to see all aggregated logs since the start of the container.

### Dashboard
Open http://127.0.0.1:8080/?from=20m URL to see dashboard with statistics

### Application parameters

| Command line   | Environment    | Default                       | Description                     |
| ---------------| ---------------| ------------------------------| ------------------------------- |
| port           | PORT           | `80`                          | Web server port                 |
| bolt           | BOLT_FILE      | `/tmp/rlb-stats.bd`           | boltdb file path                |
| dbg            | DEBUG          | `false`                       | debug mode                      |
|                | TIME_ZONE      | `America/Chicago`             | container timezone              |

## API

### Load candles

`GET /api/candle`, parameters - `?from=<RFC3339_date>&to=<RFC3339_date>&aggregate=<duration>`

Retrieve candles from storage.
- `from` (required) is the beginning of the interval, format is RFC3339, for example `2006-01-02T15:04:05+07:00`
- `to` (optional) is the end of the interval
- `aggregate` (optional) is the aggregation interval (truncated to minute), format examples are `5m`, `600s`, `1h`

`POST /api/insert`

Insert LogRecord to storage. Expects LogRecord as a body:
```json
{
	"from_ip": "172.21.0.1",
	"ts": "2021-03-24T08:20:00Z",
	"file_name": "rtfiles/rt_podcast659.mp3",
	"dest": "n3.radio-t.com"
}
```
