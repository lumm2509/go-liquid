package liquid

import (
	"fmt"
)

type BaseError struct {
	Message       string
	LineNumber    int
	TemplateName  string
	MarkupContext string
	Cause         error
}

func (e BaseError) format(prefix string) string {
	var res string
	if e.LineNumber > 0 {
		templatePart := ""
		if e.TemplateName != "" {
			templatePart = e.TemplateName + " "
		}
		res = fmt.Sprintf("%s (%sline %d): ", prefix, templatePart, e.LineNumber)
	} else {
		res = prefix + ": "
	}
	res += e.Message
	if e.MarkupContext != "" {
		res += " " + e.MarkupContext
	}
	return res
}

func (e BaseError) Error() string {
	return e.format("Liquid error")
}

func (e BaseError) Unwrap() error {
	return e.Cause
}

type ArgumentError struct{ BaseError }
type ContextError struct{ BaseError }
type FileSystemError struct{ BaseError }
type StandardError struct{ BaseError }
type SyntaxError struct{ BaseError }
type StackLevelError struct{ BaseError }
type MemoryError struct{ BaseError }
type ZeroDivisionError struct{ BaseError }
type FloatDomainError struct{ BaseError }
type UndefinedVariable struct{ BaseError }
type UndefinedDropMethod struct{ BaseError }
type UndefinedFilter struct{ BaseError }
type MethodOverrideError struct{ BaseError }
type DisabledError struct {
	BaseError
	TagName string
}
type InternalError struct{ BaseError }
type TemplateEncodingError struct{ BaseError }

func (e SyntaxError) Error() string {
	return e.BaseError.format("Liquid syntax error")
}

func (e DisabledError) Error() string {
	if e.TagName != "" {
		return fmt.Sprintf("Tag '%s' is disabled", e.TagName)
	}
	return e.BaseError.Error()
}
