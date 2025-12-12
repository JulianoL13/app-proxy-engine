package proxy

import (
	"fmt"
	"net/url"
	"time"
)

type Protocol string

const (
	HTTP   Protocol = "http"
	HTTPS  Protocol = "https"
	SOCKS4 Protocol = "socks4"
	SOCKS5 Protocol = "socks5"
)

type AnonymityLevel string

const (
	Transparent AnonymityLevel = "transparent"
	Anonymous   AnonymityLevel = "anonymous"
	Elite       AnonymityLevel = "elite"
	Unknown     AnonymityLevel = "unknown"
)

type Proxy struct {
	IP            string
	Port          int
	Protocol      Protocol
	Anonymity     AnonymityLevel
	Source        string
	FirstSeenAt   time.Time
	LastCheckAt   time.Time
	Latency       time.Duration
	FailCount     int
	CooldownUntil time.Time
}

func NewProxy(ip string, port int, protocol Protocol, source string) *Proxy {
	return &Proxy{
		IP:          ip,
		Port:        port,
		Protocol:    protocol,
		Source:      source,
		FirstSeenAt: time.Now(),
		Anonymity:   Unknown,
	}
}

func (p *Proxy) Address() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}
func (p *Proxy) URL() *url.URL {
	return &url.URL{
		Scheme: string(p.Protocol),
		Host:   p.Address(),
	}
}

func (p *Proxy) IsReady() bool {
	return time.Now().After(p.CooldownUntil)
}

func (p *Proxy) MarkSuccess(latency time.Duration, anonymity AnonymityLevel) {
	p.FailCount = 0
	p.CooldownUntil = time.Time{}
	p.LastCheckAt = time.Now()
	p.Latency = latency
	p.Anonymity = anonymity
}

func (p *Proxy) MarkFailure() {
	p.FailCount++
	p.LastCheckAt = time.Now()

	matchFail := p.FailCount
	if matchFail < 1 {
		matchFail = 1
	}

	backoffMinutes := 5 * (1 << (matchFail - 1))

	duration := time.Duration(backoffMinutes) * time.Minute
	p.CooldownUntil = time.Now().Add(duration)
}
