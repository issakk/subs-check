package proxy

import (
	"os"
	"strings"
	"sync"

	"github.com/bestruirui/bestsub/config"
	"github.com/bestruirui/bestsub/utils/log"
)

// subURLFailThreshold is the number of failed attempts after which a
// subscription link is commented out of the config file.
const subURLFailThreshold = 5

var (
	subURLFailMutex  sync.Mutex
	subURLFailCounts = make(map[string]int)
	configFileMutex  sync.Mutex
)

// recordSubURLFailure counts a failed fetch for a subscription link (keyed by
// the raw url from the config file). Counts are cumulative for the lifetime of
// the process; once a link reaches subURLFailThreshold it is commented out of
// the config file so later runs stop requesting it.
func recordSubURLFailure(subUrl string) {
	subURLFailMutex.Lock()
	subURLFailCounts[subUrl]++
	failCount := subURLFailCounts[subUrl]
	subURLFailMutex.Unlock()

	log.Warn("subscription link [%s] failed to fetch, failure count: %d/%d", subUrl, failCount, subURLFailThreshold)
	if failCount >= subURLFailThreshold {
		commentOutSubURL(subUrl, failCount)
	}
}

func commentOutSubURL(subUrl string, failCount int) {
	configFileMutex.Lock()
	defer configFileMutex.Unlock()

	// Reset the counter so a manually re-enabled link starts counting fresh.
	subURLFailMutex.Lock()
	delete(subURLFailCounts, subUrl)
	subURLFailMutex.Unlock()

	configPath := config.ConfigPath
	if configPath == "" {
		log.Error("config path is empty, cannot comment out subscription link [%s]", subUrl)
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Error("read config file failed: %v", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	lineIndex := findSubURLLine(lines, subUrl)
	if lineIndex < 0 {
		log.Warn("subscription link [%s] not found in config file, skip commenting out", subUrl)
		return
	}

	line := lines[lineIndex]
	indent := len(line) - len(strings.TrimLeft(line, " \t"))
	lines[lineIndex] = line[:indent] + "# " + line[indent:]

	fileMode := os.FileMode(0644)
	if stat, err := os.Stat(configPath); err == nil {
		fileMode = stat.Mode()
	}
	// Written in place (not via rename) so the fsnotify watcher on the file
	// keeps working and the new url list is picked up by the next run.
	if err := os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), fileMode); err != nil {
		log.Error("write config file failed: %v", err)
		return
	}

	log.Info("subscription link [%s] failed %d times, commented out in %s", subUrl, failCount, configPath)
}

func findSubURLLine(lines []string, subUrl string) int {
	for i, line := range lines {
		if item, ok := subURLListItem(line); ok && item == subUrl {
			return i
		}
	}
	// Fallback for items with a trailing comment: "- url # some note".
	for i, line := range lines {
		item, ok := subURLListItem(line)
		if !ok {
			continue
		}
		if idx := strings.Index(item, " #"); idx != -1 && strings.TrimSpace(item[:idx]) == subUrl {
			return i
		}
	}
	return -1
}

// subURLListItem returns the value of a block-style yaml list item line
// ("  - value") with quotes stripped.
func subURLListItem(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "-") {
		return "", false
	}
	item := strings.TrimSpace(trimmed[1:])
	item = strings.Trim(item, `"'`)
	if item == "" {
		return "", false
	}
	return item, true
}
