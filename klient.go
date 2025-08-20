package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/comradequinn/klient/cli"
	"github.com/comradequinn/klient/kafio"
)

var (
	commit = "dev-build"
	tag    = "v0.0.0-dev.0"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	var (
		versionArg                 = flag.Bool("version", false, "print klient version")
		bootstrappersArg           = flag.String("b", "localhost:9092", "the comma separated list of kafka bootstrappers to use to connect")
		writeTopicArg              = flag.String("w", "", "the topic to write to")
		writeTopicHeadersArg       = flag.String("h", "", "the comma separated set of 'key=value' formatted headers to use when writing to a topic. eg \"k1=v1,k2=v2\"")
		writeTopicKeyArg           = flag.String("k", "", "the key to use when writing to a topic")
		writeTopicValueArg         = flag.String("v", "", "the value to use when writing to a topic")
		readTopicArg               = flag.String("r", "", "the topic to read")
		readTopicExcludeHeadersArg = flag.Bool("x", false, "whether to e(-x)clude message headers from the output when reading from a topic")
		readTopicStepArg           = flag.Bool("s", false, "whether to step through messages one at time by pressing enter")
		readTopicAllArg            = flag.Bool("a", false, "read all messages when reading from a topic as opposed those added since the read was started")
		createTopicArg             = flag.String("c", "", "the topic to create")
		createTopicPartitionsArg   = flag.Int("p", 3, "used with -c to specify the number of partitions to assign a new topic")
		createTopicReplicasArg     = flag.Int("n", 3, "used with -c to specify the number of replicas to assign a new topic")
		deleteTopicArg             = flag.String("d", "", "the topic to delete")
		infoArg                    = flag.Bool("i", false, "print cluster info")
	)

	flag.Parse()

	if *versionArg {
		log.Printf("klient version %v (commit %v)\n", tag, commit)
		os.Exit(0)
	}

	bootstrappers := strings.Split(*bootstrappersArg, ",")

	if len(bootstrappers) == 0 {
		log.Fatalf("error: at least one bootstrapper must be specified")
	}

	conn, err := kafio.Connect(bootstrappers)

	if err != nil {
		log.Fatalf("error: unable to connect to cluster controller. %v", err)
	}

	cli.Init(conn, log.Printf)

	require := func(param, arg string) {
		if arg == "" {
			log.Fatalf("error: missing required argument for '%v'", param)
		}
	}

	switch {
	case *infoArg:
		if err := cli.Info(); err != nil {
			log.Fatalf("error: unable to read cluster info. %v", err)
		}
	case *createTopicArg != "":
		require("topic", *createTopicArg)

		if err := cli.CreateTopic(*createTopicArg, *createTopicPartitionsArg, *createTopicReplicasArg); err != nil {
			log.Fatalf("error: unable to create topic. %v", err)
		}
	case *deleteTopicArg != "":
		require("topic", *deleteTopicArg)

		if err := cli.DeleteTopic(*deleteTopicArg); err != nil {
			log.Fatalf("error: unable to delete topic. %v", err)
		}
	case *writeTopicArg != "":
		require("topic", *writeTopicArg)
		require("key", *writeTopicKeyArg)
		require("value", *writeTopicValueArg)

		if err := cli.WriteTopic(*writeTopicArg, *writeTopicKeyArg, *writeTopicValueArg, *writeTopicHeadersArg); err != nil {
			log.Fatalf("error: unable to write to topic. %v", err)
		}
	case *readTopicArg != "":
		require("topic", *readTopicArg)

		if err := cli.ReadTopic(*readTopicArg, *readTopicAllArg, *readTopicStepArg, !*readTopicExcludeHeadersArg, os.Stdin); err != nil {
			log.Fatalf("error: unable to read from topic. %v", err)
		}

	default:
		log.Println("error: specify a valid command")
		flag.PrintDefaults()
	}
}
