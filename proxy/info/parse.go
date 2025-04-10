package info

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bestruirui/bestsub/utils/log"
	"github.com/dlclark/regexp2"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func (p *Proxy) CountryCodeFromApi() {
	ctx, cancel := context.WithCancel(p.Ctx)
	defer cancel()

	apis := []string{
		"https://api.ip.sb/geoip",
		"https://ipapi.co/json",
		"https://ip.seeip.org/geoip",
		"https://api.myip.com",
	}
	var countryCode string

	for _, api := range apis {
		for attempts := 0; attempts < 5; attempts++ {
			req, err := http.NewRequestWithContext(ctx, "GET", api, nil)
			if err != nil {
				time.Sleep(time.Second * time.Duration(attempts))
				continue
			}

			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Pragma", "no-cache")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
			req.Header.Set("Sec-Ch-Ua", `"Not A(Brand";v="8", "Chromium";v="132", "Google Chrome";v="132"`)
			req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
			req.Header.Set("Sec-Ch-Ua-Platform", "Windows")
			req.Header.Set("Sec-Fetch-Dest", "document")
			req.Header.Set("Sec-Fetch-Mode", "navigate")
			req.Header.Set("Sec-Fetch-Site", "none")
			req.Header.Set("Sec-Fetch-User", "?1")
			req.Header.Set("Upgrade-Insecure-Requests", "1")

			resp, err := p.Client.Do(req)
			if err != nil {
				time.Sleep(time.Second * time.Duration(attempts))
				continue
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				time.Sleep(time.Second * time.Duration(attempts))
				continue
			}

			ipinfo := map[string]any{}
			err = json.Unmarshal(body, &ipinfo)
			if err != nil {
				time.Sleep(time.Second * time.Duration(attempts))
				continue
			}

			ok := false
			switch api {
			case "https://api.ip.sb/geoip":
				if code, exists := ipinfo["country_code"].(string); exists {
					countryCode = code
					ok = true
				}
			case "https://ipapi.co/json":
				if code, exists := ipinfo["country_code"].(string); exists {
					countryCode = code
					ok = true
				}
			case "https://ip.seeip.org/geoip":
				if code, exists := ipinfo["country_code"].(string); exists {
					countryCode = code
					ok = true
				}
			case "https://api.myip.com":
				if code, exists := ipinfo["cc"].(string); exists {
					countryCode = code
					ok = true
				}
			}

			if ok && countryCode != "" {
				break
			}
		}
	}
	if len(countryCode) == 0 {
		p.Info.Country = "UN"
	} else {
		p.Info.Country = countryCode
	}
}
func getFlag(countryCode string) string {
	code := strings.ToUpper(countryCode)
	if len(code) != 2 {
		code = "UN"
	}
	const flagBase = 127397

	first := string(rune(code[0]) + flagBase)
	second := string(rune(code[1]) + flagBase)

	return first + second
}
func (p *Proxy) CountryFlag() {
	p.Info.Flag = getFlag(p.Info.Country)
}

type Country struct {
	Name        string `yaml:"name"`
	Recognition string `yaml:"recognition"`
}

var CountryCodeRegex []Country

func CountryCodeRegexInit(renamePath string) {
	data, err := os.ReadFile(renamePath)
	if err != nil {
		log.Error("read rename file failed: %v", err)
		log.Info("please download rename file from https://github.com/bestruirui/BestSub/tree/master/doc/rename.yaml")
		os.Exit(1)
	}

	err = yaml.Unmarshal(data, &CountryCodeRegex)
	if err != nil {
		log.Error("parse rename file failed: %v", err)
		log.Info("please download rename file from https://github.com/bestruirui/BestSub/tree/master/doc/rename.yaml")
		os.Exit(1)
	}
}

func (p *Proxy) CountryCodeRegex() {
	for _, country := range CountryCodeRegex {
		re := regexp2.MustCompile(country.Recognition, regexp2.None)
		match, err := re.MatchString(cast.ToString(p.Raw["name"]))
		if err != nil {
			fmt.Printf("Regex match error: %v\n", err)
			continue
		}
		if match {
			p.Info.Country = country.Name
			return
		}
	}
	p.Info.Country = "UN"
}

var rateRegex = regexp2.MustCompile(`(?:倍率|x(?<value>\d+(?:\.\d+)?))`, regexp2.None)

func (p *Proxy) ParseRate() {
	log.Debug("parse proxy %s rate", p.Raw)
	match, err := rateRegex.FindStringMatch(cast.ToString(p.Raw["name"]))
	if err != nil {
		log.Debug("parse rate failed: %v", err)
		return
	}

	if match != nil {
		valueGroup := match.GroupByName("value")
		if valueGroup != nil && len(valueGroup.Captures) > 0 {
			rateValue := valueGroup.Captures[0].String()
			rate, err := strconv.ParseFloat(rateValue, 64)
			if err != nil {
				log.Debug("parse rate value failed: %v", err)
				return
			}
			p.Info.Rate = float32(rate)
			log.Debug("parse proxy %s rate value: %v", p.Raw["name"], p.Info.Rate)
		} else {
			log.Debug("Rate value not found")
		}
		return
	}
}
