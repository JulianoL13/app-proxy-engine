package scraper

type Source struct {
	Name string
	URL  string
	Type string
}

func PublicSources() []Source {
	return []Source{
		// 1. TheSpeedX
		{Name: "TheSpeedX-HTTP", URL: "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt", Type: "http"},
		{Name: "TheSpeedX-SOCKS5", URL: "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks5.txt", Type: "socks5"},

		// 2. Monosans
		{Name: "Monosans-HTTP", URL: "https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt", Type: "http"},
		{Name: "Monosans-SOCKS5", URL: "https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", Type: "socks5"},

		// 3. ShiftyTR
		{Name: "ShiftyTR-HTTP", URL: "https://raw.githubusercontent.com/ShiftyTR/Proxy-List/master/http.txt", Type: "http"},
		{Name: "ShiftyTR-HTTPS", URL: "https://raw.githubusercontent.com/ShiftyTR/Proxy-List/master/https.txt", Type: "https"},
		{Name: "ShiftyTR-SOCKS5", URL: "https://raw.githubusercontent.com/ShiftyTR/Proxy-List/master/socks5.txt", Type: "socks5"},

		// 4. Hookzof
		{Name: "Hookzof-SOCKS5", URL: "https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", Type: "socks5"},
	}
}
