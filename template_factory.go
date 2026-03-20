package liquid

// TemplateFactory creates a fresh Template for the given template name.
// The default implementation ignores the name and returns NewTemplate().
// Replace the "template_factory" register to customize template instantiation.
type TemplateFactory func(templateName string) *Template

// defaultTemplateFactory is the factory used when none is registered.
func defaultTemplateFactory(_ string) *Template { return NewTemplate() }
