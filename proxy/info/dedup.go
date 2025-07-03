package info

import (
	"fmt"
	"net"
	"sync"

	"github.com/bestruirui/bestsub/config"
	"github.com/panjf2000/ants/v2"
)

var (
	dedupProxies map[string]*Proxy
	dedupMutex   sync.Mutex
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
	serverip, err := net.LookupIP(server)
	if err != nil {
		return
	}
	if len(serverip) == 0 {
		return
	}
	key := fmt.Sprintf("%s:%v", serverip[0], port)

	addDedupProxy(key, p)
}
