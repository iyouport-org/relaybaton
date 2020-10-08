package socks5

import (
	"bytes"
	"encoding/binary"
	"errors"

	log "github.com/sirupsen/logrus"
)

/*
   The SOCKS request is formed as follows:

        +----+-----+-------+------+----------+----------+
        |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
        +----+-----+-------+------+----------+----------+
        | 1  |  1  | X'00' |  1   | Variable |    2     |
        +----+-----+-------+------+----------+----------+

     Where:

          o  VER    protocol version: X'05'
          o  CMD
             o  CONNECT X'01'
             o  BIND X'02'
             o  UDP ASSOCIATE X'03'
          o  RSV    RESERVED
          o  ATYP   address type of following address
             o  IP V4 address: X'01'
             o  DOMAINNAME: X'03'
             o  IP V6 address: X'04'
          o  DST.ADDR       desired destination address
          o  DST.PORT desired destination port in network octet
             order
*/

type Request struct {
	ver byte
	Cmd
	rsv byte
	ATyp
	DstAddr []byte
	DstPort uint16
}

func NewRequestFrom(b []byte) (request Request, err error) {
	reader := bytes.NewReader(b)
	ver := make([]byte, 1)
	n, err := reader.Read(ver)
	if err != nil {
		log.Error(err)
		return request, err
	}
	if n != 1 {
		err = errors.New("SOCKS5 version not read")
		log.Error(err)
		return request, err
	}
	if ver[0] != 5 {
		err = errors.New("SOCKS5 version error")
		log.Error(err)
		return request, err
	}
	request.ver = ver[0]

	cmd := make([]byte, 1)
	n, err = reader.Read(cmd)
	if err != nil {
		log.Error(err)
		return request, err
	}
	if n != 1 {
		err = errors.New("SOCKS5 request command not read")
		log.Error(err)
		return request, err
	}
	if (cmd[0] != CmdConnect) && (cmd[0] != CmdBind) && (cmd[0] != CmdUDPAssociate) {
		err = errors.New("SOCKS5 request command error")
		log.Error(err)
		return request, err
	}
	request.Cmd = cmd[0]

	rsv := make([]byte, 1)
	n, err = reader.Read(rsv)
	if err != nil {
		log.Error(err)
		return request, err
	}
	if n != 1 {
		err = errors.New("SOCKS5 reserved not read")
		log.Error(err)
		return request, err
	}
	if rsv[0] != 0 {
		log.Warn("SOCKS5 reserved not 0")
	}
	request.rsv = rsv[0]

	aTyp := make([]byte, 1)
	n, err = reader.Read(aTyp)
	if err != nil {
		log.Error(err)
		return request, err
	}
	if n != 1 {
		err = errors.New("SOCKS5 address type not read")
		log.Error(err)
		return request, err
	}
	if (aTyp[0] != ATypeIPv4) && (aTyp[0] != ATypeDomainName) && (aTyp[0] != ATypeIPv6) {
		log.Warn("SOCKS5 reserved not 0")
	}
	request.ATyp = aTyp[0]
	var dstAddr []byte
	switch request.ATyp {
	case ATypeIPv4:
		dstAddr = make([]byte, 4)
		n, err = reader.Read(dstAddr)
		if err != nil {
			log.Error(err)
			return request, err
		}
		if n != 4 {
			err = errors.New("SOCKS5 address type IPv4 not read")
			log.Error(err)
			return request, err
		}
	case ATypeIPv6:
		dstAddr = make([]byte, 16)
		n, err = reader.Read(dstAddr)
		if err != nil {
			log.Error(err)
			return request, err
		}
		if n != 16 {
			err = errors.New("SOCKS5 address type IPv6 not read")
			log.Error(err)
			return request, err
		}
	case ATypeDomainName:
		domainNameLen := make([]byte, 1)
		n, err = reader.Read(domainNameLen)
		if err != nil {
			log.Error(err)
			return request, err
		}
		if n != 1 {
			err = errors.New("SOCKS5 address type domain name length not read")
			log.Error(err)
			return request, err
		}
		if domainNameLen[0] == 0 {
			err = errors.New("SOCKS5 address type domain name length is 0")
			log.Error(err)
			return request, err
		}

		domainName := make([]byte, domainNameLen[0])
		n, err = reader.Read(domainName)
		if err != nil {
			log.Error(err)
			return request, err
		}
		if n != int(domainNameLen[0]) {
			err = errors.New("SOCKS5 address type domain name not read")
			log.Error(err)
			return request, err
		}
		dstAddr = append(domainNameLen, domainName...)
	default:
		err = errors.New("unknown address type")
		log.Error(err)
		return request, err
	}
	request.DstAddr = dstAddr

	dstPort := make([]byte, 2)
	n, err = reader.Read(dstPort)
	if err != nil {
		log.Error(err)
		return request, err
	}
	if n != 2 {
		err = errors.New("SOCKS5 dst port not read")
		log.Error(err)
		return request, err
	}
	request.DstPort = binary.BigEndian.Uint16(dstPort)

	return
}
