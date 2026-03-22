package tags

import (
	"fmt"
	"strings"

	"github.com/go-liquid/internal/engine"
)

// loadPartialAtParseTime resolves and caches a partial using the Environment FileSystem.
// Returns nil, nil when no FileSystem is configured (skips preloading silently).
// Errors on missing or broken files so callers fail at parse time, not render time.
func loadPartialAtParseTime(templateName string, parseContext engine.TagParseContext) (*engine.ParsedPartial, error) {
	cacheKey := templateName + ":" + parseContext.GetErrorMode()
	if cached, ok := parseContext.GetParsedPartial(cacheKey); ok {
		return cached, nil
	}

	pc, ok := parseContext.(*engine.ParseContext)
	if !ok {
		// custom TagParseContext — cannot preload, but not a caller error
		return nil, nil
	}
	if pc.Environment == nil {
		return nil, fmt.Errorf("loadPartialAtParseTime: environment is nil — cannot preload partial %q", templateName)
	}
	fs := pc.Environment.GetFileSystem()
	if fs == nil {
		return nil, fmt.Errorf("loadPartialAtParseTime: no FileSystem configured in environment — cannot preload partial %q", templateName)
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
	doc, err := engine.ParseDocument(tokenizer, pc)
	if err != nil {
		return nil, err
	}

	partial := &engine.ParsedPartial{Name: templateName, Root: doc}
	parseContext.SetParsedPartial(cacheKey, partial)
	return partial, nil
}

// loadPartial resolves and returns a parsed partial, cached per unique name so the
// FileSystem and parser are invoked at most once across all renders
func loadPartial(templateName string, ctx engine.RenderContext, parseContext engine.TagParseContext) (*engine.ParsedPartial, error) {
	cacheKey := templateName + ":" + parseContext.GetErrorMode()

	if cached, ok := parseContext.GetParsedPartial(cacheKey); ok {
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
	pc, ok := parseContext.(*engine.ParseContext)
	if !ok {
		return nil, fmt.Errorf("loadPartial: unsupported TagParseContext implementation")
	}
	doc, err := engine.ParseDocument(tokenizer, pc)
	if err != nil {
		return nil, err
	}

	partial := &engine.ParsedPartial{Name: templateName, Root: doc}
	parseContext.SetParsedPartial(cacheKey, partial)
	return partial, nil
}
