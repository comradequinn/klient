package main

import (
	"flag"
	"fmt"
	"io"
	"klient/cli"
	"klient/kafka"
	"log"
	"os"
	"strings"
	"time"
)

var versionNumber = "unversioned-build"

func main() {
	var ( // klient flags
		version     = flag.Bool("version", false, "print the version of klient")
		logFilePath = flag.String("log", "./klient.log", "the log file name")
	)

	var ( // cluster flags
		bootstrappers = flag.String("bootstrappers", "localhost:9092", "the comma separated list of bootstrap servers")
		describe      = flag.Bool("describe", false, "outputs information about the cluster")
	)

	var ( // topic editing flags
		delete     = flag.String("delete", "", "deletes the topic with the specified name")
		create     = flag.String("create", "", "creates a topic with the specified name and the specfied attributes (or their default values)")
		partitions = flag.Int("partitions", 6, "used with -create to specify the number of partitions to split the topic over")
		replicas   = flag.Int("replicas", 3, "used with -create to specify the number of partition replicas to configure for the topic")
	)

	var ( // topic io flags
		unattended = flag.Bool("unattended", false, "run in an unattended-compatible manner when reading or writing to or from topics. use this flag when stdin/out is non-interactive (eg piped/redirected to/from another command)")
		delimiter  = flag.String("delimiter", "\n", "the message and key delimiter to use instead of (the default) 'new line' on data read/written to/from stdio when in -read or -write mode with -unattended specified: must be a single byte value")
	)

	var ( // write flags
		write = flag.String("write", "", "writes to specifed topic")
		keyed = flag.Bool("keyed", false, "whether to specify a key when in -write mode. in -unattended mode this should be included in the input using the specifed -delimiter (eg 'key[delim]value[delim]key[delim]value'... etc). in attended mode it will be prompted for")
	)

	var ( // read mode flags
		readRange     = flag.String("range-read", "", "reads from the specified topic using the specified -offset-from, offset-to and -partition values (or their defaults)")
		readExclusive = flag.String("exclusive-read", "", "reads from the specified topic as an exclusive consumer (from offset 0 across all partitions)")
		readGroup     = flag.String("group-read", "", "reads from the specified topic name as part of the specified -group")
		readTime      = flag.String("time-read", "", "reads from the specified topic using the specified -time-from, -time-to and -partition values (or their defaults)")
		offsetFrom    = flag.Int64("offset-from", 0, "the offset to start reading from in -range-read mode. if no value is provided, the partition is read from the beginning")
		offsetTo      = flag.Int64("offset-to", -1, "the offset to read to in -range-read mode. if no value is provided, reading continues indefinitely")
		timeFrom      = flag.String("time-from", "", "the time to start reading from in -time-read mode. specified in the format 'DD-MM-YYYY HH:MM:SS'. if no value is provided, reading starts from the previous minute")
		timeTo        = flag.String("time-to", "", "the time to start from in -time-read mode. specified in the format 'DD-MM-YYYY HH:MM:SS'. if no value is provided, reading continues indefinitely")
		partition     = flag.Int("partition", 0, "the partition to read from in -range-read or -time-read mode")
		group         = flag.String("group", "", "the consumer group to join in -group-read mode")
	)

	connectConfig := kafka.ConnectConfig{} // connect flags
	{
		flag.StringVar(&connectConfig.APIKey, "authkey", "", "the api key or username to authenticate with, if required")
		flag.StringVar(&connectConfig.APISecret, "authsecret", "", "the api secret or password to authenticate with, if required")
		flag.BoolVar(&connectConfig.TLS, "tls", false, "whether to connect with tls")
		flag.BoolVar(&connectConfig.SkipTLSVerify, "tlsnoverify", false, "whether to skip tls certificate verification (insecure)")
		flag.BoolVar(&connectConfig.SCRAM, "scram", false, "whether to authenticate with scram")
		timeout := flag.Int("timeout", 5000, "the connection timeout in ms")

		connectConfig.Timeout = time.Millisecond * time.Duration(*timeout)
	}

	flag.Parse()

	var (
		logFile io.Writer
		err     error
	)

	const dateFormat = "02-01-2006 15:04:05"

	if logFile, err = os.Create(*logFilePath); err != nil {
		log.Fatalf("unable to create log file at [%v]: %v", *logFilePath, err)
	}

	log.SetOutput(logFile)

	log.Printf("starting using bootstrappers: [%v]", *bootstrappers)

	cli := cli.New(dateFormat)

	if *version {
		cli.Version(versionNumber)
		return
	}

	countTrue := func(flags ...bool) int {
		count := 0
		for _, f := range flags {
			if f {
				count++
			}
		}

		return count
	}

	if countTrue(*describe, *create != "", *delete != "", *write != "", *readRange != "", *readExclusive != "", *readGroup != "", *readTime != "") != 1 {
		fmt.Printf("invalid flag specification: specify mode (-describe, -create, -delete, -write, -range-read, -exclusive-read, -group-read, -time-read)\n")
		flag.Usage()
		return
	}

	if len(*delimiter) > 1 {
		fmt.Printf("invalid flag specification: delimiter must be a single ascii character\n")
		flag.Usage()
		return
	}

	var (
		k *kafka.Kafka
	)

	if k, err = kafka.New(connectConfig, strings.Split(*bootstrappers, ",")...); err != nil {
		log.Fatalf("error starting: %v", err)
		return
	}

	defer k.Close()

	switch {
	case *describe:
		if err = cli.Describe(k); err != nil {
			log.Fatalf("error describing cluster using bootstrappers of [%v]: %v\n", *bootstrappers, err)
		}
	case *create != "":
		if err = cli.CreateTopic(k, *create, *partitions, *replicas); err != nil {
			log.Fatalf("error creating topic [%v] with [%v] partitions and [%v] replicas: %v\n", *create, *partitions, *replicas, err)
		}
	case *delete != "":
		if err = cli.DeleteTopic(k, *delete); err != nil {
			log.Fatalf("error deleting topic [%v]: %v\n", *delete, err)
		}
	case *write != "":
		if err = cli.Write(k, *write, *keyed, *unattended, (*delimiter)[0]); err != nil {
			log.Fatalf("error writing to topic [%v]: %v\n", *write, err)
		}
	case *readRange != "":
		var to *int64

		if *offsetTo >= 0 {
			to = offsetTo
		}

		if err = cli.ReadRange(k, *readRange, *offsetFrom, to, *partition, *unattended, (*delimiter)[0]); err != nil {
			log.Fatalf("error reading from partition [%v] of topic [%v] at offset [%v]: %v\n", *partition, *readRange, *offsetFrom, err)
		}
	case *readTime != "":
		var (
			from = time.Now().Add(-time.Minute)
			to   *time.Time
		)

		if *timeFrom != "" {
			if from, err = time.Parse(dateFormat, *timeFrom); err != nil {
				log.Fatalf("unable to parse time-from value: %v", err)
			}
		}

		if *timeTo != "" {
			if val, err := time.Parse(dateFormat, *timeTo); err == nil {
				to = &val
			} else {
				log.Fatalf("unable to parse time-to value: %v", err)
			}
		}

		if err = cli.ReadTime(k, *readTime, from, to, *partition, *unattended, (*delimiter)[0]); err != nil {
			log.Fatalf("error reading from partition [%v] of topic [%v] for time range [%v - %v]: %v\n", *partition, *readTime, from, to, err)
		}
	case *readGroup != "":
		if *group == "" {
			fmt.Printf("invalid flag specification: specfify -group to use -group-read mode\n")
			flag.Usage()
			return
		}
		if err = cli.ReadGroup(k, *readGroup, *group, *unattended, (*delimiter)[0]); err != nil {
			log.Fatalf("error reading from topic [%v] in group [%v]: %v\n", *readGroup, *group, err)
		}
	case *readExclusive != "":
		if err = cli.ReadExclusive(k, *readExclusive, *unattended, (*delimiter)[0]); err != nil {
			log.Fatalf("error reading exclusively from topic [%v]: %v\n", *readExclusive, err)
		}
	}

	log.Println("terminated")
}
