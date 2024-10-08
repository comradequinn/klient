# klient

The `klient` utility exposes simplifed kafka cluster administration operations for use by developers. Specifically:

* Describe Cluster
* Create Topic
* Delete Topic
* Consume Topic
* Produce to Topic

## Installation

If `go` is available on the machine, `klient` can be installed quickly by running the below.

```bash
go install github.com/comradequinn/klient
```

Alternatively, there is a `amd64/linux` binary in this repo's `/bin` directory. Copy this repo to a location in your `$PATH` variable and grant it execute permissions.

### Local Kafka Broker

The `Makefile` contains a convenience target to spin up a local `kafka broker` which will be available at `localhost:9092`, as shown below.

```bash
make local-kafka
```

The broker can be stopped by running the below

```bash
make stop-kafka
```

## Usage

For all tasks, `klient` requires a comma seperated list of bootstrap servers for the target cluster, containing at least one host. 

These are specified with the `-b` flag as shown below.

```bash
klient -b "localhost:9092"
```

For clusters that are used regularly, it can be useful to define these as environment variables, or alias the klient command with a `-b` argument preset. For example.

```bash
# file: ~/.bashrc
alias klient-dev="klient -b kafka-broker-1:9092,kafka-broker-2:9092,kafka-broker-3:9092"
alias klient-staging="klient -b kafka-broker-4:9092,kafka-broker-5:9092,kafka-broker-6:9092"
```

This can then be used without specifying the bootstrappers, as shown below.

```bash
klient-dev -i # print cluster info for dev
# ...output ommitted
klient-staging -i # print cluster info for staging
# ...output ommitted
```

## Cluster Info

To print the topics on a cluster and their partition counts, use the `-i` flag, as shown below.

```bash
klient -i -b "localhost:9092"
``` 

The resulting output would be of the format below:

```text
NAME                                               PARTITIONS
example-topic-1                                    3
example-topic-2                                    2
example-topic-3                                    4
example-topic-4                                    8
```

##  Topic Creation

To create a topic using the default replication factor and partition count, use the `-c` flag, as shown the below:

```bash
klient -c "my-topic" -b "localhost:9092"
``` 

The resulting output would be of the format below:

```text
topic 'my-topic' created successfully
```

Alternatively, specify these attributes as required:

```bash
# specify a partition count of 3 and a replication factor of 1
klient -c "my-topic" -p 3 -r 1 -b "localhost:9092"
``` 

##  Topic Deletion

To delete a topic, run the below:

```bash
klient -d "my-topic" -b "localhost:9092"
``` 

The resulting output would be of the format below:

```text
topic 'my-topic' deleted successfully
```

## Consuming a Topic

To perform a basic read from a topic, use the `-r` flag, as shown below.

```bash
klient -r "my-topic" -b "localhost:9092"
```

The above prints the location, value and headers of each message delivered to the topic from the point the command was run as they arrive on the topic. The output would be of the format shown below:

```text
<< [2024-10-07 16:58:16.554 +0000 UTC partition:2 offset:1]
-example-header2: example-value2
-example-header1: example-value1

example-data
```

Alternatively, the all the messages on the topic can printed by specifying the `-a` flag and the header data can be excluded from the output with the `-x` flag. The `-s` can also be used to step through messages one-by-one, as opposed to printing them immediately. An example is shown below.

```bash
klient -r "my-topic" -s -a -x -b "localhost:9092" # step through all messages on 'my-topic' without printing headers
```

The output for each message would then be of the format below:

```text
<< [2024-10-07 16:58:16.554 +0000 UTC partition:2 offset:1]
example-data
```

## Producing to a Topic

To perform a basic write to a topic, use the `-w`, `-k` and `-v` flags. The latter two flags being used to specify the required key and value, as shown below.

```bash
klient -w "my-topic" -k "mykey" -v "some-data" -b "localhost:9092"
```

Optionally, headers may also be provided as a set of `key=value` pairs separated by commas. An example is shown below.

```bash
klient -w "my-topic" -k "mykey" -v "some-data" -h "example-header1=value1,example-header2=value2" -b "localhost:9092"
```

The resulting output would be of the format below:

```text
9 bytes of data written to topic 'my-topic'
```