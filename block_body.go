package liquid

import (
	"fmt"
	"regexp"
	"strings"
)

// Regex exactos extraídos de la implementación de Ruby
var (
	LiquidTagToken  = regexp.MustCompile(`(?s)^\s*([\w#]+)\s*(.*)$`)
	FullToken       = regexp.MustCompile(`(?s)^\{%[-]?(\s*)([\w#]+)(\s*)(.*?)[-]?%\}$`)
	WhitespaceRegex = regexp.MustCompile(`^\s*$`)
)

type UnknownTagHandler func(tagName string, tagMarkup string) (bool, error)

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

// Lógica para etiquetas dentro de {% liquid ... %}
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
			// Si no coincide con la sintaxis de tag, dejamos que el llamador decida (error de sintaxis)
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

			// Importante: Liquid cuenta las líneas dentro de los tags multilínea
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
	// Crea un nuevo tokenizer específicamente para el contenido del tag liquid
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
	// Trim a la derecha del nodo anterior si hay un '-' al inicio del tag actual
	if len(token) >= 3 && token[2] == WhitespaceControl[0] {
		if len(b.NodeList) > 0 {
			lastIdx := len(b.NodeList) - 1
			if sn, ok := b.NodeList[lastIdx].(*StringNode); ok {
				sn.TrimRight()
			}
		}
	}
	// Indicar al parseContext que el siguiente nodo de texto debe ser trimmeado a la izquierda
	if len(token) >= 3 && token[len(token)-3] == WhitespaceControl[0] {
		parseContext.TrimWhitespace = true
	}
}

func (b *BlockBody) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	context.ResourceLimits.IncrementRenderScore(len(b.NodeList))

	for _, node := range b.NodeList {
		if err := b.renderNode(context, output, node); err != nil {
			return err
		}
		if context.Interrupt() {
			break
		}
		context.ResourceLimits.IncrementWriteScore(output.Len())
	}
	return nil
}

func (b *BlockBody) renderNode(context *Context, output *strings.Builder, node Node) error {
	err := node.RenderToOutputBuffer(context, output)
	if err != nil {
		if _, ok := err.(MemoryError); ok {
			return err
		}
		if context.Environment != nil && context.Environment.Logger != nil {
			context.Environment.Logger.Log(DebugEvent{
				Event: "render.node_error",
				Data:  map[string]interface{}{"line": node.LineNumber(), "error": err.Error()},
			})
		}
		return err
	}
	return nil
}

func (b *BlockBody) createVariable(token string, parseContext *ParseContext) (*Variable, error) {
	if strings.HasSuffix(token, VariableEndStr) {
		i := 2
		if token[i] == WhitespaceControl[0] {
			i = 3
		}
		parseEnd := len(token) - 2
		if token[parseEnd-1] == WhitespaceControl[0] {
			parseEnd -= 1
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

func (b *BlockBody) RemoveBlankStrings() error {
	if !b.isBlank {
		return fmt.Errorf("remove_blank_strings only support being called on a blank block body")
	}
	newNodes := make([]Node, 0)
	for _, node := range b.NodeList {
		if _, ok := node.(*StringNode); !ok {
			newNodes = append(newNodes, node)
		}
	}
	b.NodeList = newNodes
	return nil
}
