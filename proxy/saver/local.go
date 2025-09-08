package saver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bestruirui/bestsub/utils"
)

const (
	outputDirName = "output"
	fileMode      = 0644
	dirMode       = 0755
)

type LocalSaver struct {
	basePath   string
	outputPath string
}

func NewLocalSaver() (*LocalSaver, error) {
	basePath := utils.GetExecutablePath()
	if basePath == "" {
		return nil, fmt.Errorf("get executable path failed")
	}

	outputPath := filepath.Join(basePath, outputDirName)
	return &LocalSaver{
		basePath:   basePath,
		outputPath: outputPath,
	}, nil
}

func SaveToLocal(yamlData []byte, filename string) error {
	saver, err := NewLocalSaver()
	if err != nil {
		return fmt.Errorf("create local saver failed: %w", err)
	}

	return saver.Save(yamlData, filename)
}

func (ls *LocalSaver) Save(yamlData []byte, filename string) error {
	if err := ls.ensureOutputDir(); err != nil {
		return fmt.Errorf("create output directory failed: %w", err)
	}

	if err := ls.validateInput(yamlData, filename); err != nil {
		return err
	}

	filePath := filepath.Join(ls.outputPath, filename)

	// if file exists, rename it to filename_old.yaml
	if _, err := os.Stat(filePath); err == nil {
		if err := os.Rename(filePath, ls.outputPath+string(os.PathSeparator)+filename+"_old.yaml"); err != nil {
			return fmt.Errorf("rename old file failed: %w", err)
		}
	}

	if err := os.WriteFile(filePath, yamlData, fileMode); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

func (ls *LocalSaver) ensureOutputDir() error {
	if _, err := os.Stat(ls.outputPath); os.IsNotExist(err) {
		if err := os.MkdirAll(ls.outputPath, dirMode); err != nil {
			return fmt.Errorf("create directory failed: %w", err)
		}
	}
	return nil
}

func (ls *LocalSaver) validateInput(yamlData []byte, filename string) error {
	if len(yamlData) == 0 {
		return fmt.Errorf("yaml data is empty")
	}

	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	if filepath.Base(filename) != filename {
		return fmt.Errorf("filename contains illegal characters: %s", filename)
	}

	return nil
}
