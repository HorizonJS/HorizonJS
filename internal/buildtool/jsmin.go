package buildtool

import (
	"fmt"
	"strings"
)

// Port of the historical C# JSMin implementation (2003-04-21). Keep its behavior
// stable during the tooling migration; this is not a modern JavaScript parser.
//
// Copyright (c) 2002 Douglas Crockford
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to
// do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// The Software shall be used for Good, not Evil.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

const eof = rune(-1)

type jsmin struct {
	input           []rune
	pos             int
	a, b, lookahead rune
	out             strings.Builder
}

func minifyJS(source string) (string, error) {
	m := jsmin{input: []rune(source), a: '\n', lookahead: eof}
	if err := m.action(3); err != nil {
		return "", err
	}
	for m.a != eof {
		action := 1
		switch m.a {
		case ' ':
			if !isAlphanum(m.b) {
				action = 2
			}
		case '\n':
			switch m.b {
			case '{', '[', '(', '+', '-':
			case ' ':
				action = 3
			default:
				if !isAlphanum(m.b) {
					action = 2
				}
			}
		default:
			switch m.b {
			case ' ':
				if !isAlphanum(m.a) {
					action = 3
				}
			case '\n':
				switch m.a {
				case '}', ']', ')', '+', '-', '"', '\'':
				default:
					if !isAlphanum(m.a) {
						action = 3
					}
				}
			}
		}
		if err := m.action(action); err != nil {
			return "", err
		}
	}
	return m.out.String(), nil
}

func (m *jsmin) get() rune {
	c := m.lookahead
	m.lookahead = eof
	if c == eof && m.pos < len(m.input) {
		c = m.input[m.pos]
		m.pos++
	}
	if c >= ' ' || c == '\n' || c == eof {
		return c
	}
	if c == '\r' {
		return '\n'
	}
	return ' '
}

func (m *jsmin) peek() rune { m.lookahead = m.get(); return m.lookahead }

func (m *jsmin) next() (rune, error) {
	c := m.get()
	if c != '/' {
		return c, nil
	}
	switch m.peek() {
	case '/':
		for {
			c = m.get()
			if c <= '\n' {
				return c, nil
			}
		}
	case '*':
		m.get()
		for {
			switch m.get() {
			case '*':
				if m.peek() == '/' {
					m.get()
					return ' ', nil
				}
			case eof:
				return eof, fmt.Errorf("JSMin: unterminated comment")
			}
		}
	}
	return c, nil
}

func (m *jsmin) action(d int) error {
	if d <= 1 {
		m.out.WriteRune(m.a)
	}
	if d <= 2 {
		m.a = m.b
		if m.a == '\'' || m.a == '"' {
			for {
				m.out.WriteRune(m.a)
				m.a = m.get()
				if m.a == m.b {
					break
				}
				if m.a <= '\n' {
					return fmt.Errorf("JSMin: unterminated string literal")
				}
				if m.a == '\\' {
					m.out.WriteRune(m.a)
					m.a = m.get()
				}
			}
		}
	}
	if d <= 3 {
		var err error
		m.b, err = m.next()
		if err != nil {
			return err
		}
		if m.b == '/' && (m.a == '(' || m.a == ',' || m.a == '=') {
			m.out.WriteRune(m.a)
			m.out.WriteRune(m.b)
			for {
				m.a = m.get()
				if m.a == '/' {
					break
				}
				if m.a == '\\' {
					m.out.WriteRune(m.a)
					m.a = m.get()
				} else if m.a <= '\n' {
					return fmt.Errorf("JSMin: unterminated regular expression literal")
				}
				m.out.WriteRune(m.a)
			}
			m.b, err = m.next()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isAlphanum(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '$' || c == '\\' || c > 126
}
