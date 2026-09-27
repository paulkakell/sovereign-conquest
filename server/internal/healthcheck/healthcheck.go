// Package healthcheck provides the shell-free container liveness probe.
package healthcheck

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Keep the probe's deadline shorter than Docker's five-second health timeout.
const timeout = 3 * time.Second

// Run checks process liveness at the configured HTTP listen address. An empty
// address uses the same :8080 default as the API. It requires no database or
// authentication configuration and never includes response content in errors.
func Run(address string) error {
	return check(address, timeout)
}

func check(address string, deadline time.Duration) error {
	target, err := endpoint(address)
	if err != nil {
		return err
	}
	transport := &http.Transport{
		// Explicitly ignore HTTP_PROXY, HTTPS_PROXY and their lowercase aliases.
		Proxy:                  nil,
		DialContext:            (&net.Dialer{Timeout: deadline}).DialContext,
		DisableKeepAlives:      true,
		DisableCompression:     true,
		ResponseHeaderTimeout:  deadline,
		MaxResponseHeaderBytes: 8 << 10,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   deadline,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(target)
	if err != nil {
		// Transport errors may contain addresses, credentials, or server data.
		return errors.New("liveness request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("liveness endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func endpoint(address string) (string, error) {
	if address == "" {
		address = ":8080"
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", errors.New("invalid HTTP_ADDR: expected host:port")
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return "", errors.New("invalid HTTP_ADDR: expected port 1-65535")
	}
	if host == "" {
		host = "127.0.0.1"
	} else if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		if ip.To4() != nil {
			host = "127.0.0.1"
		} else {
			host = "::1"
		}
	}
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   "/api/livez",
	}).String(), nil
}
