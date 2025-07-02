package info

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/constant"
)

type Unlock struct {
	Google     bool
	Chatgpt    bool
	Netflix    bool
	Disney     bool
	Youtube    bool
	Cloudflare bool
}

type ProxyInfo struct {
	Unlock    Unlock
	Speed     int
	SpeedSkip bool
	Rate      float32
	Risk      int
	Delay     uint16
	Alive     bool
	Country   string
	Flag      string
}

type Proxy struct {
	Raw    map[string]any
	Id     int
	SubUrl string
	Ctx    context.Context
	Cancel context.CancelFunc
	Client *http.Client
	Info   ProxyInfo
}

func (p *Proxy) Close() {
	if p.Cancel != nil {
		p.Cancel()
	}
	if transport, ok := p.Client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}
func (p *Proxy) CloseTransport() {
	if transport, ok := p.Client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}
func (p *Proxy) New() error {
	p.Ctx, p.Cancel = context.WithCancel(context.Background())
	proxy, err := adapter.ParseProxy(p.Raw)
	if err != nil {
		return err
	}
	p.Client = &http.Client{
		Timeout:   time.Duration(config.GlobalConfig.Check.Timeout) * time.Millisecond,
		Transport: BuildTransport(proxy, p.Ctx),
	}
	return nil
}
func BuildTransport(proxy constant.Proxy, ctx context.Context) *http.Transport {
	transport := &http.Transport{
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			var u16Port uint16
			if port, err := strconv.ParseUint(port, 10, 16); err == nil {
				u16Port = uint16(port)
			}

			return proxy.DialContext(ctx, &constant.Metadata{
				Host:    host,
				DstPort: u16Port,
			})
		},
		MaxIdleConns:          0,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
		ForceAttemptHTTP2:     false,
		MaxConnsPerHost:       0,
		MaxIdleConnsPerHost:   0,
	}
	return transport
}
