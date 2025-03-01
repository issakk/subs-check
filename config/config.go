package config

// 在 Config 结构体中添加 WebhookURL 字段
type Config struct {
	PrintProgress   bool     `yaml:"print-progress"`
	Concurrent      int      `yaml:"concurrent"`
	CheckInterval   int      `yaml:"check-interval"`
	QualityLevel    int      `yaml:"quality-level"`
	SpeedTestUrl    string   `yaml:"speed-test-url"`
	DownloadTimeout int      `yaml:"download-timeout"`
	MinSpeed        int      `yaml:"min-speed"`
	Timeout         int      `yaml:"timeout"`
	FilterRegex     string   `yaml:"filter-regex"`
	SaveMethod      string   `yaml:"save-method"`
	WebDAVURL       string   `yaml:"webdav-url"`
	WebDAVUsername  string   `yaml:"webdav-username"`
	WebDAVPassword  string   `yaml:"webdav-password"`
	GithubToken     string   `yaml:"github-token"`
	GithubGistID    string   `yaml:"github-gist-id"`
	GithubAPIMirror string   `yaml:"github-api-mirror"`
	WorkerURL       string   `yaml:"worker-url"`
	WorkerToken     string   `yaml:"worker-token"`
	SubUrlsReTry    int      `yaml:"sub-urls-retry"`
	SubUrls         []string `yaml:"sub-urls"`
	MihomoApiUrl    string   `yaml:"mihomo-api-url"`
	MihomoApiSecret string   `yaml:"mihomo-api-secret"`
	WebhookURL       string   `yaml:"webhook-url"`
}

var GlobalConfig Config
