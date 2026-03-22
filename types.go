package liquid

import (
	"github.com/go-liquid/internal/engine"
	"github.com/go-liquid/internal/filters"
	"github.com/go-liquid/internal/tags"
)

// ---------------------------------------------------------------------------
// Public interface contracts
// ---------------------------------------------------------------------------
// These are the types custom tag and filter authors program against.
// internal/engine implements them; consumers should not depend on the
// concrete types in internal/.

type (
	// RenderContext is the interface tags receive during rendering.
	RenderContext = engine.RenderContext

	// TagParseContext is the interface tags receive during parsing.
	TagParseContext = engine.TagParseContext

	// Tag is implemented by every tag node.
	Tag = engine.Tag

	// Node is implemented by every AST node that can render itself.
	Node = engine.Node

	// TagFactory is the function signature for registering custom tags.
	TagFactory = engine.TagFactory

	// UnknownTagHandler is called during parsing for unrecognised tags.
	UnknownTagHandler = engine.UnknownTagHandler

	// ExceptionRenderer transforms render errors before returning them.
	ExceptionRenderer = engine.ExceptionRenderer

	// FileSystem abstracts template file loading for include/render tags.
	FileSystem = engine.FileSystem

	// DebugLogger is the observer interface for internal engine events.
	DebugLogger = engine.DebugLogger

	// EnvironmentIface is the interface the engine uses to call back into
	// the environment. Implement it only if you need a custom environment;
	// most users should use *Environment directly.
	EnvironmentIface = engine.EnvironmentIface
)

// ---------------------------------------------------------------------------
// Utility types for custom tag authors
// ---------------------------------------------------------------------------

// TagBase is the base implementation for Tag. Embed it in custom tags.
// Re-exported for API stability: internal/engine cannot be imported externally.
type TagBase = engine.TagBase

// StringNode holds a raw text node. Useful when building custom tags
// that produce static output during Parse.
// Re-exported for API stability: internal/engine cannot be imported externally.
type StringNode = engine.StringNode

// I18n holds locale translations used by tags that support i18n.
type I18n = engine.I18n

// ParsedPartial is a parsed partial template returned by the partial cache.
// Re-exported for API stability: internal/engine cannot be imported externally.
type ParsedPartial = engine.ParsedPartial

// ---------------------------------------------------------------------------
// Filter system
// ---------------------------------------------------------------------------

// FilterRegistry holds the filter configuration for an Environment.
// Built once; shared across all renders.
type FilterRegistry = engine.FilterRegistry

// FilterDispatcher dispatches filter calls for a single render context.
type FilterDispatcher = engine.FilterDispatcher

// StandardFilters is the standard Liquid filter set.
// Re-exported for tests and custom environments.
type StandardFilters = filters.StandardFilters

// SafeHTML marks a string as already-safe HTML that should not be escaped
// when AutoEscape is active. Re-exported from internal/engine for external use.
type SafeHTML = engine.SafeHTML

// ---------------------------------------------------------------------------
// Debug / observability
// ---------------------------------------------------------------------------

// DebugEvent is passed to DebugLogger on each internal event.
type DebugEvent = engine.DebugEvent

// DebugEventType is the type for structured debug event identifiers.
type DebugEventType = engine.DebugEventType

// Typed debug event constants — use these in DebugLogger implementations
// instead of raw strings to enable reliable filtering and alerting.
const (
	EventFilterNotFound           = engine.EventFilterNotFound
	EventEnvironmentFrozenTag     = engine.EventEnvironmentFrozenTag
	EventContextOverflow          = engine.EventContextOverflow
	EventRenderNodeError          = engine.EventRenderNodeError
	EventConditionUnknownOperator = engine.EventConditionUnknownOperator
)

// ---------------------------------------------------------------------------
// Context construction (advanced use)
// ---------------------------------------------------------------------------

// ContextConfig holds named parameters for constructing a render Context.
// Used by custom environments and tests that need to build contexts directly.
type ContextConfig = engine.ContextConfig

// ParseContext is the parse-time state. Exposed for custom tags that need
// to type-assert the TagParseContext they receive.
type ParseContext = engine.ParseContext

// ---------------------------------------------------------------------------
// Standard tags registry
// ---------------------------------------------------------------------------

// StandardTags is the default tag registry; re-exported for introspection.
var StandardTags = tags.StandardTags

// ---------------------------------------------------------------------------
// Constructor re-exports
// ---------------------------------------------------------------------------

var (
	// NewContext constructs a render Context from a ContextConfig.
	NewContext = engine.NewContext

	// NewFilterRegistry creates an empty FilterRegistry.
	NewFilterRegistry = engine.NewFilterRegistry

	// NewI18n creates an I18n locale holder from a JSON string.
	NewI18n = engine.NewI18n

	// NewTagBase constructs a TagBase for use in custom tag structs.
	// Re-exported for API stability: internal/engine cannot be imported externally.
	NewTagBase = engine.NewTagBase

	// NewStringNode constructs a raw text AST node.
	// Re-exported for API stability: internal/engine cannot be imported externally.
	NewStringNode = engine.NewStringNode

	// ParseExpression parses a markup expression string.
	ParseExpression = engine.ParseExpression

	// IsTruthy returns Liquid's truthiness for a value.
	IsTruthy = engine.IsTruthy

	// CompareValues compares two values for ordering.
	CompareValues = engine.CompareValues

	// SliceCollection converts a Liquid collection to a Go slice.
	SliceCollection = engine.SliceCollection

	// UtilsToString converts a value to its Liquid string representation.
	UtilsToString = engine.UtilsToString

	// UtilsToInteger converts a value to int, returning an error if not possible.
	UtilsToInteger = engine.UtilsToInteger

	// UtilsToNumber converts a value to a numeric type.
	UtilsToNumber = engine.UtilsToNumber

	// ToLiquidValue normalises a Go value into a Liquid-compatible type.
	// Re-exported for API stability: internal/engine cannot be imported externally.
	ToLiquidValue = engine.ToLiquidValue

	// IsEvaluatable reports whether obj implements the Evaluate(*Context) method.
	// Re-exported for API stability: internal/engine cannot be imported externally.
	IsEvaluatable = engine.IsEvaluatable
)
