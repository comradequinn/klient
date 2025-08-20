#! /bin/bash

export TEST_BROKER_NAME=kafka-broker
export TEST_TOPIC="klient-test"
export TEST_TOPIC_REPLICAS="4"
export BOOTSTRAPPERS="localhost:9092"
export KLIENT=tests/bin/klient
export TEST_KEY="test-key"
export TEST_VALUE="test-data"
export TEST_HEADER_1="test-header1"
export TEST_HEADER_VALUE_1="test-value1"
export TEST_HEADER_2="test-header2"
export TEST_HEADER_VALUE_2="test-value2"

function assert_contains() {
    term="$1"
    string="$2"

    if ! printf "%s" "$string" | grep -q "$term"; then
        echo "- test assertion failed: expected '$string' to contain '$term'"
        exit 1
    fi
}

function assert_line_count() {
    expected_lines="$1"
    string="$2"
    actual_lines="$(printf "%s\n" "$string" | wc -l)"

    if [[ "$actual_lines" != "$expected_lines" ]]; then
        echo "- test assertion failed: expected '$string' to contain $expected_lines lines. got $actual_lines"
        exit 1
    fi
}

function assert_no_error() {
    exit_code="$?"
    output="$1"
    
    if [[ "$exit_code" != "0" ]]; then
        echo "- test assertion failed: expected exit code of 0. got $exit_code. command output was '$output'"
        exit 1
    fi
}

function assert_exit_code() {
    actual_exit_code="$?"
    expected_exit_code="$1"
    output="$2"
    
    if [[ "$actual_exit_code" != "$expected_exit_code" ]]; then
        echo "- test assertion failed: expected exit code of $expected_exit_code. got $actual_exit_code. command output was '$output'"
        exit 1
    fi
}

printf "%s" "building klient... "
CGO_ENABLED=0;go build -o $KLIENT
printf "%s\n" "klient built to $KLIENT"

echo "preparing kafka broker for integration tests named '$TEST_BROKER_NAME'..."
printf "%s" "- removing any orphaned brokers from previous tests..."
podman container rm -f $TEST_BROKER_NAME
printf "\n%s" "- creating kafka test broker... "
podman container run -d -p 9092:9092 --rm --name $TEST_BROKER_NAME docker.io/apache/kafka:latest
printf "%s\n\n" "- waiting to allow '$TEST_BROKER_NAME' to be ready..."
sleep 5s

echo "test 1: list topics, expect none"
TEST_RESULT="$($KLIENT -i -b "${BOOTSTRAPPERS}")"
assert_no_error "$TEST_RESULT"
assert_contains "NAME" "$TEST_RESULT"
assert_contains "PARTITIONS" "$TEST_RESULT"
assert_line_count "1" "$TEST_RESULT"
printf "%s\n\n" "- OK"

echo "test 2: create topic named '${TEST_TOPIC}', expect no error"
TEST_RESULT="$($KLIENT -c "${TEST_TOPIC}" -p 4 -n 1 -b "${BOOTSTRAPPERS}")"
assert_no_error "$TEST_RESULT"
printf "%s\n\n" "- OK"

echo "test 3: list topics, expect one named '${TEST_TOPIC}' with ${TEST_TOPIC_REPLICAS} replicas"
TEST_RESULT="$($KLIENT -i -b "${BOOTSTRAPPERS}")"
assert_no_error "$TEST_RESULT"
assert_contains "NAME" "$TEST_RESULT"
assert_contains "PARTITIONS" "$TEST_RESULT"
assert_contains "${TEST_TOPIC}" "$TEST_RESULT"
assert_contains "${TEST_TOPIC_REPLICAS}" "$TEST_RESULT"
assert_line_count "2" "$TEST_RESULT"
printf "%s\n\n" "- OK"

echo "test 4: write message to '${TEST_TOPIC}' topic, expect no error"
TEST_RESULT="$($KLIENT -w "${TEST_TOPIC}" -k "${TEST_KEY}" -v "${TEST_VALUE}" -h "${TEST_HEADER_1}=${TEST_HEADER_VALUE_1},${TEST_HEADER_2}=${TEST_HEADER_VALUE_2}" -b "${BOOTSTRAPPERS}")"
assert_no_error "$TEST_RESULT"
assert_contains "data written to topic" "$TEST_RESULT"
printf "%s\n\n" "- OK"

echo "test 5: read from '${TEST_TOPIC}' topic, expect message from test 4"
TEST_RESULT="$(timeout 10s $KLIENT -r "${TEST_TOPIC}" -a -b "${BOOTSTRAPPERS}")"
assert_exit_code "124" "$TEST_RESULT" # 124 = timeout
assert_contains "reading from topic" "$TEST_RESULT"
assert_contains "${TEST_HEADER_1}: ${TEST_HEADER_VALUE_1}" "$TEST_RESULT"
assert_contains "${TEST_HEADER_2}: ${TEST_HEADER_VALUE_2}" "$TEST_RESULT"
assert_contains "${TEST_VALUE}" "$TEST_RESULT"
printf "%s\n\n" "- OK"

echo "test 6: delete topic named '${TEST_TOPIC}', expect no error"
TEST_RESULT="$($KLIENT -d "${TEST_TOPIC}" -b "${BOOTSTRAPPERS}")"
assert_no_error "$TEST_RESULT"
printf "%s\n\n" "- OK"

echo "test 7: list topics, expect only system topic named '__consumer_offsets'"
TEST_RESULT="$($KLIENT -i -b "${BOOTSTRAPPERS}")"
assert_no_error "$TEST_RESULT"
assert_contains "NAME" "$TEST_RESULT"
assert_contains "PARTITIONS" "$TEST_RESULT"
assert_contains "__consumer_offsets" "$TEST_RESULT"
assert_line_count "2" "$TEST_RESULT"
printf "%s\n\n" "- OK"

printf "%s" "removing test broker... "
podman container rm -f $TEST_BROKER_NAME

printf "\n%s\n"  "integration tests completed successfully"