package checker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync/atomic"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/utils/log"
	"github.com/dlclark/regexp2"
)

func (c *Checker) CheckSpeed() {
	if config.GlobalConfig.Check.SpeedSkipName != "" {
		re, err := regexp2.Compile(config.GlobalConfig.Check.SpeedSkipName, regexp2.None)
		if err != nil {
			log.Debug("compile speed skip name failed: %v", err)
			return
		}
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

	speedClient := &http.Client{
		Timeout:   time.Duration(config.GlobalConfig.Check.DownloadTimeout) * time.Second,
		Transport: c.Proxy.Client.Transport,
	}

	for _, url := range config.GlobalConfig.Check.SpeedTestUrl {
		reqCtx, cancel := context.WithTimeout(c.Proxy.Ctx, time.Duration(config.GlobalConfig.Check.Timeout)*time.Second)

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			continue
		}

		// GotFirstResponseByte fires on a transport goroutine, so the timestamp
		// must be atomic to be read safely from this one.
		var firstByteNano atomic.Int64
		trace := &httptrace.ClientTrace{
			GotFirstResponseByte: func() {
				firstByteNano.CompareAndSwap(0, time.Now().UnixNano())
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

		resp, err := speedClient.Do(req)
		if err != nil {
			cancel()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			log.Debug("speed test url %v returned status %d, skip", url, resp.StatusCode)
			continue
		}

		limitedReader := &io.LimitedReader{
			R: resp.Body,
			N: int64(config.GlobalConfig.Check.DownloadSize) * 1024 * 1024,
		}
		var bytesRead atomic.Int64
		done := make(chan struct{})
		copyCtx, copyCancel := context.WithTimeout(c.Proxy.Ctx, time.Duration(config.GlobalConfig.Check.DownloadTimeout)*time.Second)
		go func() {
			defer close(done)
			buf := make([]byte, 32*1024)
			for {
				n, err := limitedReader.Read(buf)
				if n > 0 {
					bytesRead.Add(int64(n))
				}
				if err != nil {
					break
				}
			}
		}()

		timeoutOccurred := false
		select {
		case <-done:
		case <-copyCtx.Done():
			timeoutOccurred = true
		}

		resp.Body.Close()
		// Closing the body makes the download goroutine exit immediately;
		// wait briefly so the byte count covers everything actually read.
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		copyCancel()
		cancel()

		totalBytes := bytesRead.Load()
		if totalBytes > 0 {
			startNano := firstByteNano.Load()
			if startNano == 0 {
				startNano = time.Now().UnixNano()
			}
			duration := (time.Now().UnixNano() - startNano) / int64(time.Millisecond)
			if duration <= 0 {
				duration = 1
			}

			c.Proxy.Info.Speed = int(float64(totalBytes) / 1024 * 1000 / float64(duration))

			if timeoutOccurred {
				log.Debug("Speed test for %v timed out but partial speed calculated: %v KB/s",
					c.Proxy.Raw["name"], c.Proxy.Info.Speed)
			}

			break
		}
	}
}
