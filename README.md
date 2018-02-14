# rlb-stats  [![Build Status](http://drone.umputun.com/api/badges/umputun/rlb-stats/status.svg)](http://drone.umputun.com/umputun/rlb-stats)

Stats collector for RLB with REST interface, written in Go.

## Run instructions

Use `docker-compose.yml` as an example for service start, minimum requirements:
 
 - For web-server work
     - mount `bolt` data file/folder from host system (`/tmp` in example)
     - expose a port from container to host system (`8080:8080` in example)
 
- For data collection
    - on first run set `log_tail` to `all`
    - provide `container_name` as an input parameter for docker container
    - mount `bolt` data file and `docker` socket from host system
    
    
## Application parameters

### Input parameters

       --container_name= container name [$CONTAINER_NAME]
       --docker=         docker host (default: unix:///var/run/docker.sock) [$DOCKER_HOST]
       --log_tail=       How many log entries to load from container, set to 'all' on the first run (default: 0) [$LOG_TAIL]
       --regexp=         log line regexp (default: ^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$) [$REGEXP]
       --date_format=    format of the date in log line (default: 2006/01/02 15:04:05) [$DATE_FORMAT]

### Output parameters

       --bolt=           boltdb file (default: /tmp/rlb-stats.bd) [$BOLT_FILE]
       --port=           REST server port (default: 8080) [$PORT]
       --dbg             debug mode