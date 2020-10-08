package relaybaton_mobile

import (
	"context"
	"fmt"
	"net"

	"github.com/eycorsican/go-tun2socks/core"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

type udpHandler struct {
	dnsServerMux *dns.ServeMux
}

func NewUDPHandler() core.UDPConnHandler {
	return &udpHandler{
		dnsServerMux: dns.NewServeMux(),
	}
}

func (h *udpHandler) Connect(conn core.UDPConn, target *net.UDPAddr) error {
	return nil
}

func (h *udpHandler) ReceiveTo(conn core.UDPConn, data []byte, addr *net.UDPAddr) error {
	if addr.Port == 53 {
		req := new(dns.Msg)
		err := req.Unpack(data)
		if err != nil {
			log.Error(err)
			return err
		}
		log.Debug(req.String())
		resp := new(dns.Msg)
		resp.SetReply(req)
		for _, q := range req.Question {
			switch q.Qtype {
			case dns.TypeA:
				ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", q.Name)
				if err != nil {
					log.Error(err)
					return err
				}
				for _, ip := range ips {
					rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip.To4().String()))
					if err != nil {
						log.Error(err)
						return err
					}
					resp.Answer = append(req.Answer, rr)

				}
				respBytes, err := resp.Pack()
				if err != nil {
					log.Error(err)
					return err
				}
				_, err = conn.WriteFrom(respBytes, &net.UDPAddr{
					IP:   net.IPv4(172, 19, 0, 2).To4(),
					Port: 53,
					Zone: "",
				})
				if err != nil {
					log.Error(err)
					return err
				}
			case dns.TypeAAAA:
				ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip6", q.Name)
				if err != nil {
					log.Error(err)
					return err
				}
				for _, ip := range ips {
					rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, ip.To4().String()))
					if err != nil {
						log.Error(err)
						return err
					}
					resp.Answer = append(req.Answer, rr)

				}
				respBytes, err := resp.Pack()
				if err != nil {
					log.Error(err)
					return err
				}
				_, err = conn.WriteFrom(respBytes, &net.UDPAddr{
					IP:   net.IPv4(172, 19, 0, 2).To4(),
					Port: 53,
					Zone: "",
				})
				if err != nil {
					log.Error(err)
					return err
				}
			case dns.TypeTXT:
				txts, err := net.DefaultResolver.LookupTXT(context.Background(), q.Name)
				if err != nil {
					log.Error(err)
					return err
				}
				for _, txt := range txts {
					rr, err := dns.NewRR(fmt.Sprintf("%s TXT %s", q.Name, txt))
					if err != nil {
						log.Error(err)
						return err
					}
					resp.Answer = append(req.Answer, rr)

				}
				respBytes, err := resp.Pack()
				if err != nil {
					log.Error(err)
					return err
				}
				_, err = conn.WriteFrom(respBytes, &net.UDPAddr{
					IP:   net.IPv4(172, 19, 0, 2).To4(),
					Port: 53,
					Zone: "",
				})
				if err != nil {
					log.Error(err)
					return err
				}
			case dns.TypeCNAME:
				cname, err := net.DefaultResolver.LookupCNAME(context.Background(), q.Name)
				if err != nil {
					log.Error(err)
					return err
				}

				rr, err := dns.NewRR(fmt.Sprintf("%s CNAME %s", q.Name, cname))
				if err != nil {
					log.Error(err)
					return err
				}
				resp.Answer = append(req.Answer, rr)

				respBytes, err := resp.Pack()
				if err != nil {
					log.Error(err)
					return err
				}
				_, err = conn.WriteFrom(respBytes, &net.UDPAddr{
					IP:   net.IPv4(172, 19, 0, 2).To4(),
					Port: 53,
					Zone: "",
				})
				if err != nil {
					log.Error(err)
					return err
				}
			case dns.TypeMX:
				mxs, err := net.DefaultResolver.LookupMX(context.Background(), q.Name)
				if err != nil {
					log.Error(err)
					return err
				}

				for _, mx := range mxs {
					rr, err := dns.NewRR(fmt.Sprintf("%s MX %s", q.Name, mx.Host))
					if err != nil {
						log.Error(err)
						return err
					}
					resp.Answer = append(req.Answer, rr)

				}
				respBytes, err := resp.Pack()
				if err != nil {
					log.Error(err)
					return err
				}
				_, err = conn.WriteFrom(respBytes, &net.UDPAddr{
					IP:   net.IPv4(172, 19, 0, 2).To4(),
					Port: 53,
					Zone: "",
				})
				if err != nil {
					log.Error(err)
					return err
				}
			case dns.TypeNS:
				nss, err := net.DefaultResolver.LookupNS(context.Background(), q.Name)
				if err != nil {
					log.Error(err)
					return err
				}

				for _, ns := range nss {
					rr, err := dns.NewRR(fmt.Sprintf("%s NS %s", q.Name, ns.Host))
					if err != nil {
						log.Error(err)
						return err
					}
					resp.Answer = append(req.Answer, rr)

				}
				respBytes, err := resp.Pack()
				if err != nil {
					log.Error(err)
					return err
				}
				_, err = conn.WriteFrom(respBytes, &net.UDPAddr{
					IP:   net.IPv4(172, 19, 0, 2).To4(),
					Port: 53,
					Zone: "",
				})
				if err != nil {
					log.Error(err)
					return err
				}
			}
		}
	}
	return nil
}
