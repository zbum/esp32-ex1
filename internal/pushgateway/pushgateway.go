// Package pushgateway sends one metric sample to a Prometheus Pushgateway.
//
// The gateway stores the text exposition format at
//
//	POST http://<host>:<port>/metrics/job/<job>/instance/<instance>
//
// The POST is a fresh TCP connection with "Connection: close", not net/http.
// espradio's lneto stack keeps only a couple of TCP sockets, and net/http
// leaves them in TIME_WAIT long enough that a later dial returns
// "resource exhausted". Closing from the server side, after the response is
// drained, frees the client socket within the stack's closing timeout.
package pushgateway

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Endpoint builds the Pushgateway grouping path for one job and instance.
// baseURL is the gateway root, for example "http://192.168.31.102:9091".
func Endpoint(baseURL, job, instance string) (string, error) {
	if !strings.HasPrefix(baseURL, "http://") {
		return "", fmt.Errorf("pushgateway: base URL must start with http://")
	}
	if job == "" || instance == "" {
		return "", fmt.Errorf("pushgateway: job and instance are required")
	}
	base := strings.TrimRight(baseURL, "/")
	return base + "/metrics/job/" + url.PathEscape(job) + "/instance/" + url.PathEscape(instance), nil
}

// Sample is one exposition payload: the board is up, seconds since the
// push loop started, and how many pushes have succeeded including this one.
func Sample(uptimeSec, pushTotal uint64) string {
	return "# TYPE esp32_up gauge\n" +
		"esp32_up 1\n" +
		"# TYPE esp32_uptime_seconds gauge\n" +
		"esp32_uptime_seconds " + strconv.FormatUint(uptimeSec, 10) + "\n" +
		"# TYPE esp32_push_total gauge\n" +
		"esp32_push_total " + strconv.FormatUint(pushTotal, 10) + "\n"
}

// Pusher targets one grouping URL produced by Endpoint.
type Pusher struct {
	URL string

	addr string
	path string
}

// New parses url into host and path so Push does not reparse it.
// An unusable url yields a Pusher whose Push returns an error.
func New(rawURL string) *Pusher {
	p := &Pusher{URL: rawURL}
	if !strings.HasPrefix(rawURL, "http://") {
		return p
	}
	rest := strings.TrimPrefix(rawURL, "http://")
	slash := strings.Index(rest, "/")
	if slash < 0 {
		p.addr = rest
		p.path = "/"
	} else {
		p.addr = rest[:slash]
		p.path = rest[slash:]
	}
	return p
}

// Push dials the gateway, writes one HTTP/1.1 POST, reads the status line,
// and drains the body so the socket can close. A non-2xx status is an error.
func (p *Pusher) Push(body string) error {
	if p == nil || p.URL == "" || p.addr == "" || p.path == "" {
		return fmt.Errorf("pushgateway: URL not set")
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}

	conn, err := net.Dial("tcp", p.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	w := bufio.NewWriter(conn)
	fmt.Fprintf(w, "POST %s HTTP/1.1\r\n", p.path)
	fmt.Fprintf(w, "Host: %s\r\n", p.addr)
	fmt.Fprintf(w, "User-Agent: TinyGo\r\n")
	fmt.Fprintf(w, "Connection: close\r\n")
	fmt.Fprintf(w, "Content-Type: text/plain; version=0.0.4\r\n")
	fmt.Fprintf(w, "Content-Length: %d\r\n", len(body))
	fmt.Fprintf(w, "\r\n")
	if _, err := w.WriteString(body); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	r := bufio.NewReader(conn)
	status, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	// Drain until the server FINs. Leaving unread bytes holds the lneto
	// socket past this call.
	_, _ = io.Copy(io.Discard, r)

	status = strings.TrimSpace(status)
	if !strings.HasPrefix(status, "HTTP/1.1 2") && !strings.HasPrefix(status, "HTTP/1.0 2") {
		return fmt.Errorf("pushgateway: %s", status)
	}
	return nil
}
