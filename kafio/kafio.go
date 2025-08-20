package kafio

import (
	"context"
	"fmt"
	"iter"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type (
	Connection struct {
		driver        *kafka.Conn
		bootstrappers []string
	}
	Message struct {
		Headers   map[string]string
		Partition int
		Offset    int64
		Key       []byte
		Value     []byte
		TimeStamp time.Time
	}
	ReadResult struct {
		Message Message
		Err     error
	}
	Topic struct {
		Partitions int
		Replicas   int
		Leader     Leader
	}
	Leader struct {
		ID   int
		Host string
		Port int
	}
)

var (
	hostname = func() string {
		h, _ := os.Hostname()
		return h
	}()
)

// Connect returns a Connection used to manage interactions with a kafka cluster
func Connect(bootstrappers []string) (*Connection, error) {
	connection := Connection{
		bootstrappers: bootstrappers,
	}

	var (
		controllerBroker kafka.Broker
		err              error
	)

	for _, bs := range bootstrappers {
		if connection.driver, err = kafka.Dial("tcp", bs); err != nil {
			log.Printf("error connecting to bootstrapper [%v]: %v", bs, err)
			continue
		}
		break
	}

	if err != nil {
		return nil, fmt.Errorf("unable to connect to any of the specified bootstrappers [%+v]: %v", bootstrappers, err)
	}

	if controllerBroker, err = connection.driver.Controller(); err != nil {
		return nil, fmt.Errorf("unable to ascertain the cluster controller: %v", err)
	}

	if connection.driver, err = kafka.Dial("tcp", net.JoinHostPort(controllerBroker.Host, strconv.Itoa(controllerBroker.Port))); err != nil {
		return nil, fmt.Errorf("unable to connect to the cluster controller: %v", err)
	}

	return &connection, nil
}

// Topics returns a map keyed on the cluster's topics with
// the value being the partition count
func (conn *Connection) Topics() (map[string]Topic, error) {
	partitions, err := conn.driver.ReadPartitions()

	if err != nil {
		return nil, fmt.Errorf("error reading cluster info. %v", err)
	}

	topics := map[string]Topic{}

	for _, p := range partitions {
		topic := topics[p.Topic]
		topic.Partitions++
		topic.Replicas = len(p.Replicas)
		topic.Leader.ID = p.Leader.ID
		topic.Leader.Host = p.Leader.Host
		topic.Leader.Port = p.Leader.Port
		topics[p.Topic] = topic
	}

	return topics, nil
}

// TopicExists returns whether the named topic exists on the current cluster
func (conn *Connection) TopicExists(name string) (bool, error) {
	var (
		topics map[string]Topic
		err    error
	)

	if topics, err = conn.Topics(); err != nil {
		return false, fmt.Errorf("error listing current topics. %v", err)
	}

	_, exists := topics[name]

	return exists, nil

}

// NewTopic creates a new topic with specified properties on the current cluster
func (conn *Connection) NewTopic(name string, partitions, replicas int) error {
	if exists, err := conn.TopicExists(name); exists || err != nil {
		if exists {
			return fmt.Errorf("cannot create topic. topic already exists")
		}
		return err
	}

	err := conn.driver.CreateTopics(kafka.TopicConfig{
		Topic:             name,
		NumPartitions:     partitions,
		ReplicationFactor: replicas})

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "policy violation") {
			return fmt.Errorf("topic '%v' could not be created due to a policy violation. this may be due to an invalid replication factor. some container image instances used for development purposes require a value of 1 whereas production instances often require a value of at least 3: %v", name, err)
		}

		return fmt.Errorf("topic '%v' already exists or could not be created: %v", name, err)
	}

	return nil
}

// DeleteTopic removes the specified topic from the current cluster
func (conn *Connection) DeleteTopic(name string) error {
	if exists, err := conn.TopicExists(name); !exists || err != nil {
		if err != nil {
			return err
		}
		return fmt.Errorf("cannot delete topic. topic does not exist")
	}

	if err := conn.driver.DeleteTopics(name); err != nil {
		return fmt.Errorf("unable to delete topic. %v", name)
	}

	return nil
}

// ReadTopic reads all messages from all partitions for the given a topic.
func (conn *Connection) ReadTopic(name string, all bool) (string, iter.Seq[ReadResult]) {
	consumerGroup := fmt.Sprintf("klient-%v-%v-%v", hostname, os.Getenv("USER"), strconv.FormatInt(rand.Int63(), 10))

	iteratorFunc := func(yield func(ReadResult) bool) {
		startOffset := kafka.LastOffset

		if all {
			startOffset = kafka.FirstOffset
		}

		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     conn.bootstrappers,
			Topic:       name,
			GroupID:     consumerGroup,
			StartOffset: startOffset,
		})

		defer r.Close()

		for {
			m, err := r.ReadMessage(context.Background())

			if err != nil {
				yield(ReadResult{
					Err: err,
				})
				return
			}

			headers := make(map[string]string, len(m.Headers))

			for _, h := range m.Headers {
				headers[h.Key] = string(h.Value)
			}

			if !yield(ReadResult{
				Message: Message{
					Headers:   headers,
					Partition: m.Partition,
					Offset:    m.Offset,
					Key:       m.Key,
					Value:     m.Value,
					TimeStamp: m.Time,
				},
			}) {
				return
			}
		}
	}

	return consumerGroup, iteratorFunc
}

// WriteTopic writes the specified data to the specified topic with the specified key
func (conn *Connection) WriteTopic(name, key, value string, headers map[string]string) error {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(conn.bootstrappers...),
		Topic:                  name,
		AllowAutoTopicCreation: false,
		Balancer:               &kafka.ReferenceHash{},
	}

	m := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
	}

	for k, v := range headers {
		m.Headers = append(m.Headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	return w.WriteMessages(context.Background(), m)
}
