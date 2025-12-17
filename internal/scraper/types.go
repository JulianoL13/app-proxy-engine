package scraper

type ScrapeOutput struct {
	ip       string
	port     int
	protocol string
	source   string
	username string
	password string
}

func NewScrapeOutput(ip string, port int, protocol, source string) *ScrapeOutput {
	return &ScrapeOutput{
		ip:       ip,
		port:     port,
		protocol: protocol,
		source:   source,
	}
}

func NewScrapeOutputWithAuth(ip string, port int, protocol, source, username, password string) *ScrapeOutput {
	return &ScrapeOutput{
		ip:       ip,
		port:     port,
		protocol: protocol,
		source:   source,
		username: username,
		password: password,
	}
}

func (s *ScrapeOutput) IP() string       { return s.ip }
func (s *ScrapeOutput) Port() int        { return s.port }
func (s *ScrapeOutput) Protocol() string { return s.protocol }
func (s *ScrapeOutput) Source() string   { return s.source }
func (s *ScrapeOutput) Username() string { return s.username }
func (s *ScrapeOutput) Password() string { return s.password }
func (s *ScrapeOutput) HasAuth() bool    { return s.username != "" && s.password != "" }
