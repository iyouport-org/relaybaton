# relaybaton

A pluggable transport to circumvent Internet censorship with Encrypted SNI.

The project will be updated following the adoption of ECH.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/iyouport-org/relaybaton?status.svg)](https://pkg.go.dev/github.com/iyouport-org/relaybaton)
[![Go Report Card](https://goreportcard.com/badge/github.com/iyouport-org/relaybaton)](https://goreportcard.com/report/github.com/iyouport-org/relaybaton)

## Getting Started

### Prerequisites

### Install

```bash
go get github.com/iyouport-org/relaybaton
```

### Build

#### CLI

`CGO_ENABLED=1` should be set in cross-compiling

```bash
make
```

#### C++ static library

```shell
make desktop
```

#### Android library

```shell
make mobile
```

## Deployment

For supporting ESNI features and hiding the IP address of the server from interception, the server should have a valid domain name and behind Cloudflare CDN.

Cloudflare CDN will provide TLS encryption with ESNI extension.

### Server

`sudo` is required for listening on port 80

```bash
sudo relaybation server --config /path/to/server/config.toml
```

### Client

```bash
relaybation client --config /path/to/client/config.toml
```

A local proxy server will listen on the local ports which given in the configuration file.

## Configuration

### Example

```toml
[client]
port = 1080
http_port = 1088
redir_port = 1090
server = "example.com"
username = "username"
password = "password"
proxy_all = true

[server]
port = 80
admin_password = "password"

[db]
type = "sqlite3"
username = "root"
password = "password"
host = "localhost"
port = 1433
database = "relaybaton.db"

[dns]
type = "doh"
server = "cloudflare-dns.com"
addr = "1.1.1.1"

[log]
file = "./log.xml"
level = "trace"

```

### Description of the fields

|         Field         | TOML Type |                      Go Type                      |             Description             |
| :-------------------: | :-------: | :-----------------------------------------------: | :---------------------------------: |
|      client.port      |  Integer  |                      uint16                       |  SOCKS5 port that client listen to  |
|   client.http_port    |  Integer  |                      uint16                       |   HTTP port that client listen to   |
|   client.redir_port   |  Integer  |                      uint16                       | Redirect port that client listen to |
|     client.server     |  String   |                      string                       |      domain name of the server      |
|    client.username    |  String   |                      string                       |       username of the client        |
|    client.password    |  String   |                      string                       |       password of the client        |
|   client.proxy_all    |  Boolean  |                       bool                        |        if proxy all traffic         |
|      server.port      |  Integer  |                      uint16                       |     port that server listen to      |
| server.admin_password |  String   |                      string                       |     password of account "admin"     |
|        db.type        |  String   | github.com/iyouport-org/relaybaton config.dbType  |        type of the database         |
|      db.username      |  String   |                      string                       |  username for database connection   |
|      db.password      |  String   |                      string                       |  password for database connection   |
|        db.host        |  String   |                      string                       |  hostname for database connection   |
|        db.port        |  Integer  |                      uint16                       |    port for database connection     |
|      db.database      |  String   |                      string                       |          name of database           |
|       dns.type        |  String   | github.com/iyouport-org/relaybaton config.DNSType |        type of DNS resolver         |
|      dns.server       |  String   |                      string                       |    server name of the DNS server    |
|       dns.addr        |  String   |                     net.Addr                      |    IP address of the DNS server     |
|       log.file        |  String   |                      os.File                      |        filename of log file         |
|       log.level       |  String   |      github.com/sirupsen/logrus logrus.Level      |     minimum log level to write      |

## Built With

- [github.com/cloudflare/tls-tris](https://github.com/cloudflare/tls-tris/tree/pwu/esni) - crypto/tls, now with 100% more 1.3.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/iyouport-org/relaybaton/tags).

## Authors

- [onoketa](<(https://github.com/onoketa)>)

See also the list of [contributors](https://github.com/iyouport-org/relaybaton/contributors) who participated in this project.

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details

## Acknowledgments

- Cloudflare
