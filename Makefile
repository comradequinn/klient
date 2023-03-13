.PHONY: *

local-kafka :
	@docker compose up

stop-local-kafka : 
	@docker compose down

test :
	@go test -count=1 -cover ./cli/...

integration-test :
	@printf "starting kafka test instance...\n"
	@docker compose down &> /dev/null
	@docker compose up &> /dev/null &
	@go test -count=1 -cover ./kafka/...
	@printf "stopping kafka test instance... "
	@docker compose down &> /dev/null
	@printf "stopped\n"

VERSION=1.3.2.`date +%s`
build :
	-@rm -rf ./bin 2> /dev/null
	-@mkdir ./bin
	@export CGO_ENABLED=0; go build -ldflags "-X main.versionNumber=${VERSION}" -o ./bin/klient

release: build
	@./scripts/release.sh ${VERSION}

install : build
	-@sudo rm -f /usr/local/bin/klient 2> /dev/null
	@sudo cp ./bin/klient /usr/local/bin/ && sudo chmod +x /usr/local/bin/klient

help : build
	@./bin/klient -h

#AUTH=-bootstrappers "kafka-broker-1" -authkey "username" -authsecret "password" -tls -tlsnoverify # the format of the auth string
AUTH= # anonymous auth string
#UNATTENDED=-unattended
UNATTENDED=

describe : build
	@./bin/klient -describe ${AUTH}

describe-unattended : build
	@./bin/klient -describe -unattended ${AUTH}

TOPIC="example"
PARTITIONS="6"
REPLICAS="1"
topic: build
	@./bin/klient -create ${TOPIC} -partitions ${PARTITIONS} -replicas ${REPLICAS} ${AUTH}

delete-topic: build
	@./bin/klient -delete ${TOPIC} ${AUTH}

producer : build
	@./bin/klient -write ${TOPIC} -keyed ${AUTH}

stdin-producer : build
	@cat testdata/input.json | ./bin/klient -write ${TOPIC} -unattended ${AUTH}

OFFSET_FROM="0"
OFFSET_TO="-1"
PARTITION="0"
range-consumer : build
	@./bin/klient -range-read ${TOPIC} -offset-from ${OFFSET_FROM} -offset-to ${OFFSET_TO} -partition ${PARTITION} ${AUTH}

TIME_FROM=""
TIME_TO=""
time-consumer : build
	@./bin/klient -time-read ${TOPIC} -time-from ${TIME_FROM} -time-to ${TIME_TO} ${AUTH}

exclusive-consumer : build
	@./bin/klient -exclusive-read ${TOPIC} ${AUTH}

CONSUMER_GROUP="example-consumers"
group-consumer : build
	@./bin/klient -group-read ${TOPIC} -group ${CONSUMER_GROUP} ${AUTH}

stdout-consumer : build
	@./bin/klient -exclusive-read "${TOPIC}" -unattended ${AUTH} 2> /dev/null | jq
