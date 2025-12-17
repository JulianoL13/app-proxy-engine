package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeValidationError(w http.ResponseWriter, errs []FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ValidationError{Errors: errs})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

type GetProxiesInput struct {
	Cursor     float64
	Limit      int
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

type GetProxiesOutput struct {
	Proxies    []*proxy.Proxy
	NextCursor float64
	Total      int
}

type GetProxiesUseCase interface {
	Execute(ctx context.Context, input GetProxiesInput) (GetProxiesOutput, error)
}

type GetRandomProxyInput struct {
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

type GetRandomProxyUseCase interface {
	Execute(ctx context.Context, input GetRandomProxyInput) (*proxy.Proxy, error)
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

type PaginatedResponse struct {
	Data       []ProxyResponse `json:"data"`
	NextCursor *string         `json:"next_cursor,omitempty"`
	Limit      int             `json:"limit"`
	TotalCount int             `json:"total_count"`
}

const (
	defaultLimit = 25
	maxLimit     = 100
)

var (
	validProtocols   = []string{"http", "https", "socks4", "socks5"}
	validAnonymities = []string{"transparent", "anonymous", "elite"}
)

func isValidEnum(value string, validValues []string) bool {
	for _, v := range validValues {
		if value == v {
			return true
		}
	}
	return false
}

func encodeCursor(score float64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%f", score)))
}

func decodeCursor(cursor string) (float64, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(string(decoded), 64)
}

func parsePagination(r *http.Request) (cursor float64, limit int, errs []FieldError) {
	q := r.URL.Query()
	limit = defaultLimit

	if c := q.Get("cursor"); c != "" {
		val, err := decodeCursor(c)
		if err != nil {
			errs = append(errs, FieldError{Field: "cursor", Message: "invalid cursor format"})
		} else if val < 0 {
			errs = append(errs, FieldError{Field: "cursor", Message: "invalid cursor"})
		} else {
			cursor = val
		}
	}

	if l := q.Get("limit"); l != "" {
		val, err := strconv.Atoi(l)
		if err != nil {
			errs = append(errs, FieldError{Field: "limit", Message: "must be a valid integer"})
		} else if val <= 0 {
			errs = append(errs, FieldError{Field: "limit", Message: "must be positive"})
		} else {
			limit = val
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	return cursor, limit, errs
}

func parseFilters(r *http.Request) (protocol, anonymity string, maxLatency time.Duration, errs []FieldError) {
	q := r.URL.Query()

	if p := q.Get("protocol"); p != "" {
		if !isValidEnum(p, validProtocols) {
			errs = append(errs, FieldError{Field: "protocol", Message: "must be one of: http, https, socks4, socks5"})
		} else {
			protocol = p
		}
	}

	if a := q.Get("anonymity"); a != "" {
		if !isValidEnum(a, validAnonymities) {
			errs = append(errs, FieldError{Field: "anonymity", Message: "must be one of: transparent, anonymous, elite"})
		} else {
			anonymity = a
		}
	}

	if ms := q.Get("max_latency_ms"); ms != "" {
		val, err := strconv.ParseInt(ms, 10, 64)
		if err != nil {
			errs = append(errs, FieldError{Field: "max_latency_ms", Message: "must be a valid integer"})
		} else if val <= 0 {
			errs = append(errs, FieldError{Field: "max_latency_ms", Message: "must be positive"})
		} else {
			maxLatency = time.Duration(val) * time.Millisecond
		}
	}

	return protocol, anonymity, maxLatency, errs
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetProxies(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger(r)

	cursor, limit, paginationErrs := parsePagination(r)
	protocol, anonymity, maxLatency, filterErrs := parseFilters(r)

	allErrs := append(paginationErrs, filterErrs...)
	if len(allErrs) > 0 {
		writeValidationError(w, allErrs)
		return
	}

	output, err := h.getProxies.Execute(r.Context(), GetProxiesInput{
		Cursor:     cursor,
		Limit:      limit,
		Protocol:   protocol,
		Anonymity:  anonymity,
		MaxLatency: maxLatency,
	})
	if err != nil {
		logger.Error("failed to get proxies", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	data := make([]ProxyResponse, len(output.Proxies))
	for i, p := range output.Proxies {
		data[i] = toResponse(p)
	}

	response := PaginatedResponse{
		Data:       data,
		Limit:      limit,
		TotalCount: output.Total,
	}

	if output.NextCursor > 0 {
		encoded := encodeCursor(output.NextCursor)
		response.NextCursor = &encoded
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetRandomProxy(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger(r)

	protocol, anonymity, maxLatency, errs := parseFilters(r)
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	p, err := h.getRandomProxy.Execute(r.Context(), GetRandomProxyInput{
		Protocol:   protocol,
		Anonymity:  anonymity,
		MaxLatency: maxLatency,
	})
	if err != nil {
		if errors.Is(err, proxy.ErrNoProxiesAvailable) {
			writeError(w, http.StatusNotFound, "no proxies available")
			return
		}
		logger.Error("failed to get random proxy", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(p))
}
