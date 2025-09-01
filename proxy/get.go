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
		// copy subUrl to a new variable
		subUrl := subUrl
		pool.Submit(func() {
			defer wg.Done()
			processedUrl := replaceDateTimePlaceholders(subUrl)
			taskGetProxies(processedUrl, proxies)
		})
	}
	wg.Wait()
}

func replaceDateTimePlaceholders(url string) string {
	now := time.Now()
	r := strings.NewReplacer(
		"{YYYY}", now.Format("2006"),
		"{MM}", now.Format("01"),
		"{DD}", now.Format("02"),
		"{HH}", now.Format("15"),
		"{mm}", now.Format("04"),
		"{ss}", now.Format("05"),
	)
	return r.Replace(url)
}

func taskGetProxies(args string, proxiesInfo *[]info.Proxy) {

	data, err := getDateFromSubs(args)
	if err != nil {
		log.Warn("subscription link [%s] get data failed: %v", args, err)
		return
	}
	if IsYaml(data, args) {
		err := ParseYamlProxy(data, proxiesInfo, args)
		if err != nil {
			log.Warn("subscription link [%s] has no proxies", args)
			return
		}
	} else {
		reg, _ := regexp.Compile(`^(ssr://|ss://|vmess://|trojan://|vless://|hysteria://|hy2://|hysteria2://)`)
		if !reg.Match(data) {
			log.Debug("subscription link [%s] is not a v2ray subscription link, attempting to decode the subscription link using base64", args)
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
							*proxiesInfo = append(*proxiesInfo, info.Proxy{Raw: parseProxy, SubUrl: args})
							mihomoProxiesMutex.Unlock()
							break
						}
					}
				} else {
					mihomoProxiesMutex.Lock()
					*proxiesInfo = append(*proxiesInfo, info.Proxy{Raw: parseProxy, SubUrl: args})
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
			lastErr = fmt.Errorf("subscription link [%s] returned status code: %d", subUrl, resp.StatusCode)
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
		log.Debug("subscription link [%s] is a v2ray subscription link", subUrl)
		return false
	}

	if bytes.Contains(data, []byte("proxies:")) {
		log.Debug("subscription link [%s] is a yaml file", subUrl)
		return true
	}
	return false
}
func ParseYamlProxy(data []byte, proxies *[]info.Proxy, subUrl string) error {
	log.Debug("Entering ParseYamlProxy for subUrl: %s", subUrl)
	var inProxiesSection bool
	var yamlBuffer bytes.Buffer
	var indent int
	var isFirst bool = true

	cleandata := removeAllControlCharacters(data)
	cleanedFile := bytes.NewReader(cleandata)
	scanner := bufio.NewScanner(cleanedFile)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		log.Debug("Processing line %d: %s", lineNum, trimmedLine)

		if trimmedLine == "proxies:" {
			inProxiesSection = true
			log.Debug("Found 'proxies:' section at line %d", lineNum)
			continue
		}

		if !inProxiesSection {
			continue
		}

		if isFirst {
			indent = len(line) - len(trimmedLine)
			isFirst = false
			log.Debug("Determined indent: %d", indent)
		}

		if len(line)-len(trimmedLine) == 0 && !strings.HasPrefix(trimmedLine, "-") && trimmedLine != "" {
			log.Debug("Exiting proxies section at line %d due to unindented line: %s", lineNum, trimmedLine)
			break
		}

		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			log.Debug("Skipping empty or comment line %d", lineNum)
			continue
		}

		if strings.HasPrefix(trimmedLine, "-") && len(line)-len(trimmedLine) == indent {
			if yamlBuffer.Len() > 0 {
				log.Debug("Attempting to unmarshal YAML buffer at line %d. Buffer size: %d", lineNum, yamlBuffer.Len())
				var proxy []map[string]any
				if err := yaml.Unmarshal(yamlBuffer.Bytes(), &proxy); err != nil {
					log.Warn("Failed to unmarshal YAML proxy from sub [%s] at line %d: %v. Buffer content: %s", subUrl, lineNum, err, yamlBuffer.String())
				} else {
					log.Debug("Successfully unmarshaled proxy at line %d. Proxy type: %s", lineNum, proxy[0]["type"].(string))
					if len(config.GlobalConfig.TypeInclude) > 0 {
						for _, t := range config.GlobalConfig.TypeInclude {
							if t == proxy[0]["type"].(string) {
								mihomoProxiesMutex.Lock()
								*proxies = append(*proxies, info.Proxy{Raw: proxy[0], SubUrl: subUrl})
								mihomoProxiesMutex.Unlock()
								break
							}
						}
					} else {
						mihomoProxiesMutex.Lock()
						*proxies = append(*proxies, info.Proxy{Raw: proxy[0], SubUrl: subUrl})
						mihomoProxiesMutex.Unlock()
					}
				}
				yamlBuffer.Reset()
				log.Debug("YAML buffer reset.")
			}
			yamlBuffer.WriteString(line + "\n")
			log.Debug("Added line %d to YAML buffer. Current buffer size: %d", lineNum, yamlBuffer.Len())
		} else if yamlBuffer.Len() > 0 {
			yamlBuffer.WriteString(line + "\n")
			log.Debug("Added line %d to YAML buffer (continuation). Current buffer size: %d", lineNum, yamlBuffer.Len())
		}
	}

	if yamlBuffer.Len() > 0 {
		log.Debug("Attempting to unmarshal remaining YAML buffer after loop. Buffer size: %d", yamlBuffer.Len())
		var proxy []map[string]any
		if err := yaml.Unmarshal(yamlBuffer.Bytes(), &proxy); err != nil {
			log.Warn("Failed to unmarshal remaining YAML proxy from sub [%s]: %v. Buffer content: %s", subUrl, err, yamlBuffer.String())
		} else {
			log.Debug("Successfully unmarshaled remaining proxy. Proxy type: %s", proxy[0]["type"].(string))
			if len(config.GlobalConfig.TypeInclude) > 0 {
				for _, t := range config.GlobalConfig.TypeInclude {
					if t == proxy[0]["type"].(string) {
						mihomoProxiesMutex.Lock()
						*proxies = append(*proxies, info.Proxy{Raw: proxy[0], SubUrl: subUrl})
						mihomoProxiesMutex.Unlock()
						break
					}
				}
			} else {
				mihomoProxiesMutex.Lock()
				*proxies = append(*proxies, info.Proxy{Raw: proxy[0], SubUrl: subUrl})
				mihomoProxiesMutex.Unlock()
			}
		}
	}
	log.Debug("Exiting ParseYamlProxy for subUrl: %s", subUrl)
	return nil
}
