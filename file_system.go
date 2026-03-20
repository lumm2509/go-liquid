package liquid

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystem interface {
	ReadTemplateFile(templatePath string) (string, error)
}

type BlankFileSystem struct{}

func (f *BlankFileSystem) ReadTemplateFile(templatePath string) (string, error) {
	return "", fmt.Errorf("This liquid context does not allow includes.")
}

type LocalFileSystem struct {
	Root    string
	Pattern string
}

func NewLocalFileSystem(root string, pattern string) *LocalFileSystem {
	if pattern == "" {
		pattern = "_%s.liquid"
	}
	return &LocalFileSystem{Root: root, Pattern: pattern}
}

func (f *LocalFileSystem) ReadTemplateFile(templatePath string) (string, error) {
	fullPath, err := f.FullPath(templatePath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("No such template '%s'", templatePath)
	}
	return string(data), nil
}

func (f *LocalFileSystem) FullPath(templatePath string) (string, error) {
	baseName := filepath.Base(templatePath)
	dirName := filepath.Dir(templatePath)

	fileName := fmt.Sprintf(f.Pattern, baseName)
	fullPath := filepath.Join(f.Root, dirName, fileName)

	// filepath.Join+Abs normalizes ".." sequences; the prefix check is the real guard.
	absRoot, _ := filepath.Abs(f.Root)
	absPath, _ := filepath.Abs(fullPath)

	if !strings.HasPrefix(absPath, absRoot) {
		return "", fmt.Errorf("Illegal template path '%s'", templatePath)
	}

	return absPath, nil
}
