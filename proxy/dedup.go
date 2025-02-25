package proxies

import (
	"fmt"
)

func DeduplicateProxies(proxies []map[string]any) []map[string]any {
	seen := make(map[string]map[string]any)

	for _, proxy := range proxies {
		server, serverOk := proxy["server"].(string)
		port, portOk := proxy["port"].(int)
		if !serverOk || !portOk {
			continue
		}

		// 直接使用 server:port 作为唯一键
		key := fmt.Sprintf("%s:%v", server, port)

		if _, exists := seen[key]; !exists {
			seen[key] = proxy
		}
	}

	result := make([]map[string]any, 0, len(seen))
	for _, proxy := range seen {
		result = append(result, proxy)
	}

	return result
}
