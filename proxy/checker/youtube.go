package checker

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func (c *Checker) YoutubeTest() {
	ctx, cancel := context.WithCancel(c.Proxy.Ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.youtube.com/premium", nil)
	if err != nil {
		return
	}

	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Set("sec-ch-ua", `"Chromium";v="131", "Not_A Brand";v="24", "Google Chrome";v="131"`)
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "none")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := c.Proxy.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// "countryCode" sits near the top of the page; capping the read avoids
	// pulling the whole multi-hundred-KB html for every checked node.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return
	}
	idx := strings.Index(string(body), `"countryCode"`)
	if idx != -1 {
		region := strings.Replace(string(body)[idx:idx+17], `"countryCode":"`, "", 1)
		if region != "" {
			c.Proxy.Info.Unlock.Youtube = true
		}
	}
}
