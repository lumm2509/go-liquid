package liquid

import (
	"github.com/go-liquid/internal/engine"
	"github.com/go-liquid/internal/filters"
	"github.com/go-liquid/internal/tags"
)

// Type aliases — re-export internal engine types so that code using the
// root package (liquid.Context, liquid.Tag, etc.) keeps compiling unchanged.
type (
	// Runtime types
	Context          = engine.Context
	ParseContext     = engine.ParseContext
	Document         = engine.Document
	BlockBody        = engine.BlockBody
	Block            = engine.Block
	Condition        = engine.Condition
	Variable         = engine.Variable
	VariableLookup   = engine.VariableLookup
	RangeLookup      = engine.RangeLookup
	StrainerTemplate = engine.StrainerTemplate
	Strainer         = engine.Strainer
	I18n             = engine.I18n

	// Interface and function types
	ExceptionRenderer = engine.ExceptionRenderer
	DebugLogger       = engine.DebugLogger
	TagFactory        = engine.TagFactory
	TagParseContext   = engine.TagParseContext
	Tag               = engine.Tag
	Node              = engine.Node
	RenderContext     = engine.RenderContext
	EnvironmentIface  = engine.EnvironmentIface
	UnknownTagHandler = engine.UnknownTagHandler
)

// DebugEvent is re-exported as a concrete alias so existing code that
// constructs liquid.DebugEvent{} keeps working.
type DebugEvent = engine.DebugEvent

// TagBase re-exported so embedders can write engine.TagBase via liquid.TagBase.
type TagBase = engine.TagBase

// StringNode re-exported for packages that create raw text nodes.
type StringNode = engine.StringNode

// FileSystem is re-exported here; the concrete types (BlankFileSystem,
// LocalFileSystem) remain defined in file_system.go.
type FileSystem = engine.FileSystem

// StandardFilters is the standard Liquid filter set; re-exported for tests and custom environments.
type StandardFilters = filters.StandardFilters

// StandardTags is the default tag registry; re-exported for introspection.
var StandardTags = tags.StandardTags

// Constructor re-exports — thin wrappers keep the public API stable.
var (
	NewContext          = engine.NewContext
	NewParseContext     = engine.NewParseContext
	BuildContext        = engine.BuildContext
	NewStrainerTemplate = engine.NewStrainerTemplate
	NewI18n             = engine.NewI18n
	NewTagBase          = engine.NewTagBase
	NewStringNode       = engine.NewStringNode
	NewBlockBody        = engine.NewBlockBody
	NewBlock            = engine.NewBlock
	NewCondition        = engine.NewCondition
	NewElseCondition    = engine.NewElseCondition
	ParseCondition      = engine.ParseCondition
	ParseExpression     = engine.ParseExpression
	ParseDocument       = engine.ParseDocument
	IsTruthy            = engine.IsTruthy
	CompareValues       = engine.CompareValues
	SliceCollection     = engine.SliceCollection
	UtilsToString       = engine.UtilsToString
	UtilsToInteger      = engine.UtilsToInteger
	UtilsToNumber       = engine.UtilsToNumber
	ToLiquidValue       = engine.ToLiquidValue
	IsEvaluatable       = engine.IsEvaluatable
)
