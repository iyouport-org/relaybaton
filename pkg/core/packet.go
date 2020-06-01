package core

const (
	TypeRequest = byte(0x00)
	TypeData    = byte(0x01)
	TypeClose   = byte(0x02)
	TypeReply   = byte(0x03)

	ATypIPv4 = byte(0x00)
	ATypIPv6 = byte(0x01)

	MaskType        = byte(0x30)
	MaskSessionAtyp = byte(0x0C)
	MaskRequestAtyp = byte(0x03)
)

type Packet struct {
	typs    byte
	session []byte
	payload []byte

	typ         byte
	sessionATyp byte
	requestATyp byte
}

func NewPacket() Packet {
	return Packet{} //TODO
}

func Unpack(b []byte) Packet {
	msg := Packet{}
	msg.typs = b[0]
	var lenSession int
	msg.typ, msg.sessionATyp, msg.requestATyp = DecodeTyps(msg.typs)
	if msg.sessionATyp == ATypIPv4 {
		lenSession = 4
	} else if msg.sessionATyp == ATypIPv6 {
		lenSession = 16
	}
	session := b[1 : 1+lenSession]
	msg.session = session
	msg.payload = b[1+lenSession:]
	return msg
}

func (msg Packet) Pack() []byte {
	b := []byte{msg.typs}
	b = append(b, msg.session...)
	b = append(b, msg.payload...)
	return b
}

func (msg Packet) GetType() {

}

func (msg Packet) GetSession() []byte {
	return msg.session
}

func DecodeTyps(mType byte) (typ byte, sessionATyp byte, reqATyp byte) {
	typ = (mType & MaskType) >> 4
	sessionATyp = (mType & MaskSessionAtyp) >> 2
	reqATyp = mType & MaskRequestAtyp
	return
}
