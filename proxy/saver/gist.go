package saver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/utils"
	"github.com/bestruirui/bestsub/utils/log"
)

var (
	gistAPIURL     = "https://api.github.com/gists"
	gistMaxRetries = 3
	gistRetryDelay = 2 * time.Second
)

type GistFile struct {
	Content string `json:"content"`
}

type GistPayload struct {
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]GistFile `json:"files"`
}

type GistUploader struct {
	client   *http.Client
	token    string
	id       string
	isPublic bool
}

func NewGistUploader() *GistUploader {
	if config.GlobalConfig.Save.GithubAPIMirror != "" {
		gistAPIURL = config.GlobalConfig.Save.GithubAPIMirror + "/gists"
	}

	return &GistUploader{
		client:   utils.NewHTTPClient(),
		token:    config.GlobalConfig.Save.GithubToken,
		id:       config.GlobalConfig.Save.GithubGistID,
		isPublic: false,
	}
}

func UploadToGist(yamlData []byte, filename string) error {
	uploader := NewGistUploader()
	return uploader.Upload(yamlData, filename)
}

func ValiGistConfig() error {
	if config.GlobalConfig.Save.GithubToken == "" {
		return fmt.Errorf("github token is not configured")
	}
	if config.GlobalConfig.Save.GithubGistID == "" {
		return fmt.Errorf("gist id is not configured")
	}
	return nil
}

func (g *GistUploader) Upload(yamlData []byte, filename string) error {
	if err := g.validateInput(yamlData, filename); err != nil {
		return err
	}

	payload := GistPayload{
		Description: "Configuration",
		Public:      g.isPublic,
		Files: map[string]GistFile{
			filename: {
				Content: string(yamlData),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}

	return g.uploadWithRetry(jsonData, filename)
}

func (g *GistUploader) validateInput(yamlData []byte, filename string) error {
	if len(yamlData) == 0 {
		return fmt.Errorf("yaml data is empty")
	}
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if g.token == "" {
		return fmt.Errorf("github token is not configured")
	}
	return nil
}

func (g *GistUploader) uploadWithRetry(jsonData []byte, filename string) error {
	var lastErr error

	for attempt := 0; attempt < gistMaxRetries; attempt++ {
		if err := g.doUpload(jsonData); err != nil {
			lastErr = err
			log.Error("gist upload failed(attempt %d/%d): %v", attempt+1, gistMaxRetries, err)
			time.Sleep(gistRetryDelay)
			continue
		}
		log.Info("gist upload success: %s", filename)
		return nil
	}

	return fmt.Errorf("gist upload failed, tried %d times: %w", gistMaxRetries, lastErr)
}

func (g *GistUploader) doUpload(jsonData []byte) error {
	req, err := g.createRequest(jsonData)
	if err != nil {
		return err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	return g.checkResponse(resp)
}

func (g *GistUploader) createRequest(jsonData []byte) (*http.Request, error) {
	url := gistAPIURL + "/" + g.id
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	return req, nil
}

func (g *GistUploader) checkResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response failed(status code: %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("upload failed(status code: %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
