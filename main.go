package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/proxy"
	"github.com/bestruirui/bestsub/proxy/checker"
	"github.com/bestruirui/bestsub/proxy/info"
	"github.com/bestruirui/bestsub/proxy/saver"
	"github.com/bestruirui/bestsub/utils"
	"github.com/bestruirui/bestsub/utils/log"
	"github.com/fsnotify/fsnotify"
	mihomoLog "github.com/metacubex/mihomo/log"
	"github.com/panjf2000/ants/v2"
	"gopkg.in/yaml.v3"
)

type App struct {
	renamePath  string
	configPath  string
	interval    int
	watcher     *fsnotify.Watcher
	reloadTimer *time.Timer
}

func NewApp() *App {
	configPath := flag.String("f", "", "config file path")
	renamePath := flag.String("r", "", "rename file path")
	flag.Parse()

	return &App{
		configPath: *configPath,
		renamePath: *renamePath,
	}
}

func (app *App) Initialize() error {

	if err := app.initConfigPath(); err != nil {
		return fmt.Errorf("init config path failed: %w", err)

	}

	if err := app.loadConfig(); err != nil {
		return fmt.Errorf("load config failed: %w", err)
	}
	if config.GlobalConfig.LogLevel != "" {
		log.SetLogLevel(config.GlobalConfig.LogLevel)
	} else {
		log.SetLogLevel("info")
	}

	checkConfig()

	if err := app.initConfigWatcher(); err != nil {
		return fmt.Errorf("init config watcher failed: %w", err)
	}

	app.interval = config.GlobalConfig.Check.Interval
	mihomoLog.SetLevel(mihomoLog.ERROR)
	if config.GlobalConfig.Save.Method == "http" {
		saver.StartHTTPServer()
	}
	return nil
}

func (app *App) initConfigPath() error {
	execPath := utils.GetExecutablePath()
	configDir := filepath.Join(execPath, "config")

	if app.configPath == "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("create config dir failed: %w", err)
		}

		app.configPath = filepath.Join(configDir, "config.yaml")
	}
	if app.renamePath == "" {
		app.renamePath = filepath.Join(configDir, "rename.yaml")
	}
	return nil
}

func (app *App) loadConfig() error {
	yamlFile, err := os.ReadFile(app.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Error("config file not found: %v", err)
			log.Info("Please refer to the docs to create config file: %v", app.configPath)
			log.Info("Docs: https://github.com/bestruirui/BestSub/tree/master/doc")
			os.Exit(1)
		}
		return fmt.Errorf("read config file failed: %w", err)
	}

	if err := yaml.Unmarshal(yamlFile, &config.GlobalConfig); err != nil {
		return fmt.Errorf("parse config file failed: %w", err)
	}

	info.CountryCodeRegexInit(app.renamePath)

	return nil
}

func (app *App) initConfigWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher failed: %w", err)
	}

	app.watcher = watcher
	app.reloadTimer = time.NewTimer(0)
	<-app.reloadTimer.C

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if app.reloadTimer != nil {
						app.reloadTimer.Stop()
					}
					app.reloadTimer.Reset(100 * time.Millisecond)

					go func() {
						<-app.reloadTimer.C
						log.Info("config file changed, reloading")
						if err := app.loadConfig(); err != nil {
							log.Error("reload config file failed: %v", err)
							return
						}
						app.interval = config.GlobalConfig.Check.Interval
					}()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error("config file watcher error: %v", err)
			}
		}
	}()

	if err := watcher.Add(app.configPath); err != nil {
		return fmt.Errorf("add config file watcher failed: %w", err)
	}

	log.Info("config file watcher started")
	return nil
}

func (app *App) Run() {
	defer func() {
		app.watcher.Close()
		if app.reloadTimer != nil {
			app.reloadTimer.Stop()
		}
	}()

	for {
		maintask()
		utils.UpdateSubs()
		nextCheck := time.Now().Add(time.Duration(app.interval) * time.Minute)
		log.Info("next check time: %v", nextCheck.Format("2006-01-02 15:04:05"))
		time.Sleep(time.Duration(app.interval) * time.Minute)
	}
}

func main() {

	app := NewApp()

	if err := app.Initialize(); err != nil {
		log.Error("initialize failed: %v", err)
		os.Exit(1)
	}

	app.Run()
}
func maintask() {
	proxies := make([]info.Proxy, 0)

	proxy.GetProxies(&proxies)

	log.Info("get proxies success: %v proxies", len(proxies))

	info.DeduplicateProxies(&proxies)

	log.Info("deduplicate proxies: %v proxies", len(proxies))

	var wg sync.WaitGroup

	pool, _ := ants.NewPool(config.GlobalConfig.Check.Concurrent)

	for i := range proxies {
		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			proxyCheckTask(&proxies[i])
		})
	}

	wg.Wait()
	var aliveCount int = 0

	for i := range proxies {
		if proxies[i].Info.Alive {
			proxies[i].Id = aliveCount
			aliveCount++
		}
	}

	log.Info("check end %v proxies", aliveCount)

	// 添加处理完成通知
	message := fmt.Sprintf("订阅检测处理完成!\n共处理节点数量: %v", aliveCount)
	if err := utils.SendWeworkNotification(message); err != nil {
		log.Error("发送企业微信通知失败: %v", err)
	}
	if utils.Contains(config.GlobalConfig.Check.Items, "speed") {
		log.Info("start speed test")
		pool.Tune(config.GlobalConfig.Check.SpeedCheckConcurrent)
		for i := range proxies {
			if proxies[i].Info.Alive {
				wg.Add(1)
				pool.Submit(func() {
					defer wg.Done()
					proxySpeedTask(&proxies[i])
				})
			}
		}
		wg.Wait()
		log.Info("end speed test")
	}

	log.Info("start rename proxies")
	pool.Tune(config.GlobalConfig.Check.Concurrent)
	for i := range proxies {
		if proxies[i].Info.Alive {
			wg.Add(1)
			pool.Submit(func() {
				defer wg.Done()
				proxyRenameTask(&proxies[i])
			})
		}
	}
	wg.Wait()

	log.Info("end rename proxies")

	saver.SaveConfig(&proxies)

	proxies = nil

	pool.Release()

	runtime.GC()
}

func proxyCheckTask(proxy *info.Proxy) {
	if proxy.New() != nil {
		return
	}
	defer proxy.Close()
	checker := checker.NewChecker(proxy)
	defer checker.Close()
	for i := 0; i < 3; i++ {
		checker.AliveTest("https://gstatic.com/generate_204", 204)
		if proxy.Info.Alive {
			break
		}
	}
	if proxy.Info.Alive {
		for _, item := range config.GlobalConfig.Check.Items {
			switch item {
			case "openai":
				checker.OpenaiTest()
			case "youtube":
				checker.YoutubeTest()
			case "netflix":
				checker.NetflixTest()
			case "disney":
				checker.DisneyTest()
			}
		}
	}
}
func proxySpeedTask(proxy *info.Proxy) {
	if proxy.New() != nil {
		return
	}
	defer proxy.Close()
	checker := checker.NewChecker(proxy)
	defer checker.Close()
	checker.CheckSpeed()
}
func proxyRenameTask(proxy *info.Proxy) {
	if proxy.New() != nil {
		return
	}
	defer proxy.Close()
	switch config.GlobalConfig.Rename.Method {
	case "api":
		proxy.CountryCodeFromApi()
	case "regex":
		proxy.CountryCodeRegex()
	case "mix":
		proxy.CountryCodeRegex()
		if proxy.Info.Country == "UN" {
			proxy.CountryCodeFromApi()
		}
	}

	name := fmt.Sprintf("%v %03d", proxy.Info.Country, proxy.Id)
	if config.GlobalConfig.Rename.Flag {
		proxy.CountryFlag()
		name = fmt.Sprintf("%v %v", proxy.Info.Flag, name)
	}

	proxy.ParseRate()

	if proxy.Info.Rate != 0 {
		name = fmt.Sprintf("%v x%.2f", name, proxy.Info.Rate)
	}

	if utils.Contains(config.GlobalConfig.Check.Items, "speed") && !proxy.Info.SpeedSkip {
		speed := proxy.Info.Speed
		var speedStr string
		switch {
		case speed < 1024:
			speedStr = fmt.Sprintf("%d KB/s", speed)
		case speed < 1024*1024:
			speedStr = fmt.Sprintf("%.2f MB/s", float64(speed)/1024)
		default:
			speedStr = fmt.Sprintf("%.2f GB/s", float64(speed)/(1024*1024))
		}
		name = fmt.Sprintf("%v | ⬇️ %s", name, speedStr)
	}

	proxy.Raw["name"] = name
}

var version string

func checkConfig() {

	log.Info("bestsub version: %v", version)

	if config.GlobalConfig.Check.Concurrent <= 0 {
		log.Error("concurrent must be greater than 0")
		os.Exit(1)
	}
	log.Info("concurrents: %v", config.GlobalConfig.Check.Concurrent)
	switch config.GlobalConfig.Save.Method {
	case "webdav":
		if config.GlobalConfig.Save.WebDAVURL == "" {
			log.Error("webdav-url is required when save-method is webdav")
			os.Exit(1)
		} else {
			log.Info("save method: webdav")
		}
	case "http":
		if config.GlobalConfig.Save.Port <= 0 {
			log.Error("port must be greater than 0 when save-method is http")
			os.Exit(1)
		} else {
			log.Info("save method: http")
		}
	case "gist":
		if config.GlobalConfig.Save.GithubGistID == "" {
			log.Error("github-gist-id is required when save-method is gist")
			os.Exit(1)
		}
		if config.GlobalConfig.Save.GithubToken == "" {
			log.Error("github-token is required when save-method is gist")
			os.Exit(1)
		}
		log.Info("save method: gist")
	}
	if config.GlobalConfig.SubUrls == nil {
		log.Error("sub-urls is required")
		os.Exit(1)
	}
	switch config.GlobalConfig.Rename.Method {
	case "api":
		log.Info("rename method: api")
	case "regex":
		log.Info("rename method: regex")
	case "mix":
		log.Info("rename method: mix")
	default:
		log.Error("rename-method must be one of api, regex, mix")
		os.Exit(1)
	}
	if config.GlobalConfig.Proxy.Type == "http" {
		log.Info("proxy type: http")
	} else if config.GlobalConfig.Proxy.Type == "socks" {
		log.Info("proxy type: socks")
	} else {
		log.Info("not use proxy")
	}
	log.Info("progress display: %v", config.GlobalConfig.PrintProgress)
	if config.GlobalConfig.Check.Interval < 10 {
		log.Error("check-interval must be greater than 10 minutes")
		os.Exit(1)
	}
	if len(config.GlobalConfig.Check.Items) == 0 {
		log.Info("check items: none")
	} else {
		log.Info("check items: %v", config.GlobalConfig.Check.Items)
		if utils.Contains(config.GlobalConfig.Check.Items, "speed") {
			log.Info("speed check concurrent: %v", config.GlobalConfig.Check.SpeedCheckConcurrent)
		}
	}
	if config.GlobalConfig.MihomoApiUrl != "" {
		version, err := utils.GetVersion()
		if err != nil {
			log.Error("get version failed: %v", err)
		} else {
			log.Info("auto update provider: true")
			log.Info("mihomo version: %v", version)
		}
	}
}
