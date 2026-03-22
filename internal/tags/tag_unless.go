package tags

import (
	"strings"

	"github.com/lumm2509/go-liquid/internal/engine"
)

type Unless struct {
	If
}

func NewUnless(tagName string, markup string, parseContext engine.TagParseContext) (engine.Tag, error) {
	ifTag, err := NewIf(tagName, markup, parseContext)
	if err != nil {
		return nil, err
	}
	return &Unless{If: *(ifTag.(*If))}, nil
}

func (u *Unless) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "endunless" {
		return false, nil
	}
	return u.If.UnknownTag(tag, markup)
}

func (u *Unless) RenderToOutputBuffer(ctx engine.RenderContext, output *strings.Builder) error {
	c := asContext(ctx)
	firstCondition := u.Conditions[0]
	if !firstCondition.Evaluate(c) {
		return firstCondition.Attachment.RenderToOutputBuffer(ctx, output)
	}
	for _, condition := range u.Conditions[1:] {
		if condition.IsElse() || condition.Evaluate(c) {
			return condition.Attachment.RenderToOutputBuffer(ctx, output)
		}
	}
	return nil
}
