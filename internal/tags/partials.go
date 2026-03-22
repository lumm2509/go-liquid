package tags

import (
	"fmt"
	"strings"

	"github.com/go-liquid/internal/engine"
)

// PartialTemplate is a parsed partial (include/render target).
type PartialTemplate struct {
	Name string
	Root *engine.Document
}

func loadPartial(templateName string, ctx engine.RenderContext, parseContext *engine.ParseContext) (*PartialTemplate, error) {
	cachedPartials, ok := ctx.RegisterGet("cached_partials").(map[string]*PartialTemplate)
	if !ok {
		cachedPartials = make(map[string]*PartialTemplate)
		ctx.RegisterSet("cached_partials", cachedPartials)
	}

	cacheKey := fmt.Sprintf("%s:%s", templateName, parseContext.ErrorMode)
	if cached, found := cachedPartials[cacheKey]; found {
		return cached, nil
	}

	fs, ok := ctx.RegisterGet("file_system").(engine.FileSystem)
	if !ok {
		return nil, &engine.FileSystemError{BaseError: engine.BaseError{Message: "no file system configured; set environment.FileSystem"}}
	}

	source, err := fs.ReadTemplateFile(templateName)
	if err != nil {
		if !strings.HasPrefix(templateName, "snippets/") {
			source, err = fs.ReadTemplateFile("snippets/" + templateName)
		}
		if err != nil {
			return nil, err
		}
	}

	parseContext.SetPartial(true)
	defer parseContext.SetPartial(false)

	tokenizer := parseContext.NewTokenizer(source, 1, false)
	doc, err := engine.ParseDocument(tokenizer, parseContext)
	if err != nil {
		return nil, err
	}

	partial := &PartialTemplate{Name: templateName, Root: doc}
	cachedPartials[cacheKey] = partial
	return partial, nil
}
