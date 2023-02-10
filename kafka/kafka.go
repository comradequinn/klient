package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	sasl "github.com/segmentio/kafka-go/sasl"
	plain "github.com/segmentio/kafka-go/sasl/plain"
	scram "github.com/segmentio/kafka-go/sasl/scram"
)

type (
	Kafka struct {
		bootstrappers []string
	}
	Topic struct {
		Name       string
		Partitions map[int]*Partition
	}
	Broker struct {
		Host   string
		Port   int
		Topics map[string]*Topic
	}
	Partition struct {
		ID int
	}
	Message struct {
		Partition int
		Offset    int64
		Key       []byte
		Value     []byte
		TimeStamp time.Time
	}
	ConnectConfig struct {
		APIKey        string
		APISecret     string
		TLS           bool
		SkipTLSVerify bool
		SCRAM         bool
		Timeout       time.Duration
	}
)

func New(c ConnectConfig, bootstrappers ...string) (*Kafka, error) {
	var transport *kafka.Transport

	kafka.DefaultDialer.Timeout = c.Timeout

	if c.TLS {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.SkipTLSVerify,
		}
		kafka.DefaultDialer.TLS = tlsCfg
		transport = &kafka.Transport{
			TLS: &tls.Config{
				InsecureSkipVerify: c.SkipTLSVerify,
			},
		}
	}

	if c.APIKey != "" {
		var (
			m   sasl.Mechanism
			err error
		)

		if c.SCRAM {
			if m, err = scram.Mechanism(scram.SHA512, c.APIKey, c.APISecret); err != nil {
				return nil, fmt.Errorf("error creating scram sasl mechanism: %v", err)
			}
		} else {
			m = plain.Mechanism{Username: c.APIKey, Password: c.APISecret}
		}

		kafka.DefaultDialer.SASLMechanism = m

		if transport == nil {
			transport = &kafka.Transport{}
		}

		transport.SASL = m
	}

	if transport != nil {
		kafka.DefaultTransport = transport
	}

	return &Kafka{bootstrappers: bootstrappers}, nil
}

func (k *Kafka) Bootstrappers() []string {
	return k.bootstrappers
}

func (k *Kafka) Close() error {
	controller, err := k.controllerConnection()

	if err != nil {
		return nil
	}

	return controller.Close()
}

func (k *Kafka) CreateTopic(topic string, partitions, replicas int) error {
	var err error

	controller, err := k.controllerConnection()

	if err != nil {
		return err
	}

	err = controller.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: replicas})

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "policy violation") {
			return fmt.Errorf("topic '%v' could not be created due to a policy violation. this may be due to an invalid replication factor. some container image instances used for development purposes require a value of 1 whereas production instances often require a value of at least 3: %v", topic, err)
		}
		return fmt.Errorf("topic '%v' already exists or could not be created: %v", topic, err)
	}

	return nil
}

func (k *Kafka) DeleteTopic(topic string) error {
	controller, err := k.controllerConnection()

	if err != nil {
		return err
	}

	log.Printf("deleting topic '%v'\n", topic)

	if err := controller.DeleteTopics(topic); err != nil {
		return fmt.Errorf("topic '%v' does not exist or could not be deleted: %v", topic, err)
	}

	return nil
}

func (k *Kafka) Writer(topic string, keyed bool) (writeFunc func(key, value []byte) error, closeFunc func() error, err error) {
	balancer := kafka.Balancer(&kafka.RoundRobin{})

	if keyed {
		balancer = &kafka.ReferenceHash{}
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(k.bootstrappers...),
		Topic:                  topic,
		AllowAutoTopicCreation: false,
		Balancer:               balancer,
	}

	return func(key, value []byte) error {
		return w.WriteMessages(context.Background(),
			kafka.Message{
				Key:   key,
				Value: value,
			},
		)
	}, func() error { return w.Close() }, nil
}

func (k *Kafka) ReadRange(topic string, from int64, to *int64, partition int, readFunc func(m Message) bool) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   k.bootstrappers,
		Topic:     topic,
		Partition: partition,
	})

	if from > 0 {
		if err := r.SetOffset(from); err != nil {
			return err
		}
	}

	defer r.Close()

	continueFunc := func(m kafka.Message) bool { return true }

	if to != nil {
		continueFunc = func(m kafka.Message) bool {
			return m.Offset <= *to
		}
	}

	return k.read(r, readFunc, continueFunc)
}

func (k *Kafka) ReadTime(topic string, from time.Time, to *time.Time, partition int, readFunc func(m Message) bool) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   k.bootstrappers,
		Topic:     topic,
		Partition: partition,
	})

	if err := r.SetOffsetAt(context.Background(), from); err != nil {
		return err
	}

	defer r.Close()

	continueFunc := func(m kafka.Message) bool { return true }

	if to != nil {
		continueFunc = func(m kafka.Message) bool {
			return m.Time.Before(*to) || m.Time.Equal(*to)
		}
	}

	return k.read(r, readFunc, continueFunc)
}

func (k *Kafka) ReadGroup(topic, group string, readFunc func(m Message) bool) error {
	if group == "" {
		return fmt.Errorf("group id is required")
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.bootstrappers,
		Topic:   topic,
		GroupID: group,
	})

	defer r.Close()

	return k.read(r, readFunc, nil)
}

func (k *Kafka) ReadExclusive(topic string, readFunc func(m Message) bool) error {
	// create a unique groupid to configure this reader as an exclusive consumer (ensuring all partitions are read and this process reads all messages)
	group := strconv.FormatInt(newGroup(), 10)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.bootstrappers,
		Topic:   topic,
		GroupID: group,
	})

	defer r.Close()

	return k.read(r, readFunc, nil)
}

func (k *Kafka) Describe() (map[string]*Broker, error) {
	controller, err := k.controllerConnection()

	if err != nil {
		return nil, err
	}

	partitions, err := controller.ReadPartitions()

	if err != nil {
		return nil, err
	}

	brokers := map[string]*Broker{}

	for _, p := range partitions {
		broker, ok := brokers[p.Leader.Host]

		if !ok {
			broker = &Broker{Host: p.Leader.Host, Port: p.Leader.Port, Topics: map[string]*Topic{}}
			brokers[p.Leader.Host] = broker
		}

		topic, ok := broker.Topics[p.Topic]

		if !ok {
			topic = &Topic{Name: p.Topic, Partitions: map[int]*Partition{}}
			broker.Topics[p.Topic] = topic
		}

		_, ok = topic.Partitions[p.ID]

		if !ok {
			partition := &Partition{ID: p.ID}
			topic.Partitions[p.ID] = partition
		}
	}

	return brokers, nil
}

func (k *Kafka) read(r *kafka.Reader, readFunc func(m Message) bool, continueFunc func(m kafka.Message) bool) error {
	for {
		m, err := r.ReadMessage(context.Background())

		if err != nil {
			return err
		}

		if continueFunc != nil && !continueFunc(m) {
			return nil
		}

		if !readFunc(Message{
			Partition: m.Partition,
			Offset:    m.Offset,
			Key:       m.Key,
			Value:     m.Value,
			TimeStamp: m.Time,
		}) {
			return nil
		}
	}
}

func (k *Kafka) controllerConnection() (*kafka.Conn, error) {
	var (
		controllerConn    *kafka.Conn
		bootstrapperConn  *kafka.Conn
		controllerAddress kafka.Broker
		err               error
	)

	for _, bs := range k.bootstrappers {
		if bootstrapperConn, err = kafka.Dial("tcp", bs); err != nil {
			log.Printf("error connecting to bootstrapper [%v]: %v", bs, err)
			continue
		}
		break
	}

	if err != nil {
		return nil, fmt.Errorf("unable to connect to any of the specified bootstrappers [%+v]: %v", k.bootstrappers, err)
	}

	if controllerAddress, err = bootstrapperConn.Controller(); err != nil {
		return nil, fmt.Errorf("unable to ascertain the cluster controller: %v", err)
	}

	if controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controllerAddress.Host, strconv.Itoa(controllerAddress.Port))); err != nil {
		return nil, fmt.Errorf("unable to connect to the cluster controller: %v", err)
	}

	return controllerConn, nil
}

var newGroup = func() func() int64 {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func() int64 { return rnd.Int63() }
}()
