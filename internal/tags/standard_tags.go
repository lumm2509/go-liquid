package tags

import "github.com/go-liquid/internal/engine"

// StandardTags is the default set of tags registered in a new Environment.
var StandardTags = map[string]engine.TagFactory{
	"assign":    NewAssign,
	"for":       NewFor,
	"if":        NewIf,
	"echo":      NewEcho,
	"unless":    NewUnless,
	"comment":   NewComment,
	"raw":       NewRaw,
	"render":    NewRender,
	"include":   NewInclude,
	"case":      NewCase,
	"cycle":     NewCycle,
	"tablerow":  NewTableRow,
	"increment": NewIncrement,
	"decrement": NewDecrement,
	"doc":       NewDoc,
	"#":         NewInlineComment,
	"ifchanged": NewIfchanged,
	"break":     NewBreak,
	"continue":  NewContinue,
	"capture":   NewCapture,
}
