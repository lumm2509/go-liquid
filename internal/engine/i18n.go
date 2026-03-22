package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var interpolationRegex = regexp.MustCompile(`%\{(\w+)\}`)

type I18n struct {
	Path   string
	mu     sync.RWMutex
	locale map[string]interface{}
}

func NewI18n(path string) *I18n {
	if path == "" {
		path = "locales/en.yml"
	}
	return &I18n{Path: path}
}

func (i *I18n) Translate(name string, vars map[string]interface{}) (string, error) {
	translation, err := i.deepFetchTranslation(name)
	if err != nil {
		return "", err
	}
	return i.interpolate(translation, vars), nil
}

func (i *I18n) T(name string, vars map[string]interface{}) (string, error) {
	return i.Translate(name, vars)
}

func (i *I18n) LoadLocale() error {
	i.mu.RLock()
	loaded := i.locale != nil
	i.mu.RUnlock()
	if loaded {
		return nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.locale != nil {
		return nil
	}
	data, err := os.ReadFile(i.Path)
	if err != nil {
		return fmt.Errorf("could not read locale file: %w", err)
	}
	locale := make(map[string]interface{})
	if err = yaml.Unmarshal(data, &locale); err != nil {
		return fmt.Errorf("could not parse YAML locale: %w", err)
	}
	i.locale = locale
	return nil
}

func (i *I18n) interpolate(name string, vars map[string]interface{}) string {
	return interpolationRegex.ReplaceAllStringFunc(name, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "%{"), "}")
		if val, ok := vars[key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
}

func (i *I18n) deepFetchTranslation(name string) (string, error) {
	if err := i.LoadLocale(); err != nil {
		return "", err
	}
	keys := strings.Split(name, ".")
	var current interface{} = i.locale
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if next, found := m[key]; found {
				current = next
				continue
			}
		}
		return "", fmt.Errorf("Translation for %s does not exist in locale %s", name, i.Path)
	}
	result, ok := current.(string)
	if !ok {
		return "", fmt.Errorf("Translation for %s is not a string", name)
	}
	return result, nil
}
