.PHONY: *

build :
	@CGO_ENABLED=0;go build -o bin/klient

stop-kafka:
	-@podman container rm -f kafka-broker

start-kafka: stop-kafka
	@podman container run -d -p 9092:9092 --rm --name kafka-broker docker.io/apache/kafka:latest

BOOTSTRAPPERS="localhost:9092"
info : build
	@bin/klient -i -b "${BOOTSTRAPPERS}"

TOPIC="klient-example-0-${USER}"
read-topic : build
	@bin/klient -r ${TOPIC} -a -b ${BOOTSTRAPPERS}

KEY="example-key"
VALUE="example-data"
HEADERS="example-header1=example-value1,example-header2=example-value2"
write-topic : build
	@bin/klient -w ${TOPIC} -k "${KEY}" -v "${VALUE}" -h "${HEADERS}" -b ${BOOTSTRAPPERS}

PARTITIONS=4
REPLICAS=1
create-topic : build
	@bin/klient -c ${TOPIC} -p ${PARTITIONS} -n ${REPLICAS} -b ${BOOTSTRAPPERS}

delete-topic : build
	-@bin/klient -d ${TOPIC} -b ${BOOTSTRAPPERS}