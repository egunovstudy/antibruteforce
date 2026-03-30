package httpapi

import (
	"antibf/internal/model"
	"antibf/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Handler struct {
	svc *service.AntiBruteforceService
}

func NewHandler(svc *service.AntiBruteforceService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/v1/auth/check", h.check)
	mux.HandleFunc("/api/v1/buckets/reset", h.reset)
	mux.HandleFunc("/api/v1/whitelist", h.addWhitelist)
	mux.HandleFunc("/api/v1/blacklist", h.addBlacklist)
	mux.HandleFunc("/api/v1/whitelist/", h.removeWhitelist)
	mux.HandleFunc("/api/v1/blacklist/", h.removeBlacklist)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	var req model.AuthAttempt
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok, err := h.svc.Check(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, model.AuthResult{OK: ok})
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	var req model.ResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.Reset(r.Context(), req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addWhitelist(w http.ResponseWriter, r *http.Request) {
	h.addNetwork(w, r, model.ListTypeWhitelist)
}

func (h *Handler) addBlacklist(w http.ResponseWriter, r *http.Request) {
	h.addNetwork(w, r, model.ListTypeBlacklist)
}

func (h *Handler) removeWhitelist(w http.ResponseWriter, r *http.Request) {
	h.removeNetwork(w, r, model.ListTypeWhitelist, "/api/v1/whitelist/")
}

func (h *Handler) removeBlacklist(w http.ResponseWriter, r *http.Request) {
	h.removeNetwork(w, r, model.ListTypeBlacklist, "/api/v1/blacklist/")
}

type networkRequest struct {
	CIDR string `json:"cidr"`
}

func (h *Handler) addNetwork(w http.ResponseWriter, r *http.Request, listType model.NetworkListType) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	var req networkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.AddNetwork(r.Context(), listType, req.CIDR); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) removeNetwork(w http.ResponseWriter, r *http.Request, listType model.NetworkListType, prefix string) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	cidr := strings.TrimPrefix(r.URL.Path, prefix)
	decoded, err := url.PathUnescape(strings.TrimSpace(cidr))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err = h.svc.RemoveNetwork(r.Context(), listType, decoded); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
