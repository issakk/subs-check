package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func ParseVless(data string) (map[string]any, error) {
	parsedURL, err := url.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %v", err)
	}

	if parsedURL.Scheme != "vless" {
		return nil, fmt.Errorf("not vless format")
	}

	hostPort := strings.Split(parsedURL.Host, ":")
	if len(hostPort) != 2 {
		return nil, nil
	}

	port, err := strconv.Atoi(parsedURL.Port())
	if err != nil {
		return nil, fmt.Errorf("format error: incorrect port format")
	}

	query := parsedURL.Query()
	security := query.Get("security")

	proxy := map[string]any{
		"name":               parsedURL.Fragment,
		"type":               "vless",
		"server":             parsedURL.Hostname(),
		"port":               port,
		"uuid":               parsedURL.User.String(),
		"network":            query.Get("type"),
		"tls":                security == "tls" || security == "reality",
		"udp":                query.Get("udp") == "true",
		"servername":         query.Get("sni"),
		"flow":               query.Get("flow"),
		"client-fingerprint": query.Get("fp"),
		"ws-opts": map[string]any{
			"path": query.Get("path"),
			"headers": map[string]any{
				"Host": query.Get("host"),
			},
		},
		"reality-opts": map[string]any{
			"public-key": query.Get("pbk"),
			"short-id":   query.Get("sid"),
		},
		"grpc-opts": map[string]any{
			"grpc-service-name": query.Get("serviceName"),
		},
		"security":    security,
		"sni":         query.Get("sni"),
		"fp":          query.Get("fp"),
		"pbk":         query.Get("pbk"),
		"sid":         query.Get("sid"),
		"path":        query.Get("path"),
		"host":        query.Get("host"),
		"serviceName": query.Get("serviceName"),
		"mode":        query.Get("mode"),
	}

	return proxy, nil
}
