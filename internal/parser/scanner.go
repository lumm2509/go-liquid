package parser

import (
	"regexp"
	"unicode/utf8"
)

type StringScanner struct {
	source string
	pos    int
}

func NewStringScanner(source string) *StringScanner {
	return &StringScanner{source: source, pos: 0}
}

func (s *StringScanner) SetString(source string) {
	s.source = source
	s.pos = 0
}

func (s *StringScanner) String() string {
	return s.source
}

func (s *StringScanner) Pos() int {
	return s.pos
}

func (s *StringScanner) SetPos(pos int) {
	s.pos = pos
}

func (s *StringScanner) EOS() bool {
	return s.pos >= len(s.source)
}

func (s *StringScanner) Scan(re *regexp.Regexp) string {
	loc := re.FindStringIndex(s.source[s.pos:])
	if loc == nil || loc[0] != 0 {
		return ""
	}
	match := s.source[s.pos : s.pos+loc[1]]
	s.pos += loc[1]
	return match
}

func (s *StringScanner) Skip(re *regexp.Regexp) int {
	loc := re.FindStringIndex(s.source[s.pos:])
	if loc == nil || loc[0] != 0 {
		return 0
	}
	n := loc[1]
	s.pos += n
	return n
}

func (s *StringScanner) PeekByte() (byte, bool) {
	if s.pos >= len(s.source) {
		return 0, false
	}
	return s.source[s.pos], true
}

func (s *StringScanner) ScanByte() (byte, bool) {
	if s.pos >= len(s.source) {
		return 0, false
	}
	b := s.source[s.pos]
	s.pos++
	return b, true
}

func (s *StringScanner) Getch() string {
	if s.pos >= len(s.source) {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s.source[s.pos:])
	s.pos += size
	return string(r)
}

func (s *StringScanner) Rest() string {
	return s.source[s.pos:]
}

func (s *StringScanner) Terminate() {
	s.pos = len(s.source)
}

func (s *StringScanner) SkipUntil(re *regexp.Regexp) int {
	loc := re.FindStringIndex(s.source[s.pos:])
	if loc == nil {
		return -1
	}
	n := loc[1]
	s.pos += n
	return n
}
