package socks5

import (
	"bytes"
	"errors"

	log "github.com/sirupsen/logrus"
)

/*
   The client connects to the server, and sends a version
   identifier/method selection message:

                   +----+----------+----------+
                   |ver | nMethods | methods  |
                   +----+----------+----------+
                   | 1  |    1     | 1 to 255 |
                   +----+----------+----------+

   The ver field is set to X'05' for this version of the protocol.  The
   nMethods field contains the number of method identifier octets that
   appear in the methods field.
*/

type MethodRequest struct {
	ver      byte
	nMethods byte
	methods  []Method
}

func NewMethodRequestFrom(b []byte) (mr MethodRequest, err error) {
	reader := bytes.NewReader(b)
	ver := make([]byte, 1)
	n, err := reader.Read(ver)
	if err != nil {
		log.Error(err)
		return
	}
	if n != 1 {
		err = errors.New("SOCKS5 version not read")
		log.WithField("ver", ver).Error(err)
		return
	}
	if ver[0] != 5 {
		err = errors.New("SOCKS5 version error")
		log.Error(err)
		return
	}
	mr.ver = ver[0]

	mMethods := make([]byte, 1)
	n, err = reader.Read(mMethods)
	if err != nil {
		log.Error(err)
		return
	}
	if n != 1 {
		err = errors.New("SOCKS5 number of method not read")
		log.Error(err)
		return
	}
	mr.nMethods = mMethods[0]

	methods := make([]byte, mMethods[0])
	n, err = reader.Read(methods)
	if err != nil {
		log.Error(err)
		return
	}
	if n != int(mMethods[0]) {
		err = errors.New("number of method not match")
		log.Error(err)
		return
	}
	mr.methods = methods
	return
}

func (mr MethodRequest) Methods() []Method {
	return mr.methods
}
