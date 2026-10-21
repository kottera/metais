// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"path"
	"strings"
)

// simple is one simple command of a shell command line: its words, with
// quotes removed and redirections taken out, and the files it redirects
// to or from.
type simple struct {
	words []string
	// outs are the files its output is redirected to, ins the files its
	// input is read from.
	outs, ins []string
}

// name is the command's name: its first word after any variable
// assignments and wrappers, as the base name of the path it is given as.
// rest is the words after it.
func (s simple) name() (name string, rest []string) {
	w := s.words
	for len(w) > 0 {
		switch {
		case isAssignment(w[0]):
			w = w[1:]
		case w[0] == "env" || w[0] == "sudo" || w[0] == "time" || w[0] == "nohup" || w[0] == "exec" || w[0] == "command":
			w = w[1:]
			for len(w) > 0 && strings.HasPrefix(w[0], "-") {
				w = w[1:]
			}
		default:
			return path.Base(w[0]), w[1:]
		}
	}
	return "", nil
}

func isAssignment(w string) bool {
	k, _, ok := strings.Cut(w, "=")
	if !ok || k == "" {
		return false
	}
	for i, r := range k {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// parseShell splits a Bash tool call's command line into its simple
// commands, in order. It reads the POSIX shell's quoting, its control
// operators, subshells, command substitutions and here-documents, and
// what a shell's -c runs, which is all a session's commands are made of.
// It is not a shell: it expands no variable and no glob, so a command
// that names metais through a variable is not seen as metais.
func parseShell(src string) []simple {
	var (
		out      []simple
		cur      simple
		word     strings.Builder
		inWord   bool
		redirect string // ">" or "<" when the next word is a redirection's file
		heredocs []heredoc
	)
	endWord := func() {
		if !inWord {
			return
		}
		w := word.String()
		word.Reset()
		inWord = false
		switch redirect {
		case ">":
			cur.outs = append(cur.outs, w)
		case "<":
			cur.ins = append(cur.ins, w)
		case "<<":
			heredocs = append(heredocs, heredoc{delim: w})
		default:
			cur.words = append(cur.words, w)
		}
		redirect = ""
	}
	endCommand := func() {
		endWord()
		if len(cur.words) > 0 {
			out = append(out, cur)
			// A shell's -c runs its argument as a command line of its own.
			if n, rest := cur.name(); (n == "bash" || n == "sh" || n == "zsh") && len(rest) >= 2 && rest[0] == "-c" {
				out = append(out, parseShell(rest[1])...)
			}
		}
		cur = simple{}
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '\\' && i+1 < len(src):
			if src[i+1] != '\n' {
				word.WriteByte(src[i+1])
				inWord = true
			}
			i++
		case c == '\'':
			j := strings.IndexByte(src[i+1:], '\'')
			if j < 0 {
				j = len(src) - i - 1
			}
			word.WriteString(src[i+1 : i+1+j])
			inWord = true
			i += j + 1
		case c == '"':
			i++
			for ; i < len(src) && src[i] != '"'; i++ {
				if src[i] == '\\' && i+1 < len(src) && strings.IndexByte("\"\\$`\n", src[i+1]) >= 0 {
					i++
				}
				word.WriteByte(src[i])
			}
			inWord = true
		case c == '\n':
			endCommand()
			// The here-documents begun on the line just ended are its next
			// lines, up to each one's delimiter: their text is no command.
			for _, h := range heredocs {
				for i+1 < len(src) {
					end := strings.IndexByte(src[i+1:], '\n')
					line := src[i+1:]
					if end >= 0 {
						line = src[i+1 : i+1+end]
					}
					if end < 0 {
						i = len(src)
					} else {
						i += end + 1
					}
					if strings.TrimLeft(line, "\t") == h.delim {
						break
					}
				}
			}
			heredocs = nil
		case c == ';' || c == '|' || c == '&' || c == '(' || c == ')' || c == '`' || c == '{' && !inWord || c == '}' && !inWord:
			if c == '&' && (redirect == ">" || i > 0 && (src[i-1] == '>' || src[i-1] == '<')) {
				// 2>&1: a descriptor, not a control operator.
				for i+1 < len(src) && (src[i+1] >= '0' && src[i+1] <= '9' || src[i+1] == '-') {
					i++
				}
				redirect = ""
				continue
			}
			if c == '&' && i+1 < len(src) && src[i+1] == '>' {
				// &> file
				endWord()
				redirect = ">"
				i++
				if i+1 < len(src) && src[i+1] == '>' {
					i++
				}
				continue
			}
			endCommand()
		case c == '$' && i+1 < len(src) && src[i+1] == '(':
			endCommand()
			i++
		case c == '>' || c == '<':
			// A descriptor number just before it belongs to the operator.
			if inWord && isDigits(word.String()) {
				word.Reset()
				inWord = false
			}
			endWord()
			if c == '<' && i+1 < len(src) && src[i+1] == '<' {
				i++
				if i+1 < len(src) && src[i+1] == '<' { // <<< here-string
					i++
					redirect = ""
					continue
				}
				if i+1 < len(src) && src[i+1] == '-' {
					i++
				}
				redirect = "<<"
				continue
			}
			if i+1 < len(src) && src[i+1] == '>' {
				i++
			}
			if i+1 < len(src) && src[i+1] == '(' { // <(…) and >(…)
				endCommand()
				i++
				continue
			}
			redirect = string(c)
		case c == ' ' || c == '\t':
			endWord()
		case c == '#' && !inWord:
			// A comment, to the end of the line.
			for i+1 < len(src) && src[i+1] != '\n' {
				i++
			}
		default:
			word.WriteByte(c)
			inWord = true
		}
	}
	endCommand()
	return out
}

type heredoc struct{ delim string }

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
