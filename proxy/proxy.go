package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/proxy/info"
	"github.com/metacubex/mihomo/adapter"
)

func NewProxy(raw map[string]any) *info.Proxy {
	proxy, err := adapter.ParseProxy(raw)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &info.Proxy{
		Raw:    raw,
		Ctx:    ctx,
		Cancel: cancel,
		Client: &http.Client{
			Timeout:   time.Duration(config.GlobalConfig.Check.Timeout) * time.Millisecond,
			Transport: info.BuildTransport(proxy, ctx),
		},
	}
}
