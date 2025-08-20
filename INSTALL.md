# Installing Klient

If `go` is available on the machine, `klient` can be installed quickly by running the below.

```bash
go install github.com/comradequinn/klient
```

Alternatively, you can install `klient` by downloading the appropriate tarball for your `os` from the [releases](https://github.com/comradequinn/klient/releases) page. Extract the binary and place it somewhere accessible to your `$PATH` variable.

Optionally, `linux` users can have the below script quickly do this for them.

```bash
export VERSION="v1.2.0"; export OS="linux-amd64"; wget "https://github.com/comradequinn/klient/releases/download/${VERSION}/klient-${VERSION}-${OS}.tar.gz" && tar -xf "klient-${VERSION}-${OS}.tar.gz" && rm -f "klient-${VERSION}-${OS}.tar.gz" && chmod +x klient && sudo mv klient /usr/local/bin/
```