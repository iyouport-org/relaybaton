package dns

import (
	"context"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

func LookupHTTPS(domain string) ([]dns.HTTPS, error) {
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}
	var resolver *net.Resolver
	if net.DefaultResolver != nil && net.DefaultResolver.Dial != nil {
		resolver = net.DefaultResolver
	} else {
		resolver = &net.Resolver{
			PreferGo:     false,
			StrictErrors: false,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial(network, address)
			},
		}
	}
	conn, err := resolver.Dial(context.Background(), "udp", "1.1.1.1:53")
	if err != nil {
		log.Error(err)
	}
	defer conn.Close()
	req := new(dns.Msg)
	rep := new(dns.Msg)
	req.SetQuestion(domain, dns.TypeHTTPS)
	wire, err := req.Pack()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	_, err = conn.Write(wire)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	repBytes := make([]byte, 2<<10)
	n, err := conn.Read(repBytes)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = rep.Unpack(repBytes[:n])
	if err != nil {
		log.Error(err)
		return nil, err
	}
	answers := make([]dns.HTTPS, len(rep.Answer))
	for i, answerRR := range rep.Answer {
		answer, ok := answerRR.(*dns.HTTPS)
		if !ok {
			log.Error("RR Type is not HTTPS")
			return nil, err
		}
		answers[i] = *answer
	}
	return answers, nil
}
