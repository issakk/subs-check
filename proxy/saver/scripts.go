package saver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/proxy/info"
	"github.com/bestruirui/bestsub/utils/log"
)

const scriptTimeout = 5 * time.Minute

func ExecuteScripts(scripts []string) error {
	if len(scripts) == 0 {
		return nil
	}

	var errs []error
	for _, scriptPath := range scripts {
		if err := executeScript(scriptPath); err != nil {
			log.Error("Failed to execute script %s: %v", scriptPath, err)
			errs = append(errs, fmt.Errorf("%s: %w", scriptPath, err))
		}
	}

	return errors.Join(errs...)
}

func executeScript(scriptPath string) error {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("script file not found: %s", scriptPath)
	}

	ext := strings.ToLower(filepath.Ext(scriptPath))
	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()

	var cmd *exec.Cmd
	switch ext {
	case ".js":
		cmd = exec.CommandContext(ctx, "node", scriptPath)
	case ".py":
		cmd = exec.CommandContext(ctx, "python", scriptPath)
	case ".sh":
		cmd = exec.CommandContext(ctx, "sh", scriptPath)
	case ".bat":
		cmd = exec.CommandContext(ctx, "cmd", "/c", scriptPath)
	case ".ps1":
		cmd = exec.CommandContext(ctx, "powershell", "-File", scriptPath)
	default:
		cmd = exec.CommandContext(ctx, scriptPath)
	}

	logFilePath := scriptPath + ".log"
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	log.Info("Executing script: %s", scriptPath)
	log.Info("Logging output to: %s", logFilePath)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("script execution timed out after %s", scriptTimeout)
		}
		return err
	}
	return nil
}

func buildScriptProxyPayload(results *[]info.Proxy) []map[string]any {
	rawProxies := make([]map[string]any, 0, len(*results))
	for i := range *results {
		proxyData := make(map[string]any, len((*results)[i].Raw)+6)
		for key, value := range (*results)[i].Raw {
			proxyData[key] = value
		}
		proxyData["country"] = (*results)[i].Info.Country
		proxyData["speed"] = (*results)[i].Info.Speed
		proxyData["disney"] = (*results)[i].Info.Unlock.Disney
		proxyData["youtube"] = (*results)[i].Info.Unlock.Youtube
		proxyData["netflix"] = (*results)[i].Info.Unlock.Netflix
		proxyData["chatgpt"] = (*results)[i].Info.Unlock.Chatgpt
		rawProxies = append(rawProxies, proxyData)
	}
	return rawProxies
}

func BeforeSaveDo(results *[]info.Proxy) error {
	log.Info("Executing before-save scripts")

	rawProxies := buildScriptProxyPayload(results)
	jsonData, err := json.MarshalIndent(map[string]any{
		"proxies": rawProxies,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize proxies failed: %w", err)
	}

	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "bestsub_temp_proxies.json")
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("save proxies to temp file failed: %w", err)
	}

	log.Debug("Proxies saved to temp file: %s", tempFile)

	execErr := ExecuteScripts(config.GlobalConfig.Save.BeforeSaveDo)

	if config.GlobalConfig.LogLevel == "debug" {
		log.Debug("Debug mode, not removing temp file: %s", tempFile)
	} else {
		if err := os.Remove(tempFile); err != nil {
			return fmt.Errorf("remove temp file failed: %w", err)
		}
		log.Debug("Removed temp file: %s", tempFile)
	}

	if execErr != nil {
		return execErr
	}
	return nil
}

func AfterSaveDo(results *[]info.Proxy) error {
	log.Info("Executing after-save scripts")
	return ExecuteScripts(config.GlobalConfig.Save.AfterSaveDo)
}
