package proxies

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bestruirui/mihomo-check/config"
	"github.com/bestruirui/mihomo-check/proxy/parser"
	"github.com/metacubex/mihomo/log"
	"gopkg.in/yaml.v3"
)

func GetProxies() ([]map[string]any, error) {
	totalSubs := len(config.GlobalConfig.SubUrls)
	log.Infoln("当前共设置了%d个订阅链接", totalSubs)
	var mihomoProxies []map[string]any

	for i, subUrl := range config.GlobalConfig.SubUrls {
		log.Infoln("正在获取订阅 (%d/%d): %s", i+1, totalSubs, subUrl)
		data, err := GetDateFromSubs(subUrl)
		if err != nil {
			log.Errorln("获取订阅失败 (%d/%d): %s, 错误: %v", i+1, totalSubs, subUrl, err)
			continue
		}
		log.Infoln("成功获取订阅 (%d/%d): %s", i+1, totalSubs, subUrl)
        
		var config map[string]any
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			reg, _ := regexp.Compile("(ssr|ss|vmess|trojan|vless|hysteria|hy2|hysteria2)://")
			// 如果不匹配则base64解码
			if !reg.Match(data) {
				data = []byte(parser.DecodeBase64(string(data)))
			}
			if reg.Match(data) {
				proxies := strings.Split(string(data), "\n")

				for _, proxy := range proxies {
					parseProxy, err := ParseProxy(proxy)
					if err != nil {
						continue
					}
					if parseProxy == nil {
						continue
					}
					mihomoProxies = append(mihomoProxies, parseProxy)
				}
				continue // 改为 continue 而不是 return，处理完当前订阅后继续下一个
			}
		}
		proxyInterface, ok := config["proxies"]
		if !ok || proxyInterface == nil {
			log.Errorln("订阅链接: %s 没有proxies", subUrl)
			continue
		}

		proxyList, ok := proxyInterface.([]any)
		if !ok {
			continue
		}

		for _, proxy := range proxyList {
			proxyMap, ok := proxy.(map[string]any)
			if !ok {
				continue
			}
			mihomoProxies = append(mihomoProxies, proxyMap)
		}
	}
	
	if len(mihomoProxies) == 0 {
		return nil, fmt.Errorf("没有找到可用的代理节点")
	}
	return mihomoProxies, nil
}

// 订阅链接中获取数据
func GetDateFromSubs(subUrl string) ([]byte, error) {
	maxRetries := 10
	var lastErr error

	client := &http.Client{}

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
