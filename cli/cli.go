package cli

import (
	"bufio"
	"fmt"
	"io"
	"iter"
	"log"
	"slices"
	"strings"

	"github.com/comradequinn/klient/kafio"
)

type (
	WriterFunc        func(format string, v ...any)
	KafkaReaderWriter interface {
		DeleteTopic(name string) error
		NewTopic(name string, partitions int, replicas int) error
		ReadTopic(name string, all bool) (string, iter.Seq[kafio.ReadResult])
		TopicExists(name string) (bool, error)
		Topics() (map[string]int, error)
		WriteTopic(name string, key string, value string, headers map[string]string) error
	}
)

var (
	kafka      KafkaReaderWriter
	writerFunc WriterFunc
)

// generate me a unit test for this 
func Init(k KafkaReaderWriter, wf WriterFunc) {
	kafka = k
	writerFunc = wf
}

func Info() error {
	topics, err := kafka.Topics()

	if err != nil {
		return err
	}

	sortedTopics := make([]string, 0, len(topics))

	for topic := range topics {
		sortedTopics = append(sortedTopics, topic)
	}

	slices.Sort(sortedTopics)

	writerFunc("%-50v %v\n", "NAME", "PARTITIONS")

	for _, topic := range sortedTopics {
		partitions := topics[topic]
		writerFunc("%-50v %v\n", topic, partitions)
	}

	return nil
}

func CreateTopic(name string, partitions, replicas int) error {
	if err := kafka.NewTopic(name, partitions, replicas); err != nil {
		return err
	}

	writerFunc("topic '%v' created successfully", name)

	return nil
}

func DeleteTopic(name string) error {
	if err := kafka.DeleteTopic(name); err != nil {
		return err
	}

	writerFunc("topic '%v' deleted successfully", name)

	return nil
}

func WriteTopic(name, key, value string, headerCSV string) error {
	headers := map[string]string{}

	if headerCSV != "" {
		func() {
			defer func() {
				if err := recover(); err != nil {
					log.Fatalf("error: unable to parse headers. %v", err)
				}
			}()

			for _, kv := range strings.Split(headerCSV, ",") {
				header := strings.Split(kv, "=")
				headers[strings.TrimSpace(header[0])] = strings.TrimSpace(header[1])
			}
		}()
	}

	if err := kafka.WriteTopic(name, key, value, headers); err != nil {
		return err
	}

	writerFunc("%v bytes of data written to topic '%v'", len(value), name)

	return nil
}

func ReadTopic(name string, readAll, stepThrough, includeHeaders bool, inputReader io.Reader) error {
	consumerGroup, topicIteratorFunc := kafka.ReadTopic(name, readAll)
	msg, input := fmt.Sprintf("reading from topic '%v' as consumer '%v'", name, consumerGroup), bufio.NewScanner(inputReader)

	if stepThrough {
		writerFunc("%v. press enter to read the first message and enter again after each message to move to the next...\n", msg)
		input.Scan()
	} else {
		writerFunc("%v...\n\n", msg)
	}

	for readResult := range topicIteratorFunc {
		if readResult.Err != nil {
			return readResult.Err
		}

		msg := readResult.Message

		writerFunc("<< [%v partition:%v offset:%v]\n", msg.TimeStamp, msg.Partition, msg.Offset)

		if includeHeaders {
			for k, v := range msg.Headers {
				writerFunc("-%v: %v\n", k, v)

			}
			writerFunc("\n")
		}

		writerFunc("%s\n\n", msg.Value)

		if stepThrough {
			input.Scan()
		}
	}

	return nil
}
