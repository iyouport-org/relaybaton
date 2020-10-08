package dns

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

type DoHResolverFactory struct {
	dialer       net.Dialer
	port         uint16
	addr         net.Addr
	url          url.URL
	strictErrors bool
	server       dns.Server
	client       http.Client
}

func NewDoHResolverFactory(dialer net.Dialer, port uint16, serverName string, addr net.Addr, strictErrors bool) (*DoHResolverFactory, error) {
	u, err := url.ParseRequestURI("https://" + serverName + "/dns-query")
	if err != nil {
		return nil, err
	}
	factory := &DoHResolverFactory{
		dialer:       dialer,
		port:         port,
		url:          *u,
		addr:         addr,
		strictErrors: strictErrors,
		server: dns.Server{
			Addr: fmt.Sprintf("127.0.0.1:%d", port),
			Net:  "tcp",
		},
		client: http.Client{
			Transport: &http.Transport{
				Proxy: nil,
				DialTLSContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return tls.Dial("tcp", addr.String()+":443", &tls.Config{
						ServerName: u.Hostname(),
					})
				},
			},
			Timeout: http.DefaultClient.Timeout,
		},
	}

	go func() {
		err = factory.server.ListenAndServe()
		if err != nil {
			log.Error()
		}
	}()
	dns.HandleFunc(".", factory.handleRequest)
	return factory, nil
}

func (factory *DoHResolverFactory) GetResolver() *net.Resolver {
	resolver := &net.Resolver{
		StrictErrors: factory.strictErrors,
		Dial:         factory.getDialFunction(),
	}
	return resolver
}

func (factory *DoHResolverFactory) getDialFunction() func(ctx context.Context, network string, address string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		conn, err := factory.dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", factory.port))
		if err != nil {
			log.WithFields(log.Fields{
				"network": network,
				"address": address,
			}).Error(err)
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

func (factory *DoHResolverFactory) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
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
		log.WithFields(log.Fields{
			"url":  factory.url.String(),
			"wire": str,
		}).Error(err)
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
		fields := log.Fields{}
		for k, v := range resp.Header {
			fields[k] = v
		}
		log.WithFields(fields).Error(err)
		return
	}
	err = m.Unpack(body)
	if err != nil {
		log.WithField("body", body).Error(err)
		return
	}
	//log.Debug(m.String())	//test
	err = w.WriteMsg(m)
	if err != nil {
		log.Error(err)
		err = w.Close()
		if err != nil {
			log.Warn(err)
		}
		return
	}
}
