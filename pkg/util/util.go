package util

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
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
