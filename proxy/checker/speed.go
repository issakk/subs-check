package checker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/utils/log"
	"github.com/dlclark/regexp2"
)

func (c *Checker) CheckSpeed() {

	if config.GlobalConfig.Check.SpeedSkipName != "" {
		re := regexp2.MustCompile(config.GlobalConfig.Check.SpeedSkipName, regexp2.None)
		match, err := re.MatchString(c.Proxy.Raw["name"].(string))
		if err != nil {
			log.Debug("check speed skip name failed: %v", err)
			return
		}
		if match {
			c.Proxy.Info.SpeedSkip = true
			log.Debug("check speed skip : %v", c.Proxy.Raw["name"])
			return
		}
	}

	ctx, cancel := context.WithCancel(c.Proxy.Ctx)
	defer cancel()

	speedClient := &http.Client{
		Timeout:   time.Duration(config.GlobalConfig.Check.DownloadTimeout) * time.Second,
		Transport: c.Proxy.Client.Transport,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", config.GlobalConfig.Check.SpeedTestUrl, nil)
	if err != nil {
		return
	}

	var startTime time.Time
	var totalBytes int64

	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			startTime = time.Now()
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := speedClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	totalBytes, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return
	}

	duration := time.Since(startTime).Milliseconds()
	if duration == 0 {
		duration = 1
	}

	c.Proxy.Info.Speed = int(float64(totalBytes) / 1024 * 1000 / float64(duration))
}
