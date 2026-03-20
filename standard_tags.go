package liquid

var StandardTags = map[string]TagFactory{
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
