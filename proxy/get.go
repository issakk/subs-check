package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/proxy/info"
	"github.com/bestruirui/bestsub/proxy/parser"
	"github.com/bestruirui/bestsub/utils"
	"github.com/bestruirui/bestsub/utils/log"
	"github.com/panjf2000/ants/v2"
	"gopkg.in/yaml.v3"
)

var mihomoProxiesMutex sync.Mutex

func GetProxies(proxies *[]info.Proxy) {
	log.Info("subscription links count: %v", len(config.GlobalConfig.SubUrls))
	numWorkers := min(len(config.GlobalConfig.SubUrls), config.GlobalConfig.Check.Concurrent)

	pool, _ := ants.NewPool(numWorkers)
	defer pool.Release()
	var wg sync.WaitGroup
	for _, subUrl := range config.GlobalConfig.SubUrls {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			taskGetProxies(subUrl, proxies)
		})
	}
	wg.Wait()
}

func taskGetProxies(args string, proxiesInfo *[]info.Proxy) {

	data, err := getDateFromSubs(args)
	if err != nil {
		log.Warn("subscription link: %s get data failed: %v", log.MaskURL(args), err)
		return
	}
	if IsYaml(data, args) {
		err := ParseYamlProxy(data, proxiesInfo)
		if err != nil {
			log.Warn("subscription link: %s has no proxies", log.MaskURL(args))
			return
		}
	} else {
		reg, _ := regexp.Compile(`^(ssr://|ss://|vmess://|trojan://|vless://|hysteria://|hy2://|hysteria2://)`)
		if !reg.Match(data) {
			log.Debug("subscription link: %s is not a v2ray subscription link, attempting to decode the subscription link using base64", log.MaskURL(args))
			data = []byte(parser.DecodeBase64(string(data)))
		}
		if reg.Match(data) {
			proxies := strings.Split(string(data), "\n")

			for _, proxy := range proxies {
				parseProxy, err := parser.ParseProxy(proxy)
				if err != nil {
					continue
				}
				if parseProxy == nil {
					continue
				}
				if len(config.GlobalConfig.TypeInclude) > 0 {
					for _, t := range config.GlobalConfig.TypeInclude {
						if t == parseProxy["type"].(string) {
							mihomoProxiesMutex.Lock()
							*proxiesInfo = append(*proxiesInfo, info.Proxy{Raw: parseProxy})
							mihomoProxiesMutex.Unlock()
							break
						}
					}
				} else {
					mihomoProxiesMutex.Lock()
					*proxiesInfo = append(*proxiesInfo, info.Proxy{Raw: parseProxy})
					mihomoProxiesMutex.Unlock()
				}

			}
		}
	}
}

func getDateFromSubs(subUrl string) ([]byte, error) {
	var lastErr error
	client := utils.NewHTTPClient()
	maxRetries := config.GlobalConfig.SubUrlsReTry

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
		req.Close = true

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("subscription link: %s returned status code: %d", subUrl, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}
		return body, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}
func removeAllControlCharacters(data []byte) []byte {
	var cleanedData []byte
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r != utf8.RuneError && (r >= 32 && r <= 126) || r == '\n' || r == '\t' || r == '\r' || unicode.Is(unicode.Han, r) {
			cleanedData = append(cleanedData, data[:size]...)
		}
		data = data[size:]
	}
	return cleanedData
}

func IsYaml(data []byte, subUrl string) bool {
	reg, _ := regexp.Compile(`^(ssr://|ss://|vmess://|trojan://|vless://|hysteria://|hy2://|hysteria2://)`)

	decodedData := parser.DecodeBase64(string(data))
	if reg.MatchString(decodedData) {
		log.Debug("subscription link: %s is a v2ray subscription link", log.MaskURL(subUrl))
		return false
	}

	if bytes.Contains(data, []byte("proxies:")) {
		log.Debug("subscription link: %s is a yaml file", log.MaskURL(subUrl))
		return true
	}
	return false
}
func ParseYamlProxy(data []byte, proxies *[]info.Proxy) error {
	var inProxiesSection bool
	var yamlBuffer bytes.Buffer
	var indent int
	var isFirst bool = true

	cleandata := removeAllControlCharacters(data)
	cleanedFile := bytes.NewReader(cleandata)
	scanner := bufio.NewScanner(cleanedFile)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "proxies:" {
			inProxiesSection = true
			continue
		}

		if !inProxiesSection {
			continue
		}

		if isFirst {
			indent = len(line) - len(trimmedLine)
			isFirst = false
		}

		if len(line)-len(trimmedLine) == 0 && !strings.HasPrefix(trimmedLine, "-") && trimmedLine != "" {
			break
		}

		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		if strings.HasPrefix(trimmedLine, "-") && len(line)-len(trimmedLine) == indent {
			if yamlBuffer.Len() > 0 {
				var proxy []map[string]any
				if err := yaml.Unmarshal(yamlBuffer.Bytes(), &proxy); err != nil {

				} else {
					if len(config.GlobalConfig.TypeInclude) > 0 {
						for _, t := range config.GlobalConfig.TypeInclude {
							if t == proxy[0]["type"].(string) {
								mihomoProxiesMutex.Lock()
								*proxies = append(*proxies, info.Proxy{Raw: proxy[0]})
								mihomoProxiesMutex.Unlock()
								break
							}
						}
					} else {
						mihomoProxiesMutex.Lock()
						*proxies = append(*proxies, info.Proxy{Raw: proxy[0]})
						mihomoProxiesMutex.Unlock()
					}
				}
				yamlBuffer.Reset()
			}
			yamlBuffer.WriteString(line + "\n")
		} else if yamlBuffer.Len() > 0 {
			yamlBuffer.WriteString(line + "\n")
		}
	}

	if yamlBuffer.Len() > 0 {
		var proxy []map[string]any
		if err := yaml.Unmarshal(yamlBuffer.Bytes(), &proxy); err != nil {
		} else {
			if len(config.GlobalConfig.TypeInclude) > 0 {
				for _, t := range config.GlobalConfig.TypeInclude {
					if t == proxy[0]["type"].(string) {
						mihomoProxiesMutex.Lock()
						*proxies = append(*proxies, info.Proxy{Raw: proxy[0]})
						mihomoProxiesMutex.Unlock()
						break
					}
				}
			} else {
				mihomoProxiesMutex.Lock()
				*proxies = append(*proxies, info.Proxy{Raw: proxy[0]})
				mihomoProxiesMutex.Unlock()
			}
		}
	}

	return nil
}
