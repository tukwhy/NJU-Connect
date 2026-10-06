package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/mythologyli/zju-connect/client/atrust/auth"
	"github.com/mythologyli/zju-connect/configs"
)

func defaultsForProfile(profile string) (configs.Config, error) {
	if profile != "nju" {
		return configs.Config{}, fmt.Errorf("unknown profile %q", profile)
	}
	cfg := configs.Default()
	cfg.Profile = profile
	cfg.Protocol = "atrust"
	cfg.ServerAddress = "vpn.nju.edu.cn"
	cfg.AuthType = "auth/psw"
	cfg.LoginDomain = "auto"
	cfg.SocksBind = "127.0.0.1:11080"
	cfg.HTTPBind = "127.0.0.1:11081"
	cfg.TCPTunnelMode = true
	// A school proxy must refuse destinations outside its granted resources,
	// rather than silently sending them through the workstation's other routes.
	cfg.ProxyAll = true
	cfg.DisableRemoteDNS = true
	cfg.ClashRulesFile = "clash-nju.generated.yaml"
	return cfg, nil
}

func validateProfile(cfg configs.Config) error {
	if defaultProfile == "nju" && cfg.Profile != "nju" {
		return fmt.Errorf("this NJU build requires profile nju; system networking modes are disabled")
	}
	if cfg.Profile == "" {
		return nil
	}
	if cfg.Profile != "nju" {
		return fmt.Errorf("unknown profile %q", cfg.Profile)
	}
	if cfg.Protocol != "atrust" || !cfg.TCPTunnelMode || cfg.TUNMode || cfg.AddRoute || cfg.DNSHijack || cfg.FakeIP {
		return fmt.Errorf("NJU profile requires aTrust TCP-only mode; TUN, route changes, DNS hijack and fake IP must stay disabled")
	}
	if cfg.ShadowsocksURL != "" || cfg.DNSServerBind != "" {
		return fmt.Errorf("NJU profile only supports local TCP proxies and forwarding; DNS and Shadowsocks listeners must stay disabled")
	}
	if !cfg.ProxyAll || cfg.DisableServerConfig {
		return fmt.Errorf("NJU profile requires proxy-all and server resources to prevent direct fallback for school destinations")
	}
	for _, address := range []string{cfg.SocksBind, cfg.HTTPBind} {
		if address != "" {
			if err := validateLoopbackListener(address); err != nil {
				return err
			}
		}
	}
	for _, forwarding := range cfg.PortForwardingList {
		if forwarding.NetworkType != "tcp" {
			return fmt.Errorf("NJU profile only supports TCP forwarding")
		}
		if err := validateLoopbackListener(forwarding.BindAddress); err != nil {
			return err
		}
	}
	if cfg.CheckTarget != "" {
		_, port, err := net.SplitHostPort(cfg.CheckTarget)
		if err != nil {
			return fmt.Errorf("invalid check-target: %w", err)
		}
		if err := validatePort(port); err != nil {
			return err
		}
	}
	return nil
}

func validateLoopbackListener(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid local listener %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("NJU listeners must bind to a numeric loopback address: %q", address)
	}
	return validatePort(port)
}

func validatePort(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %q", value)
	}
	return nil
}

func selectLoginDomain(info []auth.AuthInfo, authType string) (string, error) {
	domains := make(map[string]bool)
	var candidates []string
	for _, method := range info {
		if method.AuthType == authType && method.LoginDomain != "" && !domains[method.LoginDomain] {
			domains[method.LoginDomain] = true
			candidates = append(candidates, method.LoginDomain)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("NJU gateway does not advertise %s; use --auth-info to inspect its login methods", authType)
	}
	return "", fmt.Errorf("multiple login domains for %s: %s; choose --login-domain explicitly", authType, strings.Join(candidates, ", "))
}

// checkSSHBanner does not send credentials or start an SSH authentication.
func checkSSHBanner(ctx context.Context, dial func(context.Context, string, string) (net.Conn, error), target string) (string, error) {
	conn, err := dial(ctx, "tcp", target)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline := time.Now().Add(8 * time.Second)
	if value, ok := ctx.Deadline(); ok && value.Before(deadline) {
		deadline = value
	}
	if err := conn.SetReadDeadline(deadline); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(io.LimitReader(conn, 8192))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "SSH-2.0-") || strings.HasPrefix(line, "SSH-1.99-") {
			return line, nil
		}
		if strings.HasPrefix(line, "SSH-") {
			return "", fmt.Errorf("unsupported SSH protocol version")
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read SSH banner: %w", err)
	}
	return "", fmt.Errorf("target did not provide an SSH banner")
}
