package info

import (
	"fmt"
	"net"
	"sync"

	"github.com/bestruirui/bestsub/config"
	"github.com/panjf2000/ants/v2"
)

var (
	dedupProxies   map[string]*Proxy
	dedupMutex     sync.Mutex
	dnsCache       map[string]string
	dnsCacheMutex  sync.Mutex
)

func addDedupProxy(key string, p *Proxy) {
	dedupMutex.Lock()
	defer dedupMutex.Unlock()
	if _, exists := dedupProxies[key]; !exists {
		dedupProxies[key] = p
	}
}

func DeduplicateProxies(proxies *[]Proxy) {
	var wg sync.WaitGroup
	dedupProxies = make(map[string]*Proxy)
	dnsCache = make(map[string]string)

	pool, _ := ants.NewPool(config.GlobalConfig.Check.Concurrent)
	defer pool.Release()

	for i := range *proxies {
		wg.Add(1)
		i := i
		pool.Submit(func() {
			defer wg.Done()
			deduplicateTask(&(*proxies)[i])
		})
	}
	wg.Wait()

	*proxies = (*proxies)[:0]
	for _, proxy := range dedupProxies {
		*proxies = append(*proxies, *proxy)
	}

	dedupProxies = nil
	dnsCache = nil
}

func resolveServerKey(server string) string {
	dnsCacheMutex.Lock()
	if cached, ok := dnsCache[server]; ok {
		dnsCacheMutex.Unlock()
		return cached
	}
	dnsCacheMutex.Unlock()

	serverIP, err := net.LookupIP(server)
	resolved := server
	if err == nil && len(serverIP) > 0 {
		resolved = serverIP[0].String()
	}

	dnsCacheMutex.Lock()
	dnsCache[server] = resolved
	dnsCacheMutex.Unlock()
	return resolved
}

func deduplicateTask(p *Proxy) {
	arg := p.Raw
	server, serverOk := "", false
	if arg["type"] == "vless" || arg["type"] == "vmess" {
		server, serverOk = arg["servername"].(string)
		if !serverOk || server == "" {
			server, serverOk = arg["server"].(string)
		}
	} else {
		server, serverOk = arg["server"].(string)
	}
	port, portOk := arg["port"].(int)

	if !serverOk || !portOk {
		return
	}

	key := fmt.Sprintf("%s:%v", resolveServerKey(server), port)
	addDedupProxy(key, p)
}
