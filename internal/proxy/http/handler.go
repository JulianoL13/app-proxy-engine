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

type Reader interface {
	GetAlive(ctx context.Context, cursor float64, limit int) ([]*proxy.Proxy, float64, int, error)
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

func (h *Handler) getLogger(r *http.Request) logs.Logger {
	if l := LoggerFromContext(r.Context()); l != nil {
		return l
	}
	return h.logger
}

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

type ProxyFilter struct {
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

type PaginatedResponse struct {
	Data       []ProxyResponse `json:"data"`
	NextCursor *float64        `json:"next_cursor,omitempty"`
	Limit      int             `json:"limit"`
	TotalCount int             `json:"total_count"`
}

const defaultLimit = 50

func parsePagination(r *http.Request) (cursor float64, limit int) {
	q := r.URL.Query()

	limit = defaultLimit

	if c := q.Get("cursor"); c != "" {
		if val, err := strconv.ParseFloat(c, 64); err == nil && val >= 0 {
			cursor = val
		}
	}

	if l := q.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	return cursor, limit
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

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetProxies(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger(r)

	cursor, limit := parsePagination(r)

	proxies, nextCursor, total, err := h.reader.GetAlive(r.Context(), cursor, limit)
	if err != nil {
		logger.Error("failed to get proxies", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filters := parseFilters(r)
	proxies = filterProxies(proxies, filters)

	data := make([]ProxyResponse, len(proxies))
	for i, p := range proxies {
		data[i] = toResponse(p)
	}

	response := PaginatedResponse{
		Data:       data,
		Limit:      limit,
		TotalCount: total,
	}

	// Only include next_cursor if there's more data
	if nextCursor > 0 {
		response.NextCursor = &nextCursor
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetRandomProxy(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger(r)

	// Get all proxies (no pagination for random selection)
	proxies, _, _, err := h.reader.GetAlive(r.Context(), 0, 0)
	if err != nil {
		logger.Error("failed to get proxies", "error", err)
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
