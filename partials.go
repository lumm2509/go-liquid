package liquid

import (
	"fmt"
	"strings"
)

func loadPartial(templateName string, context *Context, parseContext *ParseContext) (*Template, error) {
	cachedPartials, ok := context.Registers.Get("cached_partials").(map[string]interface{})
	if !ok {
		cachedPartials = make(map[string]interface{})
		context.Registers.Set("cached_partials", cachedPartials)
	}

	cacheKey := fmt.Sprintf("%s:%s", templateName, parseContext.ErrorMode)
	if cached, found := cachedPartials[cacheKey]; found {
		if t, ok := cached.(*Template); ok {
			return t, nil
		}
	}

	fileSystem, ok := context.Registers.Get("file_system").(FileSystem)
	if !ok {
		return nil, &FileSystemError{BaseError: BaseError{Message: "no file system configured; set environment.FileSystem"}}
	}
	source, err := fileSystem.ReadTemplateFile(templateName)
	if err != nil {
		// Try fallback to snippets/ if not present
		if !strings.HasPrefix(templateName, "snippets/") {
			fallbackName := "snippets/" + templateName
			source, err = fileSystem.ReadTemplateFile(fallbackName)
		}

		if err != nil {
			return nil, err
		}
	}
	parseContext.SetPartial(true)
	defer parseContext.SetPartial(false)

	templateFactory, ok := context.Registers.Get("template_factory").(TemplateFactory)
	if !ok {
		return nil, &InternalError{BaseError: BaseError{Message: "template_factory not registered"}}
	}
	template := templateFactory(templateName)
	template.Environment = context.Environment

	_, err = template.Parse(source, parseContext.options)
	if err != nil {
		return nil, err
	}

	template.Name = templateName
	cachedPartials[cacheKey] = template

	return template, nil
}
