package dns

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

type DoHResolverFactory struct {
	dialer       net.Dialer
	port         uint16
	ip           net.IP
	url          url.URL
	strictErrors bool
	server       dns.Server
	client       http.Client
}

func NewDoHResolverFactory(dialer net.Dialer, port uint16, serverName string, ipStr string, strictErrors bool) (*DoHResolverFactory, error) {
	u, err := url.ParseRequestURI("https://" + serverName + "/dns-query")
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		err = errors.New("IP parse error")
		log.Error(err)
		return nil, err
	}
	factory := &DoHResolverFactory{
		dialer:       dialer,
		port:         port,
		url:          *u,
		ip:           ip,
		strictErrors: strictErrors,
		server: dns.Server{
			Addr: fmt.Sprintf(":%d", port),
			Net:  "tcp",
		},
		client: http.Client{
			Transport: &http.Transport{
				Proxy: nil,
				DialTLS: func(network, addr string) (net.Conn, error) {
					return tls.Dial("tcp", ip.String()+":443", &tls.Config{
						ServerName: u.Hostname(),
					})
				},
			},
			Timeout: http.DefaultClient.Timeout,
		},
	}

	go factory.server.ListenAndServe()
	dns.HandleFunc(".", factory.handleRequest)
	return factory, nil
}

func (factory DoHResolverFactory) GetResolver() *net.Resolver {
	resolver := &net.Resolver{
		StrictErrors: factory.strictErrors,
		Dial:         factory.getDialFunction(),
	}
	return resolver
}

func (factory DoHResolverFactory) getDialFunction() func(ctx context.Context, network string, address string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		conn, err := factory.dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", factory.port))
		if err != nil {
			log.Error(err)
			if conn != nil {
				err = conn.Close()
				if err != nil {
					log.Error(err)
				}
			}
			return nil, err
		}
		return conn, nil
	}
}

func (factory DoHResolverFactory) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	wire, err := r.Pack()
	if err != nil {
		log.Error(err)
		return
	}
	str := base64.RawURLEncoding.EncodeToString(wire)
	req, err := http.NewRequest(http.MethodGet, factory.url.String()+"?dns="+str, nil)
	if err != nil {
		log.Error(err)
		return
	}
	req.Header.Add("content-type", "application/dns-message")
	req.Header.Add("accept", "application/dns-message")
	resp, err := factory.client.Do(req)
	if err != nil {
		log.Error(err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return
	}
	err = m.Unpack(body)
	if err != nil {
		log.Error(err)
		return
	}
	err = w.WriteMsg(m)
	if err != nil {
		log.Error(err)
		return
	}
}
