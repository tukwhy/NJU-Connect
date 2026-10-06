package main

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/mythologyli/zju-connect/client/atrust/auth"
)

func TestNJUProfileAndOverrides(t *testing.T) {
	file := writeConfig(t, "profile = \"nju\"\nsocks_bind = \"127.0.0.1:12080\"\n")
	options, _, err := loadStartupOptions([]string{"--config", file, "--http-bind=127.0.0.1:12081"}, func() []string { return nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg := options.Config
	if cfg.ServerAddress != "vpn.nju.edu.cn" || cfg.LoginDomain != "auto" || cfg.Protocol != "atrust" || !cfg.TCPTunnelMode {
		t.Fatalf("NJU connection defaults missing: %#v", cfg)
	}
	if cfg.SocksBind != "127.0.0.1:12080" || cfg.HTTPBind != "127.0.0.1:12081" {
		t.Fatalf("file/CLI overrides lost: %#v", cfg)
	}
	if len(cfg.PortForwardingList) != 0 || cfg.CheckTarget != "" || cfg.ClashRulesFile == "" {
		t.Fatal("NJU defaults must provide a general campus proxy, without a fixed SSH target")
	}
}

func TestNJURejectsNetworkModeOverridesFromEverySource(t *testing.T) {
	file := writeConfig(t, "profile = \"nju\"\nadd_route = true\n")
	for _, tt := range []struct {
		name string
		args []string
		env  []string
	}{
		{"file route", []string{"--config", file}, nil},
		{"environment TUN", []string{"--profile=nju"}, []string{"ZJU_CONNECT_TUN_MODE=true"}},
		{"CLI DNS hijack", []string{"--profile=nju", "--dns-hijack"}, nil},
		{"CLI L3", []string{"--profile=nju", "--tcp-tunnel-mode=false"}, nil},
		{"LAN listener", []string{"--profile=nju", "--socks-bind=0.0.0.0:11080"}, nil},
		{"direct fallback", []string{"--profile=nju", "--proxy-all=false"}, nil},
		{"UDP forward", []string{"--profile=nju", "--udp-port-forwarding=127.0.0.1:1234-10.0.0.1:1234"}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := loadStartupOptions(tt.args, func() []string { return tt.env }); err == nil {
				t.Fatal("unsafe NJU configuration accepted")
			}
		})
	}
}

func TestLockedNJUBuildCannotDisableProfile(t *testing.T) {
	previous := defaultProfile
	defaultProfile = "nju"
	t.Cleanup(func() { defaultProfile = previous })
	for _, args := range [][]string{{"--profile="}, {"--profile=other"}} {
		if _, _, err := loadStartupOptions(args, func() []string { return nil }); err == nil {
			t.Fatal("locked profile could be disabled")
		}
	}
	options, _, err := loadStartupOptions(nil, func() []string { return nil })
	if err != nil || options.Config.Profile != "nju" {
		t.Fatalf("locked build did not load NJU defaults: %v", err)
	}
}

func TestNJUAuthenticationDiscovery(t *testing.T) {
	info := []auth.AuthInfo{
		{LoginDomain: "customOAuth51300", AuthType: "auth/httpsOauth2"},
		{LoginDomain: "openldap13924", AuthType: "auth/psw"},
		{AuthType: "auth/qrcode"},
	}
	for _, authType := range []string{"auth/psw", "auth/httpsOauth2"} {
		domain, err := selectLoginDomain(info, authType)
		if err != nil || domain == "" {
			t.Fatalf("discovery failed: %v", err)
		}
	}
	info = append(info, auth.AuthInfo{LoginDomain: "second", AuthType: "auth/psw"})
	if _, err := selectLoginDomain(info, "auth/psw"); err == nil {
		t.Fatal("ambiguous login silently selected a domain")
	}
	if _, err := selectLoginDomain(info, "auth/cas"); err == nil {
		t.Fatal("unavailable method accepted")
	}
}

func TestSSHCheckReadsBannerWithoutSendingCredentials(t *testing.T) {
	for _, tt := range []struct {
		name string
		data string
		ok   bool
	}{
		{"SSH", "Welcome\r\nSSH-2.0-OpenSSH_test\r\n", true},
		{"HTTP", "HTTP/1.1 200 OK\r\n", false},
		{"legacy SSH", "SSH-1.5-old\r\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			local, peer := net.Pipe()
			done := make(chan []byte, 1)
			go func() {
				defer peer.Close()
				_, _ = io.WriteString(peer, tt.data)
				if !tt.ok {
					_ = peer.Close()
				}
				data, _ := io.ReadAll(peer)
				done <- data
			}()
			banner, err := checkSSHBanner(context.Background(), func(_ context.Context, network, target string) (net.Conn, error) {
				if network != "tcp" || target != "10.0.0.42:22" {
					t.Fatal("check used an unexpected route")
				}
				return local, nil
			}, "10.0.0.42:22")
			if (err == nil) != tt.ok || (tt.ok && !strings.HasPrefix(banner, "SSH-2.0-")) {
				t.Fatalf("banner %q, error %v", banner, err)
			}
			if sent := <-done; len(sent) != 0 {
				t.Fatal("banner check transmitted data")
			}
		})
	}
}
