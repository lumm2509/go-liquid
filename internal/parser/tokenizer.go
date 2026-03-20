package parser

type Tokenizer struct {
	source       []byte
	pos          int
	length       int
	LineNumber   int
	withLineNums bool
	ForLiquidTag bool
}

func NewTokenizer(source string, startLineNumber int, forLiquidTag bool) *Tokenizer {
	return &Tokenizer{
		source:       []byte(source),
		length:       len(source),
		pos:          0,
		LineNumber:   startLineNumber,
		withLineNums: true,
		ForLiquidTag: forLiquidTag,
	}
}

func (t *Tokenizer) Shift() (string, bool) {
	return t.Next()
}

func (t *Tokenizer) Next() (string, bool) {
	if t.pos >= t.length {
		return "", false
	}

	if t.ForLiquidTag {
		return t.readLine()
	}

	if t.pos+1 < t.length && t.source[t.pos] == '{' {
		next := t.source[t.pos+1]
		if next == '{' {
			return t.readVariable()
		}
		if next == '%' {
			return t.readTag()
		}
	}

	return t.readText()
}

func (t *Tokenizer) readLine() (string, bool) {
	start := t.pos
	for t.pos < t.length {
		if t.source[t.pos] == '\n' {
			res := string(t.source[start:t.pos])
			t.pos++
			if t.withLineNums {
				t.LineNumber++
			}
			return res, true
		}
		t.pos++
	}
	return string(t.source[start:t.pos]), true
}

func (t *Tokenizer) peek() byte {
	return t.source[t.pos]
}

func (t *Tokenizer) advance(n int) {
	t.pos += n
}

func (t *Tokenizer) readText() (string, bool) {
	start := t.pos

	for t.pos < t.length {
		if t.pos+1 < t.length && t.source[t.pos] == '{' {
			next := t.source[t.pos+1]
			if next == '{' || next == '%' {
				break
			}
		}
		if t.withLineNums && t.source[t.pos] == '\n' {
			t.LineNumber++
		}
		t.pos++
	}

	return string(t.source[start:t.pos]), true
}

func (t *Tokenizer) readVariable() (string, bool) {
	start := t.pos
	t.advance(2)

	for t.pos < t.length-1 {
		if t.source[t.pos] == '}' && t.source[t.pos+1] == '}' {
			t.advance(2)
			return string(t.source[start:t.pos]), true
		}
		if t.withLineNums && t.source[t.pos] == '\n' {
			t.LineNumber++
		}
		t.pos++
	}

	res := string(t.source[start:])
	t.pos = t.length
	return res, true
}

func (t *Tokenizer) readTag() (string, bool) {
	start := t.pos
	t.advance(2)

	for t.pos < t.length-1 {
		if t.source[t.pos] == '%' && t.source[t.pos+1] == '}' {
			t.advance(2)
			return string(t.source[start:t.pos]), true
		}
		if t.withLineNums && t.source[t.pos] == '\n' {
			t.LineNumber++
		}
		t.pos++
	}

	res := string(t.source[start:])
	t.pos = t.length
	return res, true
}
