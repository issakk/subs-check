package proxies

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/mihomo-check/config"
	"github.com/bestruirui/mihomo-check/proxy/parser"
	"github.com/metacubex/mihomo/log"
	"gopkg.in/yaml.v3"
)

func GetProxies() ([]map[string]any, error) {
	totalSubs := len(config.GlobalConfig.SubUrls)
	log.Infoln("当前共设置了%d个订阅链接", totalSubs)
	
	// 创建一个互斥锁保护共享资源
	var mu sync.Mutex
	var mihomoProxies []map[string]any
	var wg sync.WaitGroup
	
	// 创建错误通道
	errChan := make(chan error, totalSubs)
	
	// 设置最大并发数
	maxConcurrent := 5
	if totalSubs < maxConcurrent {
		maxConcurrent = totalSubs
	}
	
	// 创建信号量控制并发
	sem := make(chan struct{}, maxConcurrent)
	
	for i, subUrl := range config.GlobalConfig.SubUrls {
		wg.Add(1)
		
		go func(i int, subUrl string) {
			defer wg.Done()
			
			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()
			
			log.Infoln("正在获取订阅 (%d/%d): %s", i+1, totalSubs, subUrl)
			data, err := GetDateFromSubs(subUrl)
			if err != nil {
				log.Errorln("获取订阅失败 (%d/%d): %s, 错误: %v", i+1, totalSubs, subUrl, err)
				errChan <- err
				return
			}
			log.Infoln("成功获取订阅 (%d/%d): %s", i+1, totalSubs, subUrl)
			
			// 处理订阅数据
			proxies, err := processSubscriptionData(data)
			if err != nil {
				log.Errorln("处理订阅数据失败 (%d/%d): %s, 错误: %v", i+1, totalSubs, subUrl, err)
				return
			}
			
			// 安全地添加到结果列表
			if len(proxies) > 0 {
				mu.Lock()
				mihomoProxies = append(mihomoProxies, proxies...)
				mu.Unlock()
			}
		}(i, subUrl)
	}
	
	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)
	
	// 检查是否所有订阅都失败了
	if len(mihomoProxies) == 0 {
		// 收集所有错误
		var errMsgs []string
		for err := range errChan {
			errMsgs = append(errMsgs, err.Error())
		}
		
		if len(errMsgs) > 0 {
			return nil, fmt.Errorf("所有订阅获取失败: %s", strings.Join(errMsgs, "; "))
		}
		return nil, fmt.Errorf("没有找到可用的代理节点")
	}
	
	return mihomoProxies, nil
}

// 处理订阅数据
func processSubscriptionData(data []byte) ([]map[string]any, error) {
	var proxies []map[string]any
	var config map[string]any
	
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		reg, _ := regexp.Compile("(ssr|ss|vmess|trojan|vless|hysteria|hy2|hysteria2)://")
		// 如果不匹配则base64解码
		if !reg.Match(data) {
			data = []byte(parser.DecodeBase64(string(data)))
		}
		if reg.Match(data) {
			proxyLines := strings.Split(string(data), "\n")

			for _, proxy := range proxyLines {
				parseProxy, err := ParseProxy(proxy)
				if err != nil {
					continue
				}
				if parseProxy == nil {
					continue
				}
				proxies = append(proxies, parseProxy)
			}
			return proxies, nil
		}
		return nil, err
	}
	
	proxyInterface, ok := config["proxies"]
	if !ok || proxyInterface == nil {
		return nil, fmt.Errorf("订阅没有proxies字段")
	}

	proxyList, ok := proxyInterface.([]any)
	if !ok {
		return nil, fmt.Errorf("proxies字段不是数组")
	}

	for _, proxy := range proxyList {
		proxyMap, ok := proxy.(map[string]any)
		if !ok {
			continue
		}
		proxies = append(proxies, proxyMap)
	}
	
	return proxies, nil
}

// 订阅链接中获取数据
func GetDateFromSubs(subUrl string) ([]byte, error) {
	maxRetries := 3
	var lastErr error

	client := &http.Client{
		Timeout: 5 * time.Second, // 设置超时时间
	}

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Second)
		}

		req, err := http.NewRequest("GET", subUrl, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("User-Agent", "clash.meta")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("订阅链接: %s 返回状态码: %d", subUrl, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}
		return body, nil
	}

	return nil, fmt.Errorf("重试%d次后失败: %v", maxRetries, lastErr)
}
