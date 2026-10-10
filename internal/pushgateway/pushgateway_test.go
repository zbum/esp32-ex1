package pushgateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEndpoint(t *testing.T) {
	got, err := Endpoint("http://192.168.31.102:9091/", "esp32-ex1", "esp32")
	if err != nil {
		t.Fatal(err)
	}
	want := "http://192.168.31.102:9091/metrics/job/esp32-ex1/instance/esp32"
	if got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}

	if _, err := Endpoint("https://example", "job", "inst"); err == nil {
		t.Fatal("https base was accepted")
	}
	if _, err := Endpoint("http://example", "", "inst"); err == nil {
		t.Fatal("empty job was accepted")
	}
}

func TestSample(t *testing.T) {
	got := Sample(12, 3)
	for _, line := range []string{
		"# TYPE esp32_up gauge",
		"esp32_up 1",
		"esp32_uptime_seconds 12",
		"esp32_push_total 3",
	} {
		if !strings.Contains(got, line+"\n") {
			t.Fatalf("sample missing %q:\n%s", line, got)
		}
	}
}

func TestPush(t *testing.T) {
	const body = "esp32_up 1\n"
	var gotMethod, gotPath, gotHost, gotConn, gotType, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHost = r.Host
		gotConn = r.Header.Get("Connection")
		gotType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := New(srv.URL + "/metrics/job/esp32-ex1/instance/esp32")
	if err := p.Push(body); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s", gotMethod)
	}
	if gotPath != "/metrics/job/esp32-ex1/instance/esp32" {
		t.Fatalf("path = %s", gotPath)
	}
	if !strings.Contains(gotHost, ":") {
		t.Fatalf("host header missing port: %q", gotHost)
	}
	if !strings.EqualFold(gotConn, "close") {
		t.Fatalf("connection = %q", gotConn)
	}
	if !strings.Contains(gotType, "text/plain") {
		t.Fatalf("content-type = %q", gotType)
	}
	if gotBody != body {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestPushRejectsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	err := New(srv.URL + "/metrics/job/j/instance/i").Push("esp32_up 1\n")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v", err)
	}
}

func TestPushUnsetURL(t *testing.T) {
	if err := New("").Push("x\n"); err == nil {
		t.Fatal("empty URL was accepted")
	}
}
