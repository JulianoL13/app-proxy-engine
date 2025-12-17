package events

type ProxyDiscoveredEvent struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Source   string `json:"source"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
