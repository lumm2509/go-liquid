package liquid

import (
	"fmt"
	"strings"
)

// Document representa la raíz del árbol de parseo de Liquid.
type Document struct {
	ParseContext *ParseContext
	Body         *BlockBody // Asumiendo que BlockBody ya está definido
}

// NewDocument es el equivalente a Document.new
func NewDocument(parseContext *ParseContext) *Document {
	doc := &Document{
		ParseContext: parseContext,
	}
	doc.Body = doc.newBody()
	return doc
}

// Parse es el punto de entrada principal (equivalente a self.parse)
func ParseDocument(tokenizer *Tokenizer, parseContext *ParseContext) (*Document, error) {
	doc := NewDocument(parseContext)
	err := doc.Parse(tokenizer, parseContext)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// NodeList devuelve los nodos contenidos en el cuerpo del documento
func (d *Document) NodeList() []Node {
	return d.Body.NodeList
}

// Parse ejecuta el bucle de parseo
func (d *Document) Parse(tokenizer *Tokenizer, parseContext *ParseContext) error {
	for {
		continued, err := d.parseBody(tokenizer)
		if err != nil {
			// En Ruby se asigna el line_number al error si no lo tiene
			// Aquí podrías envolver el error con más contexto
			return fmt.Errorf("line %d: %w", parseContext.LineNumber, err)
		}
		if !continued {
			break
		}
	}
	// En Ruby se hace @body.freeze, en Go simplemente dejamos de modificarlo
	return nil
}

// UnknownTag maneja etiquetas inesperadas o desconocidas
func (d *Document) UnknownTag(tag string, markup string, tokenizer *Tokenizer) error {
	switch tag {
	case "else", "end":
		// Aquí idealmente usarías un sistema de i18n, pero fmt.Errorf es el estándar
		return fmt.Errorf("syntax error: unexpected outer tag '%s'", tag)
	default:
		return fmt.Errorf("syntax error: unknown tag '%s'", tag)
	}
}

// RenderToOutputBuffer renderiza el documento en un builder (más eficiente que strings)
func (d *Document) RenderToOutputBuffer(context *Context, output *strings.Builder) error {
	return d.Body.RenderToOutputBuffer(context, output)
}

// Render devuelve el string final renderizado
func (d *Document) Render(context *Context) (string, error) {
	var sb strings.Builder
	err := d.RenderToOutputBuffer(context, &sb)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

// --- Métodos Privados ---

func (d *Document) newBody() *BlockBody {
	return d.ParseContext.NewBlockBody()
}

func (d *Document) parseBody(tokenizer *Tokenizer) (bool, error) {
	// En Ruby, @body.parse acepta un bloque.
	// En Go, pasamos una función anónima (callback).
	return d.Body.Parse(tokenizer, d.ParseContext, func(tagName string, tagMarkup string) (bool, error) {
		if tagName != "" {
			err := d.UnknownTag(tagName, tagMarkup, tokenizer)
			return true, err
		}
		return false, nil
	})
}
