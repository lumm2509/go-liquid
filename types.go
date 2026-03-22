package liquid

import (
	"github.com/go-liquid/internal/engine"
	"github.com/go-liquid/internal/filters"
	"github.com/go-liquid/internal/tags"
)

// types custom tag and filter authors program against; do not import internal/ directly
type (
	RenderContext     = engine.RenderContext
	TagParseContext   = engine.TagParseContext
	Tag               = engine.Tag
	Node              = engine.Node
	TagFactory        = engine.TagFactory
	UnknownTagHandler = engine.UnknownTagHandler
	ExceptionRenderer = engine.ExceptionRenderer
	FileSystem        = engine.FileSystem
	DebugLogger       = engine.DebugLogger
	// EnvironmentIface is only needed for custom environment implementations
	EnvironmentIface = engine.EnvironmentIface
)

// re-exported for API stability; internal/engine cannot be imported externally
type TagBase      = engine.TagBase
type StringNode   = engine.StringNode
type ParsedPartial = engine.ParsedPartial

type I18n = engine.I18n

type FilterRegistry = engine.FilterRegistry
type FilterDispatcher = engine.FilterDispatcher
type StandardFilters  = filters.StandardFilters

// SafeHTML marks a string as already-safe HTML; bypasses auto-escaping
type SafeHTML = engine.SafeHTML

type DebugEvent     = engine.DebugEvent
type DebugEventType = engine.DebugEventType

const (
	EventFilterNotFound           = engine.EventFilterNotFound
	EventEnvironmentFrozenTag     = engine.EventEnvironmentFrozenTag
	EventContextOverflow          = engine.EventContextOverflow
	EventRenderNodeError          = engine.EventRenderNodeError
	EventConditionUnknownOperator = engine.EventConditionUnknownOperator
)

// ContextConfig is for custom environments and tests that construct contexts directly
type ContextConfig = engine.ContextConfig

// ParseContext is exposed for custom tags that need to type-assert TagParseContext
type ParseContext = engine.ParseContext

var StandardTags = tags.StandardTags

var (
	NewContext        = engine.NewContext
	NewFilterRegistry = engine.NewFilterRegistry
	NewI18n           = engine.NewI18n
	NewTagBase        = engine.NewTagBase
	NewStringNode     = engine.NewStringNode
	ParseExpression   = engine.ParseExpression
	IsTruthy          = engine.IsTruthy
	CompareValues     = engine.CompareValues
	SliceCollection   = engine.SliceCollection
	UtilsToString     = engine.UtilsToString
	UtilsToInteger    = engine.UtilsToInteger
	UtilsToNumber     = engine.UtilsToNumber
	ToLiquidValue     = engine.ToLiquidValue
	IsEvaluatable     = engine.IsEvaluatable
)
