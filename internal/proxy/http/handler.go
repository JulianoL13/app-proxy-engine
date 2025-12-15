package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type GetProxiesInput struct {
	Cursor    float64
	Limit     int
	Protocol  string
	Anonymity string
}

type GetProxiesOutput struct {
	Proxies    []*proxy.Proxy
	NextCursor float64
	Total      int
}

type GetProxiesUseCase interface {
	Execute(ctx context.Context, input GetProxiesInput) (GetProxiesOutput, error)
}

type GetRandomProxyUseCase interface {
	Execute(ctx context.Context) (*proxy.Proxy, error)
}

type Handler struct {
	getProxies     GetProxiesUseCase
	getRandomProxy GetRandomProxyUseCase
	logger         Logger
}

func NewHandler(
	getProxies GetProxiesUseCase,
	getRandomProxy GetRandomProxyUseCase,
	logger Logger,
) *Handler {
	return &Handler{
		getProxies:     getProxies,
		getRandomProxy: getRandomProxy,
		logger:         logger,
	}
}

func (h *Handler) getLogger(r *http.Request) Logger {
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

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetProxies(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger(r)

	cursor, limit := parsePagination(r)
	filters := parseFilters(r)

	output, err := h.getProxies.Execute(r.Context(), GetProxiesInput{
		Cursor:    cursor,
		Limit:     limit,
		Protocol:  filters.Protocol,
		Anonymity: filters.Anonymity,
	})
	if err != nil {
		logger.Error("failed to get proxies", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Filter by Latency (in memory still, as Redis doesn't index latency yet)
	// Protocol and Anonymity are handled by UseCase -> Repository
	proxies := output.Proxies
	if filters.MaxLatency > 0 {
		filtered := make([]*proxy.Proxy, 0, len(proxies))
		for _, p := range proxies {
			if p.Latency <= filters.MaxLatency {
				filtered = append(filtered, p)
			}
		}
		proxies = filtered
	}

	data := make([]ProxyResponse, len(proxies))
	for i, p := range proxies {
		data[i] = toResponse(p)
	}

	response := PaginatedResponse{
		Data:       data,
		Limit:      limit,
		TotalCount: output.Total,
	}

	if output.NextCursor > 0 {
		response.NextCursor = &output.NextCursor
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetRandomProxy(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger(r)

	p, err := h.getRandomProxy.Execute(r.Context())
	if err != nil {
		if errors.Is(err, proxy.ErrNoProxiesAvailable) {
			http.Error(w, "no proxies available", http.StatusNotFound)
			return
		}
		logger.Error("failed to get random proxy", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filters := parseFilters(r)
	if !filters.matches(p) {
		http.Error(w, "no proxies available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(p))
}
