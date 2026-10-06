package atrust

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type tcpOnlyTestDialer struct {
	excluded bool
}

func (d *tcpOnlyTestDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

func (d *tcpOnlyTestDialer) ExcludeIP(net.IP) { d.excluded = true }

func TestTCPOnlySetupDoesNotRequireVIPOrL3Service(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/manifest" {
			t.Errorf("unexpected HTTP endpoint %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{}}`)
	}))
	defer server.Close()
	host, portText, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "https://"))
	port, _ := strconv.Atoi(portText)
	dialer := &tcpOnlyTestDialer{}
	c := NewClient(ClientOptions{
		Session:        SessionOptions{SID: "test-session", DeviceID: "test-device"},
		UnderlayDialer: dialer,
	})
	defer c.Close()
	saved := false
	_, err := c.Setup(SetupOptions{
		TCPOnly: true, ServerAddress: host, ServerPort: port,
		ResourceData:   []byte(`{"data":{}}`),
		SaveClientData: func(data []byte) error { saved = len(data) > 0; return nil },
	})
	if err != nil {
		t.Fatalf("TCP-only startup required unavailable VIP/L3 service: %v", err)
	}
	if c.l3Tunnel != nil || c.ip != nil || dialer.excluded {
		t.Fatal("TCP-only startup initialized virtual IP or L3 state")
	}
	if !saved {
		t.Fatal("TCP-only startup lost session persistence")
	}
	if _, err := c.NewL3Conn(); err == nil {
		t.Fatal("L3 data path unexpectedly available")
	}
}
