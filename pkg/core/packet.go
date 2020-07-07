package core

import (
	"encoding/binary"
	"errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"relaybaton/pkg/socks5"
	"relaybaton/pkg/util"
)

/*
(octal)
+--------+---------+----------+----------+
| HEADER |   SESS  |  PADDING |  PAYLOAD |
+--------+---------+----------+----------+
|    1   | 6 or 18 | Variable | Variable |
+--------+---------+----------+----------+

- HEADER	Header of the packet
	(binary)
	+-----+---------+-----------+---------+--------------+
	| TYP | NETWORK | SESS ATYP | CMD/REP | REQ/REP ATYP |
	+-----+---------+-----------+---------+--------------+
	|  2  |    1    |     1     |    3    |       1      |
	+-----+---------+-----------+---------+--------------+

	- TYP			Type of packet
		- 00		Request / Reply
		- 01		Data
		- 10		Close
		- 11		(Reserved)
	- NETWORK		Network protocol
		- 0			TCP
		- 1			UDP
	- SESS ATYP		Address type of session
		- 0			IPv6
		- 1			IPv4
	- CMD/REP		SOCKS5 Command	/	Reply
		- 000		(Reserved)		/	Succeeded
		- 001		Connect			/	General SOCKS server failure
		- 010		Bind			/	Connection not allowed by ruleset
		- 011		UDP Associate	/	Network unreachable
		- 100		(Reserved)		/	Host unreachable
		- 101		(Reserved)		/	Connection refused
		- 110		(Reserved)		/	TTL expired
		- 111		(Reserved)		/	(Reserved)
	- REQ/REP ATYP	Address type of request / reply
		- 0			IPv4
		- 1			IPv6

- SESS		Session of the packet
	(octal)
	+------+---------+
	| PORT |   ADDR  |
	+------+---------+
	|   2  | 4 or 16 |
	+------+---------+
	- PORT	Port of client-side connection
	- ADDR	IP Address of client-side connection
			The length is 4 if the session address type in the Header is IPv4, or 16 if it is IPv6.

- PADDING	Padding of the packet
	(octal)
	+-----+------+
	| LEN | DATA |
	+-----+------+
	|  2  |  LEN |
	+-----+------+
	- LEN	Length of padding data
	- DATA	Random padding data

- PAYLOAD	Payload of the packet

	(If the packet type in Header is	Request	/	Reply, octal)
	+------+---------+
	| PORT |   ADDR  |
	+------+---------+
	|   2  | 4 or 16 |
	+------+---------+
	- PORT			Desired destination port	/	Server bound port
	- ADDR			Desired destination address	/	Server bound address
					The length is 4 if the address type of request / reply in the Header is IPv4, or 16 if it is IPv6.

	(If the packet type in Header is Data, octal)
	+----------+
	|   DATA   |
	+----------+
	| Variable |
	+----------+
	- DATA			Data to be transmitted

	(No payload if the packet type in Header is Close)
*/

type PacketType = byte
type PacketNetwork = byte
type PacketATyp = byte
type PacketCmdRep = byte

const (
	TypeRequestReply = PacketType(0x00)
	TypeData         = PacketType(0x01)
	TypeClose        = PacketType(0x02)

	NetWorkTCP = PacketNetwork(0x00)
	NetWorkUDP = PacketNetwork(0x01)

	ATypIPv6 = PacketATyp(0x00)
	ATypIPv4 = PacketATyp(0x01)

	CmdConnect      = PacketCmdRep(0x01)
	CmdBind         = PacketCmdRep(0x02)
	CmdUDPAssociate = PacketCmdRep(0x03)

	RepSucceeded          = PacketCmdRep(0x00)
	RepServerFailure      = PacketCmdRep(0x01) //General SOCKS server failure
	RepNotAllowed         = PacketCmdRep(0x02) //Connection not allowed by ruleset
	RepNetworkUnreachable = PacketCmdRep(0x03)
	RepHostUnreachable    = PacketCmdRep(0x04)
	RepConnRefused        = PacketCmdRep(0x05) //Connection refused
	RepTTLExpired         = PacketCmdRep(0x06)
)

type Packet struct {
	header  byte
	session []byte
	padding []byte
	payload []byte

	PacketType
	PacketNetwork
	SessATyp PacketATyp
	PacketCmdRep
	ReqRepATyp PacketATyp
}

func Decode(header byte) (pt PacketType, network PacketNetwork, sessATyp PacketATyp, cmdRep PacketCmdRep, reqRepATyp PacketATyp) {
	maskType := byte(0xC0)       //1100 0000
	maskNetwork := byte(0x20)    //0010 0000
	maskSessATyp := byte(0x10)   //0001 0000
	maskCmdRep := byte(0x0E)     //0000 1110
	maskReqRepATyp := byte(0x01) //0000 0001

	pt = (header & maskType) >> 6
	network = (header & maskNetwork) >> 5
	sessATyp = (header & maskSessATyp) >> 4
	cmdRep = (header & maskCmdRep) >> 1
	reqRepATyp = header & maskReqRepATyp
	return
}

func Encode(pt PacketType, network PacketNetwork, sessATyp PacketATyp, cmdRep PacketCmdRep, reqRepATyp PacketATyp) byte {
	return pt<<6 | network<<5 | sessATyp<<4 | cmdRep<<1 | reqRepATyp
}

func NewDataPacket(network PacketNetwork, sessATyp PacketATyp, session []byte, payload []byte) Packet {
	return Packet{
		header:  Encode(TypeData, network, sessATyp, 0, 0),
		session: session,
		padding: []byte{0, 0},
		payload: payload,
	}
}

func NewRequestPacket(network PacketNetwork, sessATyp PacketATyp, cmd PacketCmdRep, reqRepATyp PacketATyp, session []byte, dstAddr []byte, dstPort uint16) Packet {
	return Packet{
		header:  Encode(TypeRequestReply, network, sessATyp, cmd, reqRepATyp),
		session: session,
		padding: RandomPadding(1 << 10),
		payload: append(util.Uint16ToBytes(dstPort), dstAddr...),
	}
}

func NewReplyPacket(network PacketNetwork, sessATyp PacketATyp, rep PacketCmdRep, reqRepATyp PacketATyp, session []byte, bndAddr []byte, bndPort uint16) Packet {
	return Packet{
		header:  Encode(TypeRequestReply, network, sessATyp, rep, reqRepATyp),
		session: session,
		padding: RandomPadding(1 << 10),
		payload: append(util.Uint16ToBytes(bndPort), bndAddr...),
	}
}

func NewClosePacket(network PacketNetwork, sessATyp PacketATyp, session []byte) Packet {
	return Packet{
		header:  Encode(TypeClose, network, sessATyp, 0, 0),
		session: session,
		padding: RandomPadding(1 << 10),
		payload: []byte{},
	}
}

func Unpack(b []byte) (Packet, error) {
	packet := Packet{}
	cursor := 0
	packet.header = b[cursor]
	packet.PacketType, packet.PacketNetwork, packet.SessATyp, packet.PacketCmdRep, packet.ReqRepATyp = Decode(packet.header)
	cursor++

	sessLen := packet.getSessLen()
	//log.WithField("sessLen", sessLen).Debug()
	packet.session = b[cursor : cursor+sessLen]
	cursor += sessLen
	//log.WithField("cursor", cursor).Debug()

	paddingLen := int(binary.BigEndian.Uint16(b[cursor : cursor+2]))
	//log.WithField("paddingLen", paddingLen).Debug()
	if len(b) < cursor+2+paddingLen {
		err := errors.New("out of slice")
		log.WithFields(log.Fields{
			"len":        len(b),
			"data":       b,
			"cursor":     cursor,
			"paddingLen": paddingLen,
		}).Error(err)
		return packet, err
	}
	packet.padding = b[cursor : cursor+2+paddingLen]
	cursor += 2 + paddingLen
	//log.WithField("cursor", cursor).Debug()

	packet.payload = b[cursor:]
	return packet, nil
}

func (packet *Packet) Pack() []byte {
	b := []byte{packet.header}
	b = append(b, packet.session...)
	b = append(b, packet.padding...)
	b = append(b, packet.payload...)
	return b
}

func (packet *Packet) GetSession() net.Addr {
	port := binary.BigEndian.Uint16(packet.payload[:2])
	addr := packet.payload[2:]
	if packet.PacketNetwork == NetWorkTCP {
		return &net.TCPAddr{
			IP:   addr,
			Port: int(port),
		}
	} else { //UDP
		return &net.UDPAddr{
			IP:   addr,
			Port: int(port),
		}
	}
}

func (packet *Packet) getSessLen() int {
	if packet.SessATyp == ATypIPv4 {
		return 6
	} else if packet.SessATyp == ATypIPv6 {
		return 18
	} else {
		//TODO
		return 0
	}
}

func RandomPadding(maxLen uint16) []byte {
	paddingLen := uint16(rand.Intn(int(maxLen)))
	b := util.Uint16ToBytes(paddingLen)
	paddingData := make([]byte, paddingLen)
	rand.Read(paddingData)
	return append(b, paddingData...)
}

func GetSrcSession(addr *net.TCPAddr) (PacketATyp, []byte) {
	var sessTyp PacketATyp
	ip := addr.IP
	if ip.To4() != nil {
		ip = ip.To4()
		sessTyp = ATypIPv4
	} else {
		ip = ip.To16()
		sessTyp = ATypIPv6
	}
	return sessTyp, append(util.Uint16ToBytes(uint16(addr.Port)), ip...)
}

func ATypConvert(aTyp byte) byte {
	switch aTyp {
	case socks5.ATypeIPv4:
		return ATypIPv4
	case socks5.ATypeIPv6:
		return ATypIPv6
	/*case ATypIPv4:
	return socks5.ATypeIPv4*/
	case ATypIPv6:
		return socks5.ATypeIPv6
	default:
		//TODO
		return 0
	}
}
