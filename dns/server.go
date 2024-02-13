package dns

import (
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mrhaoxx/OpenNG/log"

	"github.com/dlclark/regexp2"
	"github.com/miekg/dns"
)

type record struct {
	rtype  uint16
	rvalue string
	name   *regexp2.Regexp
}
type filter struct {
	name      *regexp2.Regexp
	allowance bool
}

type server struct {
	records []*record
	filters []*filter
	domain  string

	count uint64
}

func joinNames(questions []dns.Question) string {
	var names []string
	for _, q := range questions {
		names = append(names, q.Name)
	}
	return strings.Join(names, " ")
}

func joinTypes(questions []dns.Question) string {
	var types []string
	for _, q := range questions {
		types = append(types, dns.TypeToString[q.Qtype])
	}
	return strings.Join(types, " ")
}

func (s *server) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg).SetReply(req)

	id := atomic.AddUint64(&s.count, 1)
	startTime := time.Now()
	defer func() {
		log.Println("d"+strconv.FormatUint(id, 10), w.RemoteAddr().String(), time.Since(startTime).Round(1*time.Microsecond), RcodeTypeMap[m.Rcode], joinTypes(req.Question), joinNames(req.Question))
	}()

	for _, q := range req.Question {
		for _, r := range s.filters {
			if ok, _ := r.name.MatchString(q.Name); ok {
				if r.allowance {
					break
				} else {
					m.Rcode = dns.RcodeRefused
					goto _end
				}
			}
		}
	}

	for _, q := range req.Question {
		for _, r := range s.records {
			if q.Qtype == r.rtype {
				if ok, _ := r.name.MatchString(q.Name); ok {
					var ret dns.RR
					switch r.rtype {
					case dns.TypeA:
						ret = &dns.A{
							Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
							A:   net.ParseIP(r.rvalue)}
					case dns.TypePTR:
						ret = &dns.PTR{
							Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 60},
							Ptr: r.rvalue}
					default:
						m.Rcode = dns.RcodeNotImplemented
						goto _end
					}
					m.Answer = append(m.Answer, ret)
				}
			}
		}
	}
	if len(m.Answer) == 0 {
		m.Rcode = dns.RcodeNameError
	}
_end:
	w.WriteMsg(m)
}

func (s *server) Listen(address string) error {
	server := &dns.Server{Addr: address, Net: "udp"}
	server.Handler = s
	return server.ListenAndServe()
}

func (s *server) AddFilter(name *regexp2.Regexp, allowance bool) error {
	s.filters = append(s.filters, &filter{name: name, allowance: allowance})
	return nil
}
func (s *server) AddRecord(name *regexp2.Regexp, rtype uint16, rvalue string) error {
	s.records = append(s.records, &record{name: name, rtype: rtype, rvalue: rvalue})
	return nil
}

func (s *server) AddRecordWithIP(name string, ip string) error {
	real_subdomain := name + "." + s.domain + "."
	real_ptr := reverseIP(ip) + ".in-addr.arpa." + s.domain + "."

	s.AddRecord(regexp2.MustCompile(Dnsname2Regexp(real_subdomain), 0), dns.TypeA, ip)
	s.AddRecord(regexp2.MustCompile(Dnsname2Regexp(real_ptr), 0), dns.TypePTR, real_subdomain)

	return nil
}

func NewServer(domain string) *server {
	return &server{
		records: []*record{},
		filters: []*filter{},
		domain:  domain,
	}
}

func reverseIP(ipAddr string) string {
	segments := strings.Split(ipAddr, ".")

	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}

	return strings.Join(segments, ".")
}
