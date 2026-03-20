package liquid

type TemplateFactory struct{}

func NewTemplateFactory() *TemplateFactory {
	return &TemplateFactory{}
}

func (tf *TemplateFactory) For(templateName string) *Template {
	return NewTemplate()
}
