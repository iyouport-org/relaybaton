# relaybaton
A pluggable transport to circumvent Internet censorship with Encrypted SNI.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/iyouport-org/relaybaton?status.svg)](https://godoc.org/github.com/iyouport-org/relaybaton)
[![Go Report Card](https://goreportcard.com/badge/github.com/iyouport-org/relaybaton)](https://goreportcard.com/report/github.com/iyouport-org/relaybaton)

## Getting Started

### Prerequisites

### Install

```bash
go get github.com/iyouport-org/relaybaton
```

### Build

```CGO_ENABLED=1``` should be set in cross-compiling

```bash
make
```

## Deployment

For supporting ESNI features and hiding the IP address of the server from interception, the server should have a valid domain name and behind Cloudflare CDN.

Cloudflare CDN will provide TLS encryption with ESNI extension.

A MaxMind GeoLite2 Country database should be provided for GeoIP-based routing.

### Server
```sudo``` is required for listening on port 80

```bash
sudo relaybation server --config /path/to/server/config.toml
```

### Client
```bash
relaybation client --config /path/to/client/config.toml
```

A local SOCKS5 proxy server will listen on the local port which is given in the configuration file.

## Configuration

**config.toml** is the default configuration file, it should be located in the same path of the excutable binary file.

### Example

```toml
[client]
port = 1080
server = "example.com"
username = "username"
password = "password"
proxy_all = true

[dns]
type = "doh"
server = "cloudflare-dns.com"
addr = "1.1.1.1"

[log]
file = "./log.xml"
level = "trace"

[server]
port=80
pretend="https://www.kernel.org"
timeout=15
secure=false
cert_file=""
key_file=""

[db]
type="sqlite3"
username="root"
password="password"
host="localhost"
port=1433
database="relaybaton.db"
```

### Description of the fields

|      Field       | TOML Type |                      Go Type                      |                       Description                        |
| :--------------: | :-------: | :-----------------------------------------------: | :------------------------------------------------------: |
|     log.file     |  String   |                      os.File                      |                   filename of log file                   |
|    log.level     |  String   |      github.com/sirupsen/logrus logrus.Level      |                minimum log level to write                |
|     dns.type     |  String   | github.com/iyouport-org/relaybaton config.DNSType |                   type of DNS resolver                   |
|    dns.server    |  String   |                      string                       |              server name of the DNS server               |
|     dns.addr     |  String   |                     net.Addr                      |               IP address of the DNS server               |
|   client.port    |  Integer  |                      uint16                       |             local port that client listen to             |
|  client.server   |  String   |                      string                       |                domain name of the server                 |
| client.username  |  String   |                      string                       |                  username of the client                  |
| client.password  |  String   |                      string                       |                  password of the client                  |
| client.proxy_all |  Boolean  |                       bool                        |                   if proxy all traffic                   |
|   server.port    |  Integer  |                      uint16                       |                port that server listen to                |
|  server.pretend  |  String   |                      url.URL                      | domain name of the website that the server pretend to be |
|  server.timeout  |  Integer  |                   time.Duration                   |                 timeout for no response                  |
|  server.secure   |  Boolean  |                       bool                        |              if the server running with TLS              |
| server.cert_file |  String   |                    os.FileInfo                    |                cert file of a TLS server                 |
| server.key_file  |  String   |                    os.FileInfo                    |                 key file of a TLS server                 |
|     db.type      |  String   | github.com/iyouport-org/relaybaton config.dbType  |                   type of the database                   |
|   db.username    |  String   |                      string                       |             username for database connection             |
|   db.password    |  String   |                      string                       |             password for database connection             |
|     db.host      |  String   |                      string                       |             hostname for database connection             |
|     db.port      |  Integer  |                      uint16                       |               port for database connection               |
|   db.database    |  String   |                      string                       |                     name of database                     |

## Built With

* [github.com/cloudflare/tls-tris](https://github.com/cloudflare/tls-tris/tree/pwu/esni) - crypto/tls, now with 100% more 1.3.
* [github.com/emirpasic/gods](https://github.com/emirpasic/gods) - Implementation of various data structures and algorithms in Go.
* [github.com/eycorsican/go-tun2socks](https://github.com/eycorsican/go-tun2socks) - A tun2socks implementation written in Go.
* [github.com/fasthttp/websocket](https://github.com/fasthttp/websocket) - This fork adds fasthttp support to the latest version of gorilla/websocket.
* [github.com/jinzhu/gorm](https://github.com/jinzhu/gorm) - The fantastic ORM library for Golang.
* [github.com/miekg/dns](https://github.com/miekg/dns) - Complete and usable DNS library.
* [github.com/oschwald/geoip2-golang](https://github.com/oschwald/geoip2-golang) - This library reads MaxMind GeoLite2 and GeoIP2 databases.
* [github.com/panjf2000/gnet](https://github.com/panjf2000/gnet) - An event-driven networking framework that is fast and lightweight.
* [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus) - A structured logger for Go.
* [github.com/spf13/cobra](https://github.com/spf13/cobra) - Cobra is both a library for creating powerful modern CLI applications as well as a program to generate applications and command files.
* [github.com/spf13/viper](https://github.com/spf13/viper) - A complete configuration solution for Go applications including 12-Factor apps.
* [github.com/valyala/fasthttp](https://github.com/valyala/fasthttp) - Fast HTTP implementation for Go.
* [go.uber.org/fx](https://github.com/uber-go/fx) - An application framework for Go.
* [github.com/andlabs/ui](https://github.com/andlabs/ui) - A library that aims to provide simple GUI software development in Go.
* [github.com/go-playground/validator](https://github.com/go-playground/validator) - Validator implements value validations for structs and individual fields based on tags.
* [github.com/google/gopacket](https://github.com/google/gopacket) - This library provides packet decoding capabilities for Go.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/iyouport-org/relaybaton/tags). 

## Authors

- [onoketa]((https://github.com/onoketa))

See also the list of [contributors](https://github.com/iyouport-org/relaybaton/contributors) who participated in this project.

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE.md) file for details

## Acknowledgments

* Cloudflare
