package dns

import (
	"context"
	"crypto/tls"
	"net"

	log "github.com/sirupsen/logrus"
)

type DoTResolverFactory struct {
	dialer    net.Dialer
	tlsConfig tls.Config
	addr      net.Addr
}

func NewDoTResolverFactory(dialer net.Dialer, serverName string, addr net.Addr, insecureSkipVerify bool) DoTResolverFactory {
	return DoTResolverFactory{
		dialer: dialer,
		tlsConfig: tls.Config{
			ServerName:         serverName,
			ClientSessionCache: tls.NewLRUClientSessionCache(32),
			InsecureSkipVerify: insecureSkipVerify,
		},
		addr: addr,
	}
}

func (factory *DoTResolverFactory) GetResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial:     factory.GetDialFunction(),
	}
}

func (factory *DoTResolverFactory) GetDialFunction() func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(context context.Context, _, address string) (net.Conn, error) {
		conn, err := factory.dialer.DialContext(context, "tcp", factory.addr.String())
		if err != nil {
			log.Error(err)
			if conn != nil {
				err = conn.Close()
				if err != nil {
					log.Warn(err)
				}
			}
			return nil, err
		}
		return tls.Client(conn, &factory.tlsConfig), nil
	}
}
