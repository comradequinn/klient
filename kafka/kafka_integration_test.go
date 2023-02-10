// Integration tests for the kakfa client. A kafka instance running locally @ 9092 is required .
// Such an instance can be started using the dockercompose.yaml file in the repo's root directory, or by running 'make integration-test'
package kafka

import (
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testInstanceBootstrapper = "localhost:9092"

func TestMain(m *testing.M) {
	var (
		kafkaReadySig = make(chan struct{}, 1)
		kafkaReady    = false
		timeout       = time.After(time.Minute * 6) // allow for slow CI image pulls
		err           error
	)

	go func() {
		k := connect(log.Fatalf)

		for {
			if _, err = k.Describe(); err == nil {
				break
			}
			<-time.After(time.Second)
		}

		kafkaReadySig <- struct{}{}
	}()

	for !kafkaReady {
		select {
		case <-kafkaReadySig:
			kafkaReady = true
			log.Printf("test kafka instance ready")
		case <-timeout:
			log.Fatalf("timed out waiting for test kafka instance to respond. ensure it is running and can be bootstrapped from %v.last error was: %v", testInstanceBootstrapper, err)
		}
	}

	os.Exit(m.Run())
}

func TestIntegration(t *testing.T) {
	check := func(err error, format string, args ...any) {
		if err != nil {
			args = append(args, err)
			t.Fatalf(format, args...)
		}
	}

	k, topic, partitions, key1, val1, key2, val2 := connect(t.Fatalf), "test-topic", 1, "k1", "v1", "k2", "v2"

	{ // create topic
		check(k.CreateTopic(topic, partitions, 1), "expected no error creating a topic. got %v")

		info := brokerInfo(t)

		if info == nil || len(info.Topics) != 1 || len(info.Topics[topic].Partitions) != partitions {
			t.Fatalf("expected a topic to exist and have the specified partitions after creating it. got %+v", info)
		}
	}

	{ // produce to topic
		writeFunc, closeFunc, err := k.Writer(topic, false)

		check(err, "expected no error creating a topic writer. got %+v")

		check(writeFunc([]byte(key1), []byte(val1)), "expected no error writing %q. got %v", key1)
		check(writeFunc([]byte(key2), []byte(val2)), "expected no error writing %q. got %v", key2)
		check(closeFunc(), "expected no error closing the writer. got %v")
	}

	{ // consume from topic
		msg1 := true
		read := func(m Message) bool {
			key, val := key2, val2

			if msg1 {
				key, val = key1, val1
			}

			if !(string(m.Key) == key && string(m.Value) == val) {
				t.Fatalf("expected to receive written data. got %s: %s", m.Key, m.Value)
			}

			msg1 = !msg1
			return !msg1
		}

		{ // consume topic exclusively
			check(k.ReadExclusive(topic, read), "expected no error reading exclusively. got %v")
		}

		{ // consume topic in group
			check(k.ReadGroup(topic, "test-group", read), "expected no error reading as part of group. got %v")
		}

		{ // consume topic in group
			check(k.ReadRange(topic, 0, addressFor(int64(1)), 0, read), "expected no error reading an offset range. got %v")
		}

		{ // consume topic in group
			check(k.ReadTime(topic, time.Now().Add(-time.Minute), addressFor(time.Now()), 0, read), "expected no error reading an offset range. got %v")
		}
	}

	{ // delete topic
		if err := k.DeleteTopic(topic); err != nil {
			t.Fatalf("expected no error deleting a topic. got %v", err)
		}

		info := brokerInfo(t)

		if info != nil && info.Topics[topic] != nil {
			t.Fatalf("expected a topic to no longer exist after deleting it. got %+v", info)
		}
	}
}

func brokerInfo(t *testing.T) *Broker {
	k := connect(t.Fatalf)

	brokers, err := k.Describe()

	if err != nil {
		t.Fatalf("expected no error describing cluster. got %v", err)
	}

	if len(brokers) > 1 {
		t.Fatalf("expected a single broker test instance. got %+v", brokers)
	}

	for _, broker := range brokers {
		if broker.Host != strings.Split(testInstanceBootstrapper, ":")[0] || strconv.Itoa(broker.Port) != strings.Split(testInstanceBootstrapper, ":")[1] {
			t.Fatalf("expected described broker to refer to %v. got %+v", testInstanceBootstrapper, *broker)
		}
		return broker
	}

	return nil
}

func connect(fatalfFunc func(format string, args ...any)) *Kafka {
	var (
		k   *Kafka
		err error
	)

	if k, err = New(ConnectConfig{Timeout: time.Second}, testInstanceBootstrapper); err != nil {
		fatalfFunc("unable to configure connection credentials for test kafka instance at " + testInstanceBootstrapper)
	}

	return k
}

func addressFor[T any](v T) *T {
	return &v
}
