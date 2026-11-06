package httpapi_test

import (
	"net/http"
	"runtime"
	"testing"
)

// TestServerInfo reports real machine values to a signed-in user and refuses
// anonymous calls; the platform flag distinguishes "0" from "unreadable".
func TestServerInfo(t *testing.T) {
	ts := newTestServer(t, "open")

	if resp, _ := post(t, ts, "/api/server-info", map[string]any{}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want 401", resp.StatusCode)
	}

	_, token := createUser(t, ts, "alice")
	resp, out := postWithToken(t, ts, "/api/server-info", token, map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body %v", resp.StatusCode, out)
	}
	cpu, _ := out["cpu"].(map[string]any)
	mem, _ := out["mem"].(map[string]any)
	fs, _ := out["fs"].(map[string]any)
	load, _ := out["load"].(map[string]any)
	if cpu["cores"].(float64) != float64(runtime.NumCPU()) {
		t.Errorf("cores = %v, want %d", cpu["cores"], runtime.NumCPU())
	}
	usage, _ := cpu["usage"].(float64)
	if usage < 0 || usage > 100 {
		t.Errorf("usage = %v, want 0–100", usage)
	}
	if runtime.GOOS == "linux" {
		if mem["known"] != true || mem["total"].(float64) <= 0 {
			t.Errorf("mem = %v, want a known positive total", mem)
		}
		if fs["known"] != true || fs["total"].(float64) <= 0 {
			t.Errorf("fs = %v, want a known positive total", fs)
		}
		if load["known"] != true {
			t.Errorf("load = %v, want known on linux", load)
		}
		if out["machine"] == "" || out["machine"] == "Not specified" {
			t.Errorf("machine = %v, want the hostname", out["machine"])
		}
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/server-info", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	got, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer got.Body.Close()
	if got.StatusCode != http.StatusOK {
		t.Errorf("GET status = %d, want 200", got.StatusCode)
	}
}
