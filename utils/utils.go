package utils

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ParseProxyURL парсит tg://proxy?... или https://t.me/proxy?...
// возвращает server, port, secret в байтах.
func ParseProxyURL(raw string) (server string, port int, secret []byte, err error) {
	var queryStr string

	switch {
	case strings.HasPrefix(raw, "tg://proxy?"):
		queryStr = strings.TrimPrefix(raw, "tg://proxy?")
	case strings.HasPrefix(raw, "https://t.me/proxy?"):
		queryStr = strings.TrimPrefix(raw, "https://t.me/proxy?")
	case strings.HasPrefix(raw, "http://t.me/proxy?"):
		queryStr = strings.TrimPrefix(raw, "http://t.me/proxy?")
	default:
		return "", 0, nil, fmt.Errorf("unsupported URL format")
	}

	params, err := url.ParseQuery(queryStr)
	if err != nil {
		return "", 0, nil, fmt.Errorf("malformed query string: %w", err)
	}

	server = params.Get("server")
	if server == "" {
		return "", 0, nil, fmt.Errorf("missing parameter: server")
	}

	portStr := params.Get("port")
	if portStr == "" {
		return "", 0, nil, fmt.Errorf("missing parameter: port")
	}
	if _, scanErr := fmt.Sscanf(portStr, "%d", &port); scanErr != nil {
		return "", 0, nil, fmt.Errorf("invalid port %q: %w", portStr, scanErr)
	}
	if port < 1 || port > 65535 {
		return "", 0, nil, fmt.Errorf("port %d out of range [1, 65535]", port)
	}

	secretStr := params.Get("secret")
	if secretStr == "" {
		return "", 0, nil, fmt.Errorf("missing parameter: secret")
	}

	secret, err = decodeSecret(secretStr)
	if err != nil {
		return "", 0, nil, err
	}

	return server, port, secret, nil
}

// decodeSecret декодирует секрет из hex или base64url в байты.
func decodeSecret(s string) ([]byte, error) {
	isHex := len(s) > 0
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			isHex = false
			break
		}
	}

	if isHex {
		if len(s)%2 != 0 {
			return nil, fmt.Errorf("invalid secret: hex string has odd length")
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("invalid secret: %w", err)
		}
		return b, nil
	}

	normalized := strings.NewReplacer("-", "+", "_", "/").Replace(s)
	switch len(normalized) % 4 {
	case 2:
		normalized += "=="
	case 3:
		normalized += "="
	case 1:
		return nil, fmt.Errorf("invalid secret: invalid base64 length")
	}

	b, err := base64.StdEncoding.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: cannot decode as hex or base64: %w", err)
	}
	return b, nil
}

// BuildDialer возвращает dial-функцию привязанную к указанному локальному адресу.
// Если localAddr пуст — возвращает nil и gotd использует системный диалер по умолчанию.
func BuildDialer(localAddr string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if localAddr == "" {
		return nil
	}
	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{
			IP: net.ParseIP(localAddr),
		},
	}
	return dialer.DialContext
}
