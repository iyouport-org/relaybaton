package util

import (
	"encoding/binary"
	"fmt"
	"github.com/iyouport-org/socks5"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"net/http"
)

func Uint16ToBytes(n uint16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, n)
	return buf
}

func Header2Fields(header http.Header, body io.ReadCloser) log.Fields {
	fields := log.Fields{}
	for k, v := range header {
		if len(v) > 1 {
			for i, str := range v {
				fields[fmt.Sprintf("%s[%d]", k, i)] = str
			}
		} else {
			fields[k] = v[0]
		}
	}
	bodyStr, err := ioutil.ReadAll(body)
	if err == nil {
		fields["body"] = bodyStr
	} else {
		log.WithFields(fields).Warn(err)
	}
	err = body.Close()
	if err != nil {
		log.WithFields(fields).Warn(err)
	}
	return fields
}

func GetDstReq(request socks5.Request) string {
	switch request.Atyp {
	case socks5.ATYPIPv4:
		return fmt.Sprintf("%s:%d", net.IP(request.DstAddr).To4().String(), binary.BigEndian.Uint16(request.DstPort))
	case socks5.ATYPIPv6:
		return fmt.Sprintf("[%s]:%d", net.IP(request.DstAddr).To16().String(), binary.BigEndian.Uint16(request.DstPort))
	default: //Domain
		return fmt.Sprintf("%s:%d", string(request.DstAddr[1:]), binary.BigEndian.Uint16(request.DstPort))
	}
}
