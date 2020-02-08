# relaybaton
A pluggable transport to circumvent Internet censorship

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/iyouport-org/relaybaton?status.svg)](https://godoc.org/github.com/iyouport-org/relaybaton)
[![Go Report Card](https://goreportcard.com/badge/github.com/iyouport-org/relaybaton)](https://goreportcard.com/report/github.com/iyouport-org/relaybaton)

## Getting Started

### Prerequisites

For supporting ESNI features, the [pwu/esni](https://github.com/cloudflare/tls-tris/tree/pwu/esni) branch of [cloudflare/tls-tris](https://github.com/cloudflare/tls-tris) should be used in building instead of the default ```go build```

See [tls-tris/README.md](https://github.com/cloudflare/tls-tris/blob/pwu/esni/README.md) for instructions of building tls-tris.

### Install

```bash
go get -u github.com/iyouport-org/relaybaton
```

### Build

```bash
tls-tris/_dev/go.sh build -o relaybaton github.com/iyouport-org/relaybaton/main
```

## Deployment

For supporting ESNI features and hiding the IP address of the server from interception, the server should have a valid domain name and behind Cloudflare CDN.

Cloudflare CDN will provide TLS encryption with ESNI extension.

### Configuration file

**config.toml** is the default configuration file, it should be located in the same path of the excutable binary file.

#### Example

```toml
log_file="./log.txt"

[client]
server="example.com"
port=1081
username="username"
password="password"
doh="cloudflare"    #cloudflare, quad9

[server]
port=80
pretend="www.kernel.org"
doh="null"    #null, cloudflare, quad9

[db]
type="sqlite3" #sqlite3, mysql, postgresql, sqlserver
username="root"
password="password"
host="localhost"
port=1433
database="relaybaton.db"
```

#### Explanation of the fields

|      Field      | TOML Type |    Type     |                         Explanation                          |
| :-------------: | :-------: | :---------: | :----------------------------------------------------------: |
|    log_file     |  String   |  filename   |                   the filename of log file                   |
|  client.server  |  String   | domain name |                the domain name of the server                 |
|   client.port   |  Integer  |  TCP port   |            the local proxy port client listen to             |
| client.username |  String   |   string    |         the client username, for user authentication         |
| client.password |  String   |   string    |         the client password, for user authentication         |
|   client.doh    |  String   |   string    |        the DNS over HTTPS service used in the client         |
|   server.port   |  Integer  |  TCP port   |           the port server listen to (normally 80)            |
| server.pretend  |  String   | domain name | the domain name of the website that the server pretend to be |
|   server.doh    |  String   |   string    |        the DNS over HTTPS service used in the server         |
|     db.type     |  String   |   string    |                   the type of the database                   |
|   db.username   |  String   |   string    |           the username for the database connection           |
|   db.password   |  String   |   string    |           the password for the database connection           |
|     db.host     |  String   |   string    |           the hostname for the database connection           |
|     db.port     |  Integer  |  TCP port   |             the port for the database connection             |
|   db.database   |  String   |   string    |                   the name of the database                   |

### Server
```sudo``` is required for listening on port 80

```bash
sudo relaybation server
```

### Client
```bash
relaybaton client
```

## Built With

* [logrus](https://github.com/sirupsen/logrus) - Structured, pluggable logging for Go. 
* [gorilla/websocket](https://github.com/gorilla/websocket) -  A fast, well-tested and widely used WebSocket implementation for Go.
* [viper](https://github.com/spf13/viper) - Go configuration with fangs

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/iyouport-org/relaybaton/tags). 

## Authors

- [onoketa]((https://github.com/onoketa))

See also the list of [contributors](https://github.com/iyouport-org/relaybaton/contributors) who participated in this project.

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE.md) file for details

## Acknowledgments

* [likexian/doh-go](https://github.com/likexian/doh-go) -  DNS over HTTPS (DoH) Golang implementation
* [txthinking/socks5](https://github.com/txthinking/socks5) - SOCKS Protocol Version 5 Library in Go
* Cloudflare
