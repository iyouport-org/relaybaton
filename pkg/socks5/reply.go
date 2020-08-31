package socks5

import (
	"github.com/iyouport-org/relaybaton/pkg/util"
)

/*
   The SOCKS request information is sent by the client as soon as it has
   established a connection to the SOCKS server, and completed the
   authentication negotiations.  The server evaluates the request, and
   returns a reply formed as follows:

        +----+-----+-------+------+----------+----------+
        |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
        +----+-----+-------+------+----------+----------+
        | 1  |  1  | X'00' |  1   | Variable |    2     |
        +----+-----+-------+------+----------+----------+

     Where:

          o  VER    protocol version: X'05'
          o  REP    Reply field:
             o  X'00' succeeded
             o  X'01' general SOCKS server failure
             o  X'02' connection not allowed by ruleset
             o  X'03' Network unreachable
             o  X'04' Host unreachable
             o  X'05' Connection refused
             o  X'06' TTL expired
             o  X'07' Cmd not supported
             o  X'08' Address type not supported
             o  X'09' to X'FF' unassigned
          o  RSV    RESERVED
          o  ATYP   address type of following address
             o  IP V4 address: X'01'
             o  DOMAINNAME: X'03'
             o  IP V6 address: X'04'
          o  BND.ADDR       server bound address
          o  BND.PORT       server bound port in network octet order

   Fields marked RESERVED (RSV) must be set to X'00'.
*/

type Reply struct {
	ver byte
	Rep
	rsv byte
	ATyp
	bndAddr []byte
	bndPort uint16
}

func NewReply(rep Rep, aTyp ATyp, bndAddr []byte, bndPort uint16) Reply {
	return Reply{
		ver:     5,
		Rep:     rep,
		rsv:     0,
		ATyp:    aTyp,
		bndAddr: bndAddr,
		bndPort: bndPort,
	}
}

func (reply Reply) Pack() []byte {
	b := []byte{reply.ver, reply.Rep, reply.rsv, reply.ATyp}
	b = append(b, reply.bndAddr...)
	b = append(b, util.Uint16ToBytes(reply.bndPort)...)
	return b
}
