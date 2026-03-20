package liquid

import (
	"strings"
)

type Unless struct {
	If
}

func NewUnless(tagName string, markup string, parseContext TagParseContext) (Tag, error) {
	ifTag, err := NewIf(tagName, markup, parseContext)
	if err != nil {
		return nil, err
	}
	u := &Unless{
		If: *(ifTag.(*If)),
	}
	return u, nil
}

func (u *Unless) UnknownTag(tag string, markup string) (bool, error) {
	if tag == "endunless" {
		return false, nil
	}
	return u.If.UnknownTag(tag, markup)
}

func (u *Unless) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	// First condition is interpreted backwards
	firstCondition := u.Conditions[0]
	if !firstCondition.Evaluate(context) {
		return firstCondition.Attachment.RenderToOutputBuffer(context, output)
	}

	// After the first condition unless works just like if
	for _, condition := range u.Conditions[1:] {
		if condition.IsElse() || condition.Evaluate(context) {
			return condition.Attachment.RenderToOutputBuffer(context, output)
		}
	}

	return nil
}
