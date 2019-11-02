# rlb-stats [![Build Status](https://github.com/umputun/rlb-stats/workflows/CI%20Build/badge.svg)](https://github.com/umputun/rlb-stats/actions?query=workflow%3A%22CI+Build%22) [![Coverage Status](https://coveralls.io/repos/github/umputun/rlb-stats/badge.svg?branch=master)](https://coveralls.io/github/umputun/rlb-stats?branch=master) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rlb-stats/)

Stats collector for [RLB](https://github.com/umputun/rlb) with REST interface and WEB UI.

## Run instructions

Use `docker-compose.yml` as an example for service start, minimum requirements:

 - For web-server work
     - mount `bolt` data file/folder from host system (`/tmp` in example)
     - expose a port from container to host system (`8080:8080` in example)

- For data collection
    - on first run set `log_tail` to `all`
    - provide `container_name` as an input parameter for docker container
    - mount `bolt` data file and `docker` socket from host system

### Application parameters

| Command line   | Environment    | Default                       | Description                     |
| ---------------| ---------------| ------------------------------| ------------------------------- |
| container_name | CONTAINER_NAME |                               | container name, _required_ for data collection |
| docker         | DOCKER_HOST    | `unix:///var/run/docker.sock` | docker host                     |
| log_tail       | LOG_TAIL       | `1000`                        | how many log entries to load from container |
| regexp         | REGEXP         | `^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$` | log line regexp |
| date_format    | DATE_FORMAT    | `2006/01/02 15:04:05`         | format of the date in log line  |
| ui_port        | UI_PORT        | `80`                          | UI server port                  |
| port           | PORT           | `8080`                        | REST server port                |
| bolt           | BOLT_FILE      | `/tmp/rlb-stats.bd`           | boltdb file path                |
| dbg            | DEBUG          | `false`                       | debug mode                      |
|                | TIME_ZONE      | `America/Chicago`             | container timezone              |

## How to set up test stand

1. Uncomment `environment` section and `user` line in `docker-compose.yml`: it will result in container listening to it's own HTTP access logs
1. Run `docker-compose up -d` in order to start rlb-stats
1. API: Open [http://127.0.0.1:8080/api/candle](http://127.0.0.1:8080/api/candle?from=2018-02-18T15:35:00-00:00&to=2032-02-18T15:38:00-00:00&aggregate=2m)
endpoint from example below to see all aggregated logs since the start of the container
(would empty for a minute after you open this page for a first time)
1. Dashboard: Open http://127.0.0.1/?from=20m URL to see dashboard with statistics


## API

### Load candles

`GET /api/candle`, parameters - `?from=<RFC3339_date>&to=<RFC3339_date>&aggregate=<duration>`

- `from` (required) is the beginning of the interval, format example is `2006-01-02T15:04:05+07:00`
- `to` (optional) is the end of the interval
- `aggregate` (optional) is the aggregation interval (truncated to minute), format examples are `5m`, `600s`, `1h`

#### Example

##### API calls

<details>
<summary>api/candle</summary>

```json
$ http GET http://127.0.0.1:8080/api/candle?from=2018-02-18T15:35:00-00:00&to=2032-02-18T15:38:00-00:00&aggregate=2m

HTTP/1.1 200 OK
Content-Type: application/json

[
  {
    "Nodes": {
      "n6.radio-t.com": {
        "Volume": 1,
        "MinAnswerTime": 1,
        "MeanAnswerTime": 1,
        "MaxAnswerTime": 1,
        "Files": {
          "rt_podcast585.mp3": 1
        }
      },
      "n7.radio-t.com": {
        "Volume": 1,
        "MinAnswerTime": 2,
        "MeanAnswerTime": 2,
        "MaxAnswerTime": 2,
        "Files": {
          "rt_podcast584.mp3": 1,
        }
      },
      "all": {
        "Volume": 2,
        "MinAnswerTime": 1,
        "MeanAnswerTime": 1.5,
        "MaxAnswerTime": 2,
        "Files": {
          "rt_podcast584.mp3": 1,
          "rt_podcast585.mp3": 1
        }
      }
    },
    "StartMinute": "2018-02-18T15:35:00Z"
  },
  {
    "Nodes": {
      "n6.radio-t.com": {
        "Volume": 5,
        "MinAnswerTime": 1,
        "MeanAnswerTime": 1,
        "MaxAnswerTime": 1,
        "Files": {
          "rt_podcast579.mp3": 1,
          "rt_podcast581.mp3": 1,
          "rt_podcast583.mp3": 1,
          "rt_podcast584.mp3": 1,
          "rt_podcast585.mp3": 1
        }
      },
      "all": {
        "Volume": 5,
        "MinAnswerTime": 1,
        "MeanAnswerTime": 1,
        "MaxAnswerTime": 1,
        "Files": {
          "rt_podcast579.mp3": 1,
          "rt_podcast581.mp3": 1,
          "rt_podcast583.mp3": 1,
          "rt_podcast584.mp3": 1,
          "rt_podcast585.mp3": 1
        }
      }
    },
    "StartMinute": "2018-02-18T15:37:00Z"
  }
]
```

</details>
