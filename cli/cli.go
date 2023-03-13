package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"klient/kafka"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type (
	CLI struct {
		writeFunc  func(format string, a ...interface{}) (n int, err error)
		reader     io.Reader
		dateFormat string
	}
	KafkaDescriber interface {
		Bootstrappers() []string
		Describe() (map[string]*kafka.Broker, error)
	}
	KafkaRangeReader interface {
		ReadRange(topic string, from int64, to *int64, partition int, readFunc func(m kafka.Message) bool) error
	}
	KafkaTimeReader interface {
		ReadTime(topic string, from time.Time, to *time.Time, partition int, readFunc func(m kafka.Message) bool) error
	}
	KafkaExclusiveReader interface {
		ReadExclusive(topic string, readFunc func(m kafka.Message) bool) error
	}
	KafkaGroupReader interface {
		ReadGroup(topic, group string, readFunc func(m kafka.Message) bool) error
	}
	KafkaWriter interface {
		Writer(topic string, keyed bool) (writeFunc func(key, value []byte) error, closeFunc func() error, err error)
	}
	KafkaTopicCreator interface {
		CreateTopic(topic string, partitions, replicas int) error
	}
	KafkaTopicDeleter interface {
		DeleteTopic(topic string) error
	}
)

func New(dateFormat string) *CLI {
	return &CLI{reader: os.Stdin, dateFormat: dateFormat, writeFunc: fmt.Printf}
}

func (c *CLI) Version(version string) {
	c.writeFunc("klient version %v\n", version)
}

func (c *CLI) CreateTopic(k KafkaTopicCreator, topic string, partitions, replicas int) error {
	if err := k.CreateTopic(topic, partitions, replicas); err != nil {
		c.writeFunc("unable to create topic '%v'. see log for details\n", topic)
		return err
	}

	c.writeFunc("created topic '%v'\n", topic)
	return nil
}

func (c *CLI) DeleteTopic(k KafkaTopicDeleter, topic string) error {
	if err := k.DeleteTopic(topic); err != nil {
		c.writeFunc("unable to delete topic '%v'. see log for details\n", topic)
		return err
	}

	c.writeFunc("deleted topic %v\n", topic)
	return nil
}

func (c *CLI) Describe(k KafkaDescriber, unattended bool) error {
	brokers, err := k.Describe()

	if err != nil {
		c.writeFunc("unable to describe cluster using bootstrappers: '%v'. ensure the cluster is online and the bootstrappers are accesible", strings.Join(k.Bootstrappers(), ","))
		return err
	}

	if unattended {
		c.writeFunc("broker topic partition\n")

		for host, broker := range brokers {
			for name, topic := range broker.Topics {
				for id := range topic.Partitions {
					c.writeFunc("%v:%v %v %v\n", host, broker.Port, name, id)
				}
			}
		}
		return nil
	}

	c.writeFunc("describing cluster using bootstrappers: '%v'\n\n", strings.Join(k.Bootstrappers(), ","))

	if len(brokers) == 0 {
		c.writeFunc("> no brokers found. (brokers not acting as leader for at least one partition are not listed)\n\n")
	}

	for host, broker := range brokers {
		c.writeFunc("broker: %+v:%v\n", host, broker.Port)

		for name, topic := range broker.Topics {
			c.writeFunc("> topic: %+v\n", name)

			for id := range topic.Partitions {
				c.writeFunc("  > partition: %+v\n", id)
			}
		}
	}

	return nil
}

func (c *CLI) ReadRange(k KafkaRangeReader, topic string, from int64, to *int64, partition int, unattended bool, delimiter byte) error {
	toStr := "end"

	if to != nil {
		toStr = strconv.FormatInt(*to, 10)
	}

	msg := fmt.Sprintf("reading from partition %v of topic '%v' with from offset of %v to %v...\n\n", partition, topic, from, toStr)

	log.Print(msg)

	if !unattended {
		c.writeFunc(msg)
	}

	return k.ReadRange(topic, from, to, partition, func(m kafka.Message) bool {
		c.read(m, unattended, delimiter)
		return true
	})
}

func (c *CLI) ReadTime(k KafkaTimeReader, topic string, from time.Time, to *time.Time, partition int, unattended bool, delimiter byte) error {
	toStr := "end"

	if to != nil {
		toStr = (*to).Format(c.dateFormat)
	}

	msg := fmt.Sprintf("reading from partition %v of topic '%v' for time range '%v' to '%v'...\n\n", partition, topic, from.Format(c.dateFormat), toStr)

	log.Print(msg)

	if !unattended {
		c.writeFunc(msg)
	}

	return k.ReadTime(topic, from, to, partition, func(m kafka.Message) bool {
		c.read(m, unattended, delimiter)
		return true
	})
}

func (c *CLI) ReadGroup(k KafkaGroupReader, topic, group string, unattended bool, delimiter byte) error {
	msg := fmt.Sprintf("reading from topic '%v' as part of consumer group '%v'...\n\n", topic, group)

	log.Print(msg)

	if !unattended {
		c.writeFunc(msg)
	}

	return k.ReadGroup(topic, group, func(m kafka.Message) bool {
		c.read(m, unattended, delimiter)
		return true
	})
}

func (c *CLI) ReadExclusive(k KafkaExclusiveReader, topic string, unattended bool, delimiter byte) error {
	msg := fmt.Sprintf("reading exclusively from topic '%v'...\n\n", topic)

	log.Print(msg)

	if !unattended {
		c.writeFunc(msg)
	}

	return k.ReadExclusive(topic, func(m kafka.Message) bool {
		c.read(m, unattended, delimiter)
		return true
	})
}

func (c *CLI) read(msg kafka.Message, unattended bool, delimiter byte) {
	key := string(msg.Key)

	if unattended {
		c.writeFunc("%v%v", string(msg.Value), string(delimiter))
		return
	}

	if key == "" {
		key = "[unset]"
	}

	c.writeFunc("[%v/%v @ %v]> %v : %v\n", msg.Partition, msg.Offset, msg.TimeStamp.UTC().Format(c.dateFormat), key, string(msg.Value))
}

func (c *CLI) Write(k KafkaWriter, topic string, keyed, unattended bool, delimiter byte) error {
	writeFunc, closeFunc, err := k.Writer(topic, keyed)

	if err != nil {
		return err
	}

	stdin, key, value := bufio.NewScanner(c.reader), "", ""

	if !unattended {
		c.writeFunc("enter data to publish to topic '%v'. enter X to exit:\n> ", topic)
	} else if delimiter != '\n' {
		stdin.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) { // derived from stdlib's ScanLines SplitFunc
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := bytes.IndexByte(data, delimiter); i >= 0 {
				return i + 1, data[0:i], nil
			}
			if atEOF {
				return len(data), data, nil
			}
			return 0, nil, nil
		})
	}

	for stdin.Scan() {
		value = stdin.Text()

		if !unattended && strings.ToLower(value) == "x" {
			break
		}

		if keyed {
			if !unattended {
				if key == "" {
					key = "default"
					c.writeFunc("enter key (empty uses 'default')> ")
				} else {
					c.writeFunc("enter key (empty uses previous) > ")
				}
			}

			if !stdin.Scan() {
				break
			}

			if input := stdin.Text(); input != "" {
				key = input
			}
		}

		if err := writeFunc([]byte(key), []byte(value)); err != nil {
			if !unattended {
				c.writeFunc("error writing to topic. see log for details")
			}

			log.Printf("error writing [%v > %v]: %v\n", value, topic, err)
			break
		}

		if !unattended {
			c.writeFunc("wrote [%v > %v]\n> ", value, topic)
		}
	}

	if err := closeFunc(); err != nil {
		log.Printf("error closing writer: %v", err)
	}

	return nil
}
