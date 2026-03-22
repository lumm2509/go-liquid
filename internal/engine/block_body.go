package engine

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	TagStartStr      = "{%"
	TagEndStr        = "%}"
	VariableStartStr = "{{"
	VariableEndStr   = "}}"
	WhitespaceControl = "-"
)

var (
	LiquidTagToken  = regexp.MustCompile(`(?s)^\s*([\w#]+)\s*(.*)$`)
	FullToken       = regexp.MustCompile(`(?s)^\{%[-]?(\s*)([\w#]+)(\s*)(.*?)[-]?%\}$`)
	WhitespaceRegex = regexp.MustCompile(`^\s*$`)
)

// BlockBody holds the parsed list of nodes for a block-level tag or document.
type BlockBody struct {
	NodeList []Node
	isBlank  bool
	frozen   bool
}

func NewBlockBody() *BlockBody {
	return &BlockBody{
		NodeList: make([]Node, 0),
		isBlank:  true,
	}
}

// Parse reads tokens from tokenizer and appends nodes, calling handler for unknown tags.
// Returns (continued bool, err error). continued=true means the caller should keep looping.
func (b *BlockBody) Parse(tokenizer *Tokenizer, parseContext *ParseContext, handler UnknownTagHandler) (bool, error) {
	if b.frozen {
		return false, fmt.Errorf("can't modify frozen Liquid::BlockBody")
	}
	parseContext.LineNumber = tokenizer.LineNumber
	if tokenizer.ForLiquidTag {
		return b.parseForLiquidTag(tokenizer, parseContext, handler)
	}
	return b.parseForDocument(tokenizer, parseContext, handler)
}

func (b *BlockBody) parseForLiquidTag(tokenizer *Tokenizer, parseContext *ParseContext, handler UnknownTagHandler) (bool, error) {
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}
		if len(token) == 0 || WhitespaceRegex.MatchString(token) {
			parseContext.LineNumber = tokenizer.LineNumber
			continue
		}

		matches := LiquidTagToken.FindStringSubmatch(token)
		if matches == nil {
			return handler(token, token)
		}

		tagName := matches[1]
		markup := matches[2]

		if tagName == "liquid" {
			parseContext.LineNumber--
			if err := b.parseLiquidTag(markup, parseContext); err != nil {
				return false, err
			}
			continue
		}

		tagFactory := parseContext.Environment.TagForName(tagName)
		if tagFactory == nil {
			return handler(tagName, markup)
		}

		newTag, err := tagFactory(tagName, markup, parseContext)
		if err != nil {
			return false, err
		}
		if err := newTag.Parse(tokenizer); err != nil {
			return false, err
		}
		b.NodeList = append(b.NodeList, newTag)
		parseContext.LineNumber = tokenizer.LineNumber
	}
	return false, nil
}

func (b *BlockBody) parseForDocument(tokenizer *Tokenizer, parseContext *ParseContext, handler UnknownTagHandler) (bool, error) {
	for {
		token, ok := tokenizer.Shift()
		if !ok {
			break
		}
		if len(token) == 0 {
			continue
		}

		switch {
		case strings.HasPrefix(token, TagStartStr):
			b.whitespaceHandler(token, parseContext)
			matches := FullToken.FindStringSubmatch(token)
			if matches == nil {
				return b.handleInvalidTagToken(token, parseContext, handler)
			}

			tagName := matches[2]
			markup := matches[4]
			parseContext.LineNumber += strings.Count(matches[1], "\n") + strings.Count(matches[3], "\n")

			if tagName == "liquid" {
				if err := b.parseLiquidTag(markup, parseContext); err != nil {
					return false, err
				}
				continue
			}

			tagFactory := parseContext.Environment.TagForName(tagName)
			if tagFactory == nil {
				return handler(tagName, markup)
			}

			newTag, err := tagFactory(tagName, markup, parseContext)
			if err != nil {
				return false, err
			}
			if err := newTag.Parse(tokenizer); err != nil {
				return false, err
			}
			b.isBlank = b.isBlank && newTag.IsBlank()
			b.NodeList = append(b.NodeList, newTag)

		case strings.HasPrefix(token, VariableStartStr):
			b.whitespaceHandler(token, parseContext)
			variable, err := b.createVariable(token, parseContext)
			if err != nil {
				return false, err
			}
			b.NodeList = append(b.NodeList, variable)
			b.isBlank = false

		default:
			if parseContext.TrimWhitespace {
				token = strings.TrimLeft(token, " \t\n\r")
			}
			parseContext.TrimWhitespace = false
			sn := NewStringNode(token, parseContext.LineNumber)
			b.NodeList = append(b.NodeList, sn)
			if !sn.IsBlank() {
				b.isBlank = false
			}
		}
		parseContext.LineNumber = tokenizer.LineNumber
	}
	return false, nil
}

func (b *BlockBody) parseLiquidTag(markup string, parseContext *ParseContext) error {
	lt := parseContext.NewTokenizer(markup, parseContext.LineNumber, true)
	_, err := b.parseForLiquidTag(lt, parseContext, func(tagName string, _ string) (bool, error) {
		if tagName != "" {
			return true, fmt.Errorf("unknown tag '%s' in liquid tag", tagName)
		}
		return false, nil
	})
	return err
}

func (b *BlockBody) whitespaceHandler(token string, parseContext *ParseContext) {
	if len(token) >= 3 && token[2] == '-' {
		if len(b.NodeList) > 0 {
			if sn, ok := b.NodeList[len(b.NodeList)-1].(*StringNode); ok {
				sn.TrimRight()
			}
		}
	}
	if len(token) >= 3 && token[len(token)-3] == '-' {
		parseContext.TrimWhitespace = true
	}
}

// RenderToOutputBuffer renders all nodes into output.
func (b *BlockBody) RenderToOutputBuffer(ctx RenderContext, output *strings.Builder) error {
	c, ok := ctx.(*Context)
	if !ok {
		return fmt.Errorf("BlockBody: expected *Context")
	}

	if goCtx := c.GoCtx; goCtx != nil {
		if err := goCtx.Err(); err != nil {
			return err
		}
	}

	if err := c.ResourceLimits.IncrementRenderScore(len(b.NodeList)); err != nil {
		return MemoryError{BaseError: BaseError{Message: err.Error(), Cause: err}}
	}

	hasLimits := c.ResourceLimits.RenderLengthLimit > 0 || c.ResourceLimits.AssignScoreLimit > 0

	for _, node := range b.NodeList {
		if err := b.renderNode(c, output, node); err != nil {
			return err
		}
		if c.Interrupt() {
			break
		}
		if hasLimits {
			if err := c.ResourceLimits.IncrementWriteScore(output.Len()); err != nil {
				return MemoryError{BaseError: BaseError{Message: err.Error(), Cause: err}}
			}
		}
	}
	return nil
}

func (b *BlockBody) renderNode(c *Context, output *strings.Builder, node Node) error {
	err := node.RenderToOutputBuffer(c, output)
	if err != nil {
		if _, ok := err.(MemoryError); ok {
			return err
		}
		if c.Environment != nil {
			if logger := c.Environment.GetLogger(); logger != nil {
				logger.Log(DebugEvent{
					Event: EventRenderNodeError,
					Data:  map[string]interface{}{"line": node.LineNumber(), "error": err.Error()},
				})
			}
		}
		return err
	}
	return nil
}

func (b *BlockBody) createVariable(token string, parseContext *ParseContext) (*Variable, error) {
	if strings.HasSuffix(token, VariableEndStr) {
		i := 2
		if token[i] == '-' {
			i = 3
		}
		parseEnd := len(token) - 2
		if token[parseEnd-1] == '-' {
			parseEnd--
		}
		markup := token[i:parseEnd]
		return NewVariable(markup, parseContext), nil
	}
	return nil, fmt.Errorf("variable '%s' was not properly terminated", token)
}

func (b *BlockBody) handleInvalidTagToken(token string, parseContext *ParseContext, handler UnknownTagHandler) (bool, error) {
	if strings.HasSuffix(token, TagEndStr) {
		return handler(token, token)
	}
	return true, fmt.Errorf("tag '%s' was not properly terminated", token)
}

// RemoveBlankStrings strips StringNode entries from a blank block body.
func (b *BlockBody) RemoveBlankStrings() error {
	if !b.isBlank {
		return fmt.Errorf("remove_blank_strings only support being called on a blank block body")
	}
	newNodes := make([]Node, 0, len(b.NodeList))
	for _, node := range b.NodeList {
		if _, ok := node.(*StringNode); !ok {
			newNodes = append(newNodes, node)
		}
	}
	b.NodeList = newNodes
	return nil
}

func (b *BlockBody) IsBlank() bool { return b.isBlank }
