package socks5

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
             o  X'07' Command not supported
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

const (
	RepSucceeded                     = 0
	RepServerFailure                 = 1
	RepConnectionNotAllowedByRuleset = 2
	RepNetworkUnreachable            = 3
	RepHostUnreachable               = 4
	RepConnectionRefused             = 5
	RepTTLExpired                    = 6
	RepCmdNotSupported               = 7
	RepATypNotSupported              = 8
)

type Reply struct {
	ver     byte
	rep     byte
	rsv     byte
	aTyp    byte
	bndAddr []byte
	bndPort uint16
}

func NewReply(rep byte, aTyp byte, bndAddr []byte, bndPort uint16) Reply {
	return Reply{
		ver:     5,
		rep:     rep,
		rsv:     0,
		aTyp:    aTyp,
		bndAddr: bndAddr,
		bndPort: bndPort,
	}
}
