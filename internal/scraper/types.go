package scraper

type ScrapeOutput struct {
	ip       string
	port     int
	protocol string
	source   string
}

func NewScrapeOutput(ip string, port int, protocol, source string) *ScrapeOutput {
	return &ScrapeOutput{
		ip:       ip,
		port:     port,
		protocol: protocol,
		source:   source,
	}
}

func (s *ScrapeOutput) IP() string       { return s.ip }
func (s *ScrapeOutput) Port() int        { return s.port }
func (s *ScrapeOutput) Protocol() string { return s.protocol }
func (s *ScrapeOutput) Source() string   { return s.source }
