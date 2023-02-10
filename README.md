# Klient
A command line Kafka client with the following features:

* Simple and intuitive CLI (command line interface)
* Describe clusters by outputting information about brokers, topics and partitions
* Create topics with default values or with specific partition counts and replication factors
* Delete topics  
* Read topics quickly and simply as an exclusive consumer (all messages in all partitions)  
* Read from topics with granular control over the partition and the offset or time ranges  
* Read from topics as part of a shared consumer group  
* Format data read from topics so it can be readily piped into other processes, such parsers (eg: pipe `json` values into `jq` for formatting and querying purposes)
* Write to topics interactively by following simple CLI prompts to specify the value and, an optional, key
* Write to topics based on data read from `stdin`, such as data piped from a pre-defined file of topic values; optionally containing their associated keys
* Authenticate anonymously or with TLS and/or SASL (using plain or SCRAM credentials) 
* Ships as a single binary for ease of deployment and installation; even on scratch based container images (does not require `libc`)

# Installation
Download the appropriate binary for your system from [releases](https://github.com/comradequinn/klient/releases), add execute permissions and then execute it. 

To make `klient` available globally via the `klient` command; copy, or symlink, the downloaded binary into `/usr/local/bin/` or any other suitable directory available on your `PATH` environment variable.

Alternatively, the scripts below will download and install `klient` for you; select the one appropriate for your system and execute it in a terminal:

```bash
# linux on amd 64: amd64
sudo rm -f /usr/local/bin/klient 2> /dev/null; sudo curl -L "https://github.com/comradequinn/klient/releases/download/v1.3.1/klient.linux.amd64" -o /usr/local/bin/klient && sudo chmod +x /usr/local/bin/klient
```

```bash
# macOS on apple silicon: arm64
sudo rm -f /usr/local/bin/klient 2> /dev/null; sudo curl -L "https://github.com/comradequinn/klient/releases/download/v1.3.1/klient.darwin.arm64" -o /usr/local/bin/klient && sudo chmod +x /usr/local/bin/klient
```

```bash
# macOS on intel silicon: amd64
sudo rm -f /usr/local/bin/klient 2> /dev/null; sudo curl -L "https://github.com/comradequinn/klient/releases/download/v1.3.1/klient.darwin.amd64" -o /usr/local/bin/klient && sudo chmod +x /usr/local/bin/klient
```

## From Source
Clone this repo and `cd` into the resulting directory. Run `make install`. The repo can then, optionally, be deleted.

## Examples 
The repo also contains a `docker-compose.yaml` for creating a local `kafka` instance. This can be started with `make local-kafka` (and stopped with `make stop-local-kafka`) Once running, the `Makefile` contains sample targets illustrating how to use `klient` by operating against that local instance.

# Usage

## Quick Start
To write to a topic run the below and then follow the prompts in the terminal:

```bash
# specify at least one bootstrap broker and the topic to write to
klient -bootstrappers "kafka-broker-1:9092" -write "my-topic"
``` 

To read from a topic as the exclusive consumer, run the below and then follow the prompts in the terminal:

```bash
# specify at least one bootstrap broker, the topic to write to exclusively (all partitions with offset of 0)
klient -bootstrappers "kafka-broker-1:9092" -exclusive-read "my-topic" 
``` 

## General
For all tasks, `klient` requires a comma seperated list of bootstrap servers for the target cluster, containing at least one host. It will default to a local test instance running on the convential port if this is not provided. 

An example of the above points is shown below. For brevity, the rest of the examples assume the default broker and do not redirect the diagnostic output.

```bash
klient -bootstrappers "kafka-broker-1:9092,kafka-broker-2:9092,kafka-broker-3:9092" -describe -log "./my-custom.log" 
```

### Connection Control
By default, `klient` connects to kafka anonymously; that is, it passes no authorisation credentials and communicates over an unsecured connection. For clusters that require authorisation and/or a secure connection, the following options can be provided:

```bash
    klient  -authkey "my-user" \ # user name or api key
            -authsecret "my-password" \ # password or api secret
            -timeout 5000 \ # the connection timeout in ms
            -tls \ # secure the connection with tls 
            -tlsnoverify # if using tls, skip certificate verification (insecure)
            -scram # authenticate with scram
```

To avoid specifying lengthy connection credentials repeatedly with each command, the `klient` command can be aliased for the session to include them by default. This is shown below:

```bash
# create an alias for klient that embeds the conection credentials
alias klient='klient -bootstrappers "kafka-broker-1:9092,kafka-broker-2:9092,kafka-broker-3:9092" -authkey "api_key_or_username" -authsecret "api_secret_or_password" -tls -tlsnoverify'

# now execute klient without specifying credentials directly
klient -describe 
```

If you connect to multiple clusters, aliasing each set of connection credentials separately can be helpful. For example:

```bash
# create an alias for klient that embeds the conection credentials for the first cluster
alias klient-prod='klient -bootstrappers "kafka-broker-1:9092,kafka-broker-2:9092,kafka-broker-3:9092" -authkey "api_key_or_username" -authsecret "api_secret_or_password" -tls -tlsnoverify'
# and another for a second cluster
alias klient-dev='klient -bootstrappers "kafka-dev-broker-1:9092" -authkey "api_key_or_username" -authsecret "api_secret_or_password" -tls -tlsnoverify'

# now execute klient using the different aliases to connect to the different clusters
klient-prod -describe # describes the prod cluster
klient-dev -describe # describes the dev cluster
```

Optionally, adding these aliases to a script file and sourcing them from your `~/.bashrc` file will mean they are always available in all your future sessions. Though ensure any required security precautions regarding your password or api-keys are taken.

### Troubleshooting
In the event of unexpected behaviour, `klient` writes a log file to `~/klient.log` (by default). This may contain information to help diagnose and address any issues.

##  Describe
To describe a cluster run the below:

```bash
    klient -describe
``` 

The output shows partitions grouped by topics and then again by brokers:

```
describing cluster using bootstrappers: 'kafka-broker-1:9092,kafka-broker-2:9092,kafka-broker-3:909'

broker: kafka-broker-1:9092
> topic: example1
  > partition: 0
  > partition: 1
> topic: example2
  > partition: 0
  > partition: 1
broker: kafka-broker-2:9092
> topic: example1
  > partition: 2
  > partition: 3
> topic: example2
  > partition: 2
  > partition: 3
```

Note, brokers not acting as leader for any partition are not shown.

##  Topic Creation
To create a topic using the default replication factor and partition count, run the below:

```bash
klient -create "my-topic"
``` 

Alternatively, specify these attributes as required:

```bash
# specify a partition count of 3 and a replication factor of 2
klient -create "my-topic" -partitions 3 -replicas 2 
``` 

*Note, that many clusters have constraints around the number of replicas that are required for a topic. These will cause errors if not honoured. These errors may not clearly state the nature of the issue; instead referring to a general 'policy violation'.  Typically a production cluster will require at least three replicas. Container images running locally may  be configured to only accept a value of one.*

##  Topic Deletion
To delete a topic, run the below:

```bash
klient -delete "my-topic" 
``` 

## Writing to a Topic Interactively
To write to a topic, run the below:

```bash
   klient -write "my-topic" 
``` 

The terminal will then prompt for data to write to the topic. Enter the data as required and hit enter to write it. If successful, confirmation of the write will be displayed and the terminal then awaits input of the next message:

```bash
klient -write "my-topic"
enter data to publish to topic 'my-topic'. enter X to exit:
> hello world
wrote [hello world > my-topic]
> 
```

When this mode of writing is used, `klient` round-robins messages across any partitions. To have `klient` spread messages across partitions based on a digest of a given message key instead, run the below:

```bash
# specify the keyed flag
klient -write "my-topic" -keyed 
``` 

As previously, the terminal will then prompt for data to write to the topic. However now, on entering that data, a key is prompted for. Enter whatever value is to be used as the key. Providing an empty value will use a value of 'default', initially; for subsequent prompts it will assume the same key as was previously entered is to be re-used. 

As shown below:

```bash
klient -write "my-topic" -keyed
enter text to publish to topic 'my-topic'. enter X to exit:
> hello world
enter key (empty uses 'default')> key1
wrote [hello world > my-topic]
> hello world again
enter key (empty uses previous) >  # no key entered so `key1` from the previous entry is used
wrote [hello world again > my-topic]
> hello world yet again
enter key (empty uses previous) > key2 # a new key is entered so this is now used instead, and will be for any future empty key entries
wrote [hello world yet again > my-topic]
> 
```

*Note, that when attempting write to a topic, it must already exist. If it does not, create the topic first (see [Topic Creation](#topic-creation))*

## Writing to a Topic From Another Process (eg a File Reader) 
Data can be written directly to a topic by piping the output of another process, such as a file reader, into `klient`.  To do this, specify the `-unattended` flag, for unattended execution, and optionally the `-delimiter` flag to specify a non-default delimiter (the default is new line).  

The example below shows how to pipe data from a new-line delimited, `input.json` file into `klient`

```json
// file: input.json
{ "id": 1, "f1": "v1", "f2": "v2" }
{ "id": 2, "f1": "v1", "f2": "v2" }
{ "id": 3, "f1": "v1", "f2": "v2" }
```

```bash
# specify the name of the topic to write to and set the unattended flag
cat input.json | klient -write "my-topic" -unattended 
```

The input file may also specify a key to assign to each value. The example below shows how to specify a key and also how to use a non-new-line delimiter, in the case `|`:

```
// file: input.json, the k# values are the keys
{ "id": 1, "f1": "v1", "f2": "v2" }|k1|{ "id": 2, "f1": "v1", "f2": "v2" }|k2|{ "id": 3, "f1": "v1", "f2": "v2" }|k3|
```

```bash
# specify the -delimiter is a pipe and that the data is -keyed
cat input.json | klient -write "my-topic" -unattended -delimiter "|" -keyed 
```

*Note, that when attempting write to a topic, it must already exist. If it does not, create the topic first (see [Topic Creation](#topic-creation))*

## Reading from Topics
Reading from a topic can be undertaken in the following ways:

* Specifying a `partition` and an `offset` range (or accepting the defaults)
* Specifying a `partition` and a `time` range (or accepting the defaults)
* As an `exclusive consumer` (reading all messages from all partitions)
* As a `group consumer` (sharing the reading of messages with consumers in the same group)

Examples of the above approaches are given in the following sections

### Redirecting Topic Data to Another Process
Topic data can be written to `stdout` in a format suitable for ingestion by another process. 

For example, `json` data could be passed into `jq` for formatting or querying.  

To do this, specify the `-unattended` flag, for unattended execution; this will cause only the topic data to be written to `stdout`, delimited by a new line. Optionally the `-delimiter` flag can be provided to specify a non-new-line delimiter, for example if the data itself may contain new lines.

The example below redirects `json` topic data to `jq`:

```bash
klient -exclusive-read "my-topic" -unattended | jq
```

### Specifying a Partition & Offset Range
To read from a topic by `partition` and `offset` range, use the `-range-read` flag. 

The default settings for this mode are to read all messages from `partition` zero, and then continue reading indefinitely. 

Some, or all of these defaults can be overridden as required, as shown in the below examples.


```bash
# read all messages in partition 0, continue indefinitely 
klient -range-read "my-topic" 
```

```bash
# read all messages in partition 1, from offset 10, continue indefinitely 
klient -range-read "my-topic" -partition 1 -offset-from 10 
```

```bash
# read all messages in partition 0 between offset 10 and 20, then terminate
klient -range-read "my-topic" -offset-from 10 -offset-from 20
```

### Specifying a Partition & Time Range
To read from a topic by `partition` and `time` range, use the `-read-time` flag. 

The default settings for this mode are to read any messages from `partition` zero that were generated within the last minute and then continue reading indefinitely. 

Some, or all of these defaults can be overridden as required, as shown in the below examples.

```bash
# read all messages in partition 1 that were generated in the last minute (the default), continue reading indefinitely (the default)
klient -time-read "my-topic" -partition 1
```

```bash
# read all messages in partition 0 (the default) that were generated since the specific time stated, continue reading indefinitely (the default)
klient -time-read "my-topic" -time-from "15-02-2023 09:00:00"
```

```bash
# read all messages in partition 0 (the default) between the from and to dates specified, then terminate
klient -time-read "my-topic" -time-from "15-02-2023 09:00:00" -time-to "15-02-2023 10:00:00"
```

### Reading Exclusively
To read exclusively (from offset zero across all partitions), run the below:

```bash
klient -exclusive-read "my-topic" # specifiy the topic to read exclusively from
```

### Reading with Consumer Groups
To read as part of a consumer group, run the below:

```bash
# terminal 1
klient -group-read "my-topic" -group "my-group" # specify the topic to read from and the group to create (as it does not exist already)
```

Run the same again in further terminals to see reads then become distributed across the terminal group:

```bash
# terminal 2
klient -group-read "my-topic" -group "my-group" # specify the topic to read from and the group to join
```

```bash
# terminal 3
klient -group-read "my-topic" -group "my-group" # specify the topic to read from and the group to join
```

```bash
# terminal 4
klient -group-read "my-topic" -group "my-group" # specify the topic to read from and the group to join
```

Whichever method is used to read with `klient`, the resulting output is the the same. This is described below:

```bash
klient -read "my-topic"
[0/0 @ 15-02-2023 20:31:27]> [keyA] : msg1 # [#partition/#offset #timestamp]> [#key] : #value
[1/0 @ 15-02-2023 20:31:28]> [keyB] : msg2
[1/1 @ 15-02-2023 20:31:29]> [keyB] : msg3
[0/1 @ 15-02-2023 20:31:30]> [keyA] : msg4
[1/2 @ 15-02-2023 20:31:31]> [keyC] : msg5
[0/2 @ 15-02-2023 20:31:32]> [keyA] : msg6
```

## Contributing
Contributions and suggestions are welcome
