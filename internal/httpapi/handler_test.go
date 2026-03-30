package httpapi

import (
	"antibf/internal/model"
	"antibf/internal/service"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type repoStub struct {
	whitelist map[string]struct{}
	blacklist map[string]struct{}
}

func newRepoStub() *repoStub {
	return &repoStub{
		whitelist: make(map[string]struct{}),
		blacklist: make(map[string]struct{}),
	}
}

func (r *repoStub) Add(_ context.Context, listType model.NetworkListType, cidr string) error {
	switch listType {
	case model.ListTypeWhitelist:
		r.whitelist[cidr] = struct{}{}
	case model.ListTypeBlacklist:
		r.blacklist[cidr] = struct{}{}
	}
	return nil
}

func (r *repoStub) Remove(_ context.Context, listType model.NetworkListType, cidr string) error {
	switch listType {
	case model.ListTypeWhitelist:
		delete(r.whitelist, cidr)
	case model.ListTypeBlacklist:
		delete(r.blacklist, cidr)
	}
	return nil
}

func (r *repoStub) ContainsIP(_ context.Context, listType model.NetworkListType, ip net.IP) (bool, error) {
	var source map[string]struct{}

	switch listType {
	case model.ListTypeWhitelist:
		source = r.whitelist
	case model.ListTypeBlacklist:
		source = r.blacklist
	default:
		return false, nil
	}

	for cidr := range source {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
}

type limiterStub struct {
	allow bool
}

func (l *limiterStub) Allow(_ context.Context, _, _, _ string) (bool, error) {
	return l.allow, nil
}

func (l *limiterStub) Reset(_ context.Context, _, _ string) error {
	return nil
}

func TestHandler_IntegrationFlow(t *testing.T) {
	repo := newRepoStub()
	limiter := &limiterStub{allow: true}
	svc := service.New(repo, limiter)
	h := NewHandler(svc)
	ts := httptest.NewServer(h.Router())
	defer ts.Close()

	t.Run("auth check allowed", func(t *testing.T) {
		result := doCheck(t, ts.URL, model.AuthAttempt{
			Login:    "alice",
			Password: "secret",
			IP:       "192.168.1.10",
		})
		if !result.OK {
			t.Fatal("request must be allowed")
		}
	})

	t.Run("reset bucket", func(t *testing.T) {
		reqBody := marshal(t, model.ResetRequest{Login: "alice", IP: "192.168.1.10"})
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			ts.URL+"/api/v1/buckets/reset",
			bytes.NewReader(reqBody),
		)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("reset bucket: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	})

	t.Run("whitelist allows", func(t *testing.T) {
		addNetwork(t, ts.URL+"/api/v1/whitelist", "192.168.1.0/24", http.StatusCreated)

		result := doCheck(t, ts.URL, model.AuthAttempt{
			Login:    "bob",
			Password: "secret",
			IP:       "192.168.1.55",
		})
		if !result.OK {
			t.Fatal("whitelisted IP must be allowed")
		}
	})

	t.Run("blacklist blocks", func(t *testing.T) {
		addNetwork(t, ts.URL+"/api/v1/blacklist", "10.0.0.0/8", http.StatusCreated)

		result := doCheck(t, ts.URL, model.AuthAttempt{
			Login:    "charlie",
			Password: "secret",
			IP:       "10.1.2.3",
		})
		if result.OK {
			t.Fatal("blacklisted IP must be blocked")
		}
	})

	t.Run("remove blacklist", func(t *testing.T) {
		deleteURL := ts.URL + "/api/v1/blacklist/" + url.PathEscape("10.0.0.0/8")

		httpReq, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodDelete,
			deleteURL,
			nil,
		)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			t.Fatalf("post auth check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	})

	t.Run("health check", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("health check: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	})

	t.Run("remove whitelist", func(t *testing.T) {
		deleteURL := ts.URL + "/api/v1/whitelist/" + url.PathEscape("192.168.1.0/24")

		httpReq, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodDelete,
			deleteURL,
			nil,
		)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			t.Fatalf("delete request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	})

	t.Run("bad method", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/auth/check")
		if err != nil {
			t.Fatalf("get request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected method not allowed, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/check", bytes.NewReader([]byte("invalid")))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("post request: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", resp.StatusCode)
		}
	})

	t.Run("bad method reset", func(t *testing.T) {
		resp, _ := http.Get(ts.URL + "/api/v1/buckets/reset")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid json reset", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/buckets/reset", bytes.NewReader([]byte("invalid")))
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("service error reset", func(t *testing.T) {
		reqBody := marshal(t, model.ResetRequest{Login: "", IP: "1.1.1.1"})
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/buckets/reset", bytes.NewReader(reqBody))
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for service error, got %d", resp.StatusCode)
		}
	})

	t.Run("bad method add network", func(t *testing.T) {
		resp, _ := http.Get(ts.URL + "/api/v1/whitelist")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid json add network", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/whitelist", bytes.NewReader([]byte("invalid")))
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("service error add network", func(t *testing.T) {
		reqBody := marshal(t, map[string]string{"cidr": "invalid"})
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/whitelist", bytes.NewReader(reqBody))
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for service error, got %d", resp.StatusCode)
		}
	})

	t.Run("bad method remove network", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/whitelist/1.2.3.4", nil)
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", resp.StatusCode)
		}
	})

	t.Run("service error remove network", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/whitelist/invalid", nil)
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for service error, got %d", resp.StatusCode)
		}
	})
}

func doCheck(t *testing.T, baseURL string, attempt model.AuthAttempt) model.AuthResult {
	t.Helper()

	payload := marshal(t, attempt)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		baseURL+"/api/v1/auth/check",
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reset bucket: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var result model.AuthResult
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode auth result: %v", err)
	}

	return result
}

func addNetwork(t *testing.T, endpoint, cidr string, expectedStatus int) {
	t.Helper()

	payload := marshal(t, map[string]string{"cidr": cidr})

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("reset bucket: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func marshal(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	return data
}
