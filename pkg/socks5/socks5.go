package socks5

type Method = byte
type Cmd = byte
type ATyp = byte
type Rep = byte

const (
	MethodNoAuthRequired   = Method(0x00)
	MethodGSSAPI           = Method(0x01)
	MethodUsernamePassword = Method(0x02)
	MethodNoAcceptable     = Method(0xFF)

	ATypeIPv4       = ATyp(0x01)
	ATypeDomainName = ATyp(0x03)
	ATypeIPv6       = ATyp(0x04)

	CmdConnect      = Cmd(0x01)
	CmdBind         = Cmd(0x02)
	CmdUDPAssociate = Cmd(0x03)

	RepSucceeded                     = Rep(0x00)
	RepServerFailure                 = Rep(0x01)
	RepConnectionNotAllowedByRuleset = Rep(0x02)
	RepNetworkUnreachable            = Rep(0x03)
	RepHostUnreachable               = Rep(0x04)
	RepConnectionRefused             = Rep(0x05)
	RepTTLExpired                    = Rep(0x06)
	RepCmdNotSupported               = Rep(0x07)
	RepATypNotSupported              = Rep(0x08)
)
