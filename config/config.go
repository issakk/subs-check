package config

type ProxyConfig struct {
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
}
type RenameConfig struct {
	Method string `yaml:"method"`
	Flag   bool   `yaml:"flag"`
}
type SaveConfig struct {
	Method          string `yaml:"method"`
	Port            int    `yaml:"port"`
	WebDAVURL       string `yaml:"webdav-url"`
	WebDAVUsername  string `yaml:"webdav-username"`
	WebDAVPassword  string `yaml:"webdav-password"`
	GithubToken     string `yaml:"github-token"`
	GithubGistID    string `yaml:"github-gist-id"`
	GithubAPIMirror string `yaml:"github-api-mirror"`
	WorkerURL       string `yaml:"worker-url"`
	WorkerToken     string `yaml:"worker-token"`
}
type CheckConfig struct {
	Concurrent           int      `yaml:"concurrent"`
	Items                []string `yaml:"items"`
	Interval             int      `yaml:"interval"`
	Timeout              int      `yaml:"timeout"`
	MinSpeed             int      `yaml:"min-speed"`
	QualityLevel         int      `yaml:"quality-level"`
	DownloadTimeout      int      `yaml:"download-timeout"`
	SpeedTestUrl         string   `yaml:"speed-test-url"`
	SpeedSkipName        string   `yaml:"speed-skip-name"`
	SpeedCheckConcurrent int      `yaml:"speed-check-concurrent"`
}
type Config struct {
	Check           CheckConfig  `yaml:"check"`
	PrintProgress   bool         `yaml:"print-progress"`
	Save            SaveConfig   `yaml:"save"`
	SubUrlsReTry    int          `yaml:"sub-urls-retry"`
	SubUrls         []string     `yaml:"sub-urls"`
	MihomoApiUrl    string       `yaml:"mihomo-api-url"`
	MihomoApiSecret string       `yaml:"mihomo-api-secret"`
	Proxy           ProxyConfig  `yaml:"proxy"`
	Rename          RenameConfig `yaml:"rename"`
	LogLevel        string       `yaml:"log-level"`
	WeworkBot       string       `yaml:"wework-bot"` // 新增企业微信机器人webhook地址
}

var GlobalConfig Config
