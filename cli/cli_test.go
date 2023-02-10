package cli

import (
	"fmt"
	"klient/kafka"
	"strings"
	"testing"
	"time"
)

const dateFormat = "02-01-2006 15:04:05"

func TestVersion(t *testing.T) {
	cli, sb := New(dateFormat), strings.Builder{}
	cli.writeFunc = sbWriter(&sb)

	v := "v0.0.0"
	cli.Version(v)

	if !strings.Contains(sb.String(), v) {
		t.Fatalf("expected version string of %v. got %v", sb.String(), v)
	}
}

type KakfaTopicCreatorMock func(topic string, partitions, replicas int) error

func (f KakfaTopicCreatorMock) CreateTopic(topic string, partitions, replicas int) error {
	return f(topic, partitions, replicas)
}

func TestCreateTopic(t *testing.T) {
	cli, sb := New(dateFormat), strings.Builder{}
	cli.writeFunc = sbWriter(&sb)

	topic, partitions, replicas, invoked, err := "test-topic", 1, 2, false, fmt.Errorf("test-error")

	actualErr := cli.CreateTopic(KakfaTopicCreatorMock(func(tt string, p, r int) error {
		invoked = true
		if tt != topic || p != partitions || r != replicas {
			t.Fatalf("expected topic %s, partitions %d, replicas %d to be passed. got %s, %d, %d", topic, partitions, replicas, tt, p, r)
		}
		return err
	}), topic, partitions, replicas)

	if !invoked {
		t.Fatal("expected kakfa-topic-creator to be invoked")
	}

	if actualErr != err {
		t.Fatal("expected an error to be returned")
	}

	output := sb.String()

	if !strings.Contains(output, "unable to create") {
		t.Fatalf("expected output denote failure. got  %v", output)
	}

	sb.Reset()
	err = nil

	actualErr = cli.CreateTopic(KakfaTopicCreatorMock(func(tt string, p, r int) error {
		return err
	}), topic, partitions, replicas)

	if actualErr != nil {
		t.Fatal("expected no error")
	}

	output = sb.String()

	if !strings.Contains(output, "created topic") {
		t.Fatalf("expected output denote success. got  %v", output)
	}
}

type KakfaTopicDeleterMock func(topic string) error

func (f KakfaTopicDeleterMock) DeleteTopic(topic string) error {
	return f(topic)
}

func TestDeleteTopic(t *testing.T) {
	cli, sb := New(dateFormat), strings.Builder{}
	cli.writeFunc = sbWriter(&sb)

	topic, invoked, err := "test-topic", false, fmt.Errorf("test-error")

	actualErr := cli.DeleteTopic(KakfaTopicDeleterMock(func(tt string) error {
		invoked = true
		if tt != topic {
			t.Fatalf("expected topic %s to be passed. got %s", topic, tt)
		}
		return err
	}), topic)

	if !invoked {
		t.Fatal("expected kakfa-topic-deleter to be invoked")
	}

	if actualErr != err {
		t.Fatal("expected an error to be returned")
	}

	output := sb.String()

	if !strings.Contains(output, "unable to delete") {
		t.Fatalf("expected output denote failure. got  %v", output)
	}

	sb.Reset()
	err = nil

	actualErr = cli.DeleteTopic(KakfaTopicDeleterMock(func(tt string) error {
		return err
	}), topic)

	if actualErr != nil {
		t.Fatal("expected no error")
	}

	output = sb.String()

	if !strings.Contains(output, "deleted topic") {
		t.Fatalf("expected output denote success. got  %v", output)
	}

}

type KakfaDescriberMock func() (map[string]*kafka.Broker, error)

func (f KakfaDescriberMock) Describe() (map[string]*kafka.Broker, error) {
	return f()
}
func (f KakfaDescriberMock) Bootstrappers() []string {
	brokers, _ := f.Describe()
	bootstrappers, i := make([]string, len(brokers)), 0

	for bootstrapper := range brokers {
		bootstrappers[i] = bootstrapper
		i++
	}

	return bootstrappers
}

func TestDescriber(t *testing.T) {
	cli, sb := New(dateFormat), strings.Builder{}
	cli.writeFunc = sbWriter(&sb)

	host, port, topic, partition := "testhost", 9092, "test-topic", 0

	brokers := map[string]*kafka.Broker{
		host: {
			Host: host,
			Port: port,
			Topics: map[string]*kafka.Topic{
				topic: {
					Name: topic,
					Partitions: map[int]*kafka.Partition{
						partition: {
							ID: partition,
						}}}}},
	}

	cli.Describe(KakfaDescriberMock(func() (map[string]*kafka.Broker, error) {
		return brokers, nil
	}))

	output := sb.String()

	if !strings.Contains(output, fmt.Sprintf("describing cluster using bootstrappers: '%v'", host)) {
		t.Fatalf("expected output to contain host %v. got %v", host, output)
	}

	if !strings.Contains(output, fmt.Sprintf("broker: %v:%v", host, port)) {
		t.Fatalf("expected output to contain host %v and port %v. got %v", host, port, output)
	}
	if !strings.Contains(output, fmt.Sprintf("topic: %v", topic)) {
		t.Fatalf("expected output to contain topic %v. got %v", topic, output)
	}
	if !strings.Contains(output, fmt.Sprintf("partition: %v", partition)) {
		t.Fatalf("expected output to contain partition %v. got %v", partition, output)
	}
}

type KakfaExclusiveReaderMock func(topic string, readFunc func(m kafka.Message) bool) error

func (f KakfaExclusiveReaderMock) ReadExclusive(topic string, readFunc func(m kafka.Message) bool) error {
	return f(topic, readFunc)
}

type KakfaWriterMock func(topic string, keyed bool) (writeFunc func(key, value []byte) error, closeFunc func() error, err error)

func (f KakfaWriterMock) Writer(topic string, keyed bool) (writeFunc func(key, value []byte) error, closeFunc func() error, err error) {
	return f(topic, keyed)
}

func TestWrite(t *testing.T) {
	cli, sb := New(dateFormat), strings.Builder{}
	cli.writeFunc = sbWriter(&sb)

	key, value, topic, delimiter := []byte("k1"), []byte("v1"), "test-topic", byte('|')

	act := func(keyed, unattended bool, assertOutput func(string)) {
		sb.Reset()
		closed := false
		written := false
		mockedKafkaWriter := KakfaWriterMock(func(topic string, keyed bool) (writeFunc func(key, value []byte) error, closeFunc func() error, err error) {
			writeFunc = func(k, v []byte) error {
				written = true

				expectedKey := ""

				if keyed {
					expectedKey = string(key)
				}

				if string(k) != expectedKey {
					t.Fatalf("expected key %q. got %q", key, k)
				}

				if string(v) != string(value) {
					t.Fatalf("expected value %q. got %q", value, v)
				}

				return nil
			}
			closeFunc = func() error {
				closed = true
				return nil
			}
			return
		})

		cli.Write(mockedKafkaWriter, topic, keyed, unattended, delimiter)

		if !written {
			t.Fatalf("expected write func to be called")
		}

		if !closed {
			t.Fatalf("expected close func to be called")
		}

		assertOutput(sb.String())
	}

	cli.reader = strings.NewReader(string(value) + string(delimiter))
	act(false, true, func(s string) {
		if s != "" {
			t.Fatalf("expected no output in unattended mode. got %q", s)
		}
	})

	cli.reader = strings.NewReader(string(value) + string(delimiter) + string(key) + string(delimiter))
	act(true, true, func(s string) {
		if s != "" {
			t.Fatalf("expected no output in unattended mode. got %q", s)
		}
	})

	cli.reader = strings.NewReader(string(value) + "\n")
	act(false, false, func(s string) {
		if !strings.Contains(s, "enter data") || strings.Contains(s, "enter key ") {
			t.Fatalf("expected prompts values but not keys. got %q", s)
		}
	})

	cli.reader = strings.NewReader(string(value) + "\n" + string(key) + "\n")
	act(true, false, func(s string) {
		if !strings.Contains(s, "enter data") || !strings.Contains(s, "enter key ") {
			t.Fatalf("expected prompts for keys and values. got %q", s)
		}
	})
}
func TestRead(t *testing.T) {
	cli, sb := New(dateFormat), strings.Builder{}
	cli.writeFunc = sbWriter(&sb)

	ts, _ := time.Parse(dateFormat, "16-02-2023 20:00:00")
	msg := kafka.Message{
		Partition: 1,
		Offset:    2,
		Key:       []byte("k1"),
		Value:     []byte("val"),
		TimeStamp: ts,
	}

	cli.read(msg, false, '|')

	output, expected := sb.String(), "[1/2 @ 16-02-2023 20:00:00]> k1 : val\n"

	if output != expected {
		t.Fatalf("expected %q. got %q", expected, output)
	}

	sb.Reset()

	cli.read(msg, true, '|')

	output, expected = sb.String(), "val|"

	if output != expected {
		t.Fatalf("expected %q in unattended mode. got %q", expected, output)
	}

}
func TestReadExclusive(t *testing.T) {
	cli := New(dateFormat)

	topic := "test-topic"

	act := func(unattended bool, err error, assertOutput func(output string)) {
		sb, invoked := strings.Builder{}, false
		cli.writeFunc = sbWriter(&sb)

		actualErr := cli.ReadExclusive(KakfaExclusiveReaderMock(func(tt string, rf func(m kafka.Message) bool) error {
			invoked = true

			if topic != tt {
				t.Fatalf("expected topic %s to be passed. got %s", topic, tt)
			}

			return err
		}), topic, unattended, '\r')

		if !invoked {
			t.Fatal("expected kakfa-time-reader to be invoked")
		}

		if actualErr != err {
			t.Fatal("expected an error to be returned")
		}

		assertOutput(sb.String())
	}

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading exclusively from topic") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading exclusively from topic") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})
}

type KakfaGroupReaderMock func(topic, group string, readFunc func(m kafka.Message) bool) error

func (f KakfaGroupReaderMock) ReadGroup(topic, group string, readFunc func(m kafka.Message) bool) error {
	return f(topic, group, readFunc)
}

func TestReadGroup(t *testing.T) {
	cli := New(dateFormat)

	topic, group := "test-topic", "test-group"

	act := func(unattended bool, err error, assertOutput func(output string)) {
		sb, invoked := strings.Builder{}, false
		cli.writeFunc = sbWriter(&sb)

		actualErr := cli.ReadGroup(KakfaGroupReaderMock(func(tt, g string, rf func(m kafka.Message) bool) error {
			invoked = true

			if topic != tt || group != g {
				t.Fatalf("expected topic %s, group %s to be passed. got %s, %s", topic, group, tt, g)
			}

			return err
		}), topic, group, unattended, '\r')

		if !invoked {
			t.Fatal("expected kakfa-time-reader to be invoked")
		}

		if actualErr != err {
			t.Fatal("expected an error to be returned")
		}

		assertOutput(sb.String())
	}

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading from topic") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading from topic") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})
}

type KakfaTimeReaderMock func(topic string, from time.Time, to *time.Time, partition int, readFunc func(m kafka.Message) bool) error

func (f KakfaTimeReaderMock) ReadTime(topic string, from time.Time, to *time.Time, partition int, readFunc func(m kafka.Message) bool) error {
	return f(topic, from, to, partition, readFunc)
}

func TestReadTime(t *testing.T) {
	cli := New(dateFormat)

	topic, from, to, partition := "test-topic", time.Now().Add(-time.Hour), time.Now(), 3

	act := func(unattended bool, err error, assertOutput func(output string)) {
		sb, invoked := strings.Builder{}, false
		cli.writeFunc = sbWriter(&sb)

		actualErr := cli.ReadTime(KakfaTimeReaderMock(func(tt string, f time.Time, ttt *time.Time, p int, rf func(m kafka.Message) bool) error {
			invoked = true

			if topic != tt || from != f || to != *ttt || partition != p {
				t.Fatalf("expected topic %s, from %s, to %s, partition %d to be passed. got %s, %s, %s, %d", topic, from, to, partition, tt, f, ttt, p)
			}

			return err
		}), topic, from, &to, partition, unattended, '\r')

		if !invoked {
			t.Fatal("expected kakfa-time-reader to be invoked")
		}

		if actualErr != err {
			t.Fatal("expected an error to be returned")
		}

		assertOutput(sb.String())
	}

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading from partition") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading from partition") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})
}

type KakfaRangeReaderMock func(topic string, from int64, to *int64, partition int, readFunc func(m kafka.Message) bool) error

func (f KakfaRangeReaderMock) ReadRange(topic string, from int64, to *int64, partition int, readFunc func(m kafka.Message) bool) error {
	return f(topic, from, to, partition, readFunc)
}

func TestReadRange(t *testing.T) {
	cli := New(dateFormat)

	topic, from, to, partition := "test-topic", int64(1), int64(2), 3

	act := func(unattended bool, err error, assertOutput func(output string)) {
		sb, invoked := strings.Builder{}, false
		cli.writeFunc = sbWriter(&sb)

		actualErr := cli.ReadRange(KakfaRangeReaderMock(func(tt string, f int64, ttt *int64, p int, rf func(m kafka.Message) bool) error {
			invoked = true

			if topic != tt || from != f || to != *ttt || partition != p {
				t.Fatalf("expected topic %s, from %d, to %d, partition %d to be passed. got %s, %d, %d, %d", topic, from, to, partition, tt, f, ttt, p)
			}

			return err
		}), topic, from, &to, partition, unattended, '\r')

		if !invoked {
			t.Fatal("expected kakfa-range-reader to be invoked")
		}

		if actualErr != err {
			t.Fatal("expected an error to be returned")
		}

		assertOutput(sb.String())
	}

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading from partition") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})

	act(false, nil, func(output string) {
		if !strings.Contains(output, "reading from partition") {
			t.Fatalf("expected output advise partition read. got  %v", output)
		}
	})

	act(true, fmt.Errorf("test-error"), func(output string) {
		if output != "" {
			t.Fatalf("expected no stdout output in unattended mode. got  %v", output)
		}
	})
}

func sbWriter(sb *strings.Builder) func(format string, a ...interface{}) (n int, err error) {
	return func(format string, a ...interface{}) (n int, err error) {
		return sb.Write([]byte(fmt.Sprintf(format, a...)))
	}
}
