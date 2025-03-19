package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/miekg/dns"
)

var serverIp string

// Handle DNS queries
func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	// Create a new DNS response message
	m := new(dns.Msg)
	m.SetReply(r)

	// Check if the query type is A (Address) record
	if r.Opcode == dns.OpcodeQuery && len(r.Question) > 0 {
		for _, q := range r.Question {
			log.Printf("domain: %s type: %d", q.Name, q.Qtype)
			if q.Name == "api.thenewagetherapist.com." {
				switch q.Qtype {
				case dns.TypeA:
					rr := &dns.A{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    60,
						},
						A: net.ParseIP(serverIp),
					}
					m.Answer = append(m.Answer, rr)
					w.WriteMsg(m)
					log.Println("Found record, sent server IP")
					return
				case dns.TypeAAAA:
					if q.Name == "api.thenewagetherapist.com." {
						m.Answer = nil // No answers in the response
						m.Ns = nil     // No authoritative nameservers (if not applicable)
						m.Extra = nil  // No additional records

						// Set the response code to 'No error', but with no data
						m.Response = true
						m.Rcode = dns.RcodeNameError

						w.WriteMsg(m)
						log.Println("Requested for AAAA record, falling back to A record")
						return
					}
				}
			}
		}
	}
	// If not found locally, forward the query to an upstream DNS server
	log.Println("Record not found, forwarding to upstream dns")
	forwardToUpstreamDNS(w, r)
}

// Forward the DNS query to an upstream DNS server (e.g., Google's DNS)
func forwardToUpstreamDNS(w dns.ResponseWriter, r *dns.Msg) {
	// Define the upstream DNS server
	upstreamDNS := "1.1.1.1:53"

	// Create a new DNS client to forward the query
	c := new(dns.Client)

	// Forward the query to the upstream DNS server
	resp, _, err := c.Exchange(r, upstreamDNS)
	if err != nil {
		fmt.Printf("Failed to forward query: %v", err)
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	for _, q := range r.Question {
		if q.Qtype == dns.TypeAAAA && len(resp.Answer) > 0 {
			log.Printf("Response: %s", resp.Answer[0].String())
		}
	}

	// Return the response from the upstream DNS server
	w.WriteMsg(resp)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Server IP is needed!")
		return
	}
	serverIp = os.Args[1]
	// Set up DNS server
	server := &dns.Server{
		Addr: ":53", // DNS server listens on port 53
		Net:  "udp", // Listen for UDP requests
	}

	// Handle DNS queries using the handleRequest function
	dns.HandleFunc(".", handleRequest)

	// Start the DNS server
	log.Println("Starting DNS server on port 53...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalln("Failed to start server: ", err)
	}
}
