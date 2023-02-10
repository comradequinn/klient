Hi, 

I posted recently about a new command line Kafka client I've been developing for personal and commercial use; I now have some updates to share

Thanks to requests and suggestions on here, other subs and actual commercial usage, `klient` now has a completely revamped CLI and the following features:

* Simple and intuitive CLI (command line interface)
* Describe clusters by outputing information about brokers, topics and partitions
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
* Confidently backed by a solid suite of unit and integration tests

As a recap, or for those who  missed the original post, unlike the standard scripts, and many binary clients, `klient` is a native, statically-compiled, binary. It uses [segmentio/go-kafka](https://github.com/segmentio/kafka-go) internally, which means `CGO` can be disabled during compilation.

This makes it very portable, especially when using `scratch`-based container images.

You can get full details in the [README](https://github.com/comradequinn/klient#readme) and you can download pre-built binaries for your preferred system [here](https://github.com/comradequinn/klient#installation) or browse the releases [here](https://github.com/comradequinn/klient/releases)

Appreciate there's some other options out there, but I hope some of you find it useful :-)