package http

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
)

// Reader defines what the handler needs from storage
type Reader interface {
	GetAlive(ctx context.Context) ([]*proxy.Proxy, error)
}

type Handler struct {
	reader Reader
	logger logs.Logger
}

func NewHandler(reader Reader, logger logs.Logger) *Handler {
	return &Handler{
		reader: reader,
		logger: logger,
	}
}

// ProxyResponse is the JSON representation of a proxy
type ProxyResponse struct {
	Address   string `json:"address"`
	Protocol  string `json:"protocol"`
	Anonymity string `json:"anonymity"`
	Latency   int64  `json:"latency_ms"`
	Source    string `json:"source"`
}

func toResponse(p *proxy.Proxy) ProxyResponse {
	return ProxyResponse{
		Address:   p.Address(),
		Protocol:  string(p.Protocol),
		Anonymity: string(p.Anonymity),
		Latency:   p.Latency.Milliseconds(),
		Source:    p.Source,
	}
}

// ProxyFilter holds query parameters for filtering
type ProxyFilter struct {
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

func parseFilters(r *http.Request) ProxyFilter {
	f := ProxyFilter{}
	q := r.URL.Query()

	f.Protocol = q.Get("protocol")
	f.Anonymity = q.Get("anonymity")

	if ms := q.Get("max_latency_ms"); ms != "" {
		if val, err := strconv.ParseInt(ms, 10, 64); err == nil {
			f.MaxLatency = time.Duration(val) * time.Millisecond
		}
	}

	return f
}

func (f ProxyFilter) matches(p *proxy.Proxy) bool {
	if f.Protocol != "" && string(p.Protocol) != f.Protocol {
		return false
	}
	if f.Anonymity != "" && string(p.Anonymity) != f.Anonymity {
		return false
	}
	if f.MaxLatency > 0 && p.Latency > f.MaxLatency {
		return false
	}
	return true
}

func filterProxies(proxies []*proxy.Proxy, f ProxyFilter) []*proxy.Proxy {
	result := make([]*proxy.Proxy, 0, len(proxies))
	for _, p := range proxies {
		if f.matches(p) {
			result = append(result, p)
		}
	}
	return result
}

// Health returns service health status
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetProxies returns alive proxies with optional filters
// Query params: protocol, anonymity, max_latency_ms
func (h *Handler) GetProxies(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.reader.GetAlive(r.Context())
	if err != nil {
		h.logger.Error("failed to get proxies", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filters := parseFilters(r)
	proxies = filterProxies(proxies, filters)

	response := make([]ProxyResponse, len(proxies))
	for i, p := range proxies {
		response[i] = toResponse(p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRandomProxy returns a random alive proxy matching filters
// Query params: protocol, anonymity, max_latency_ms
func (h *Handler) GetRandomProxy(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.reader.GetAlive(r.Context())
	if err != nil {
		h.logger.Error("failed to get proxies", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filters := parseFilters(r)
	proxies = filterProxies(proxies, filters)

	if len(proxies) == 0 {
		http.Error(w, "no proxies available", http.StatusNotFound)
		return
	}

	p := proxies[rand.Intn(len(proxies))]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(p))
}
