// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"path"
	"regexp"
	"slices"
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
		case isAssignment(w[0]), slices.Contains(keywords, w[0]):
			w = w[1:]
		case w[0] == "command" && len(w) > 1 && (w[1] == "-v" || w[1] == "-V"):
			// command -v looks a command up and runs nothing.
			return "command", w[1:]
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

// keywords are the shell's reserved words that may come before a command.
var keywords = []string{"do", "then", "else", "elif", "if", "while", "until", "!"}

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

// assignments reports whether every word of s assigns a variable, as
// M=/path/to/metais does, and the variables it assigns.
func (s simple) assignments() (map[string]string, bool) {
	vars := map[string]string{}
	for _, w := range s.words {
		if !isAssignment(w) {
			return nil, false
		}
		k, v, _ := strings.Cut(w, "=")
		vars[k] = v
	}
	return vars, len(vars) > 0
}

// variable is a reference to a shell variable: $NAME or ${NAME}.
var variable = regexp.MustCompile(`\$(?:\{([A-Za-z_][A-Za-z0-9_]*)\}|([A-Za-z_][A-Za-z0-9_]*))`)

// expand replaces each reference to a variable vars holds in s's words,
// so a session that runs metais as $M, having set M, is seen to run it.
// A variable vars does not hold is left as it is.
func (s simple) expand(vars map[string]string) simple {
	if len(vars) == 0 {
		return s
	}
	ex := func(ws []string) []string {
		out := make([]string, len(ws))
		for i, w := range ws {
			out[i] = variable.ReplaceAllStringFunc(w, func(m string) string {
				sub := variable.FindStringSubmatch(m)
				name := sub[1] + sub[2]
				if v, ok := vars[name]; ok {
					return v
				}
				return m
			})
		}
		return out
	}
	return simple{words: ex(s.words), outs: ex(s.outs), ins: ex(s.ins)}
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

// lead is a command's words without the reserved words a list begins
// with inside a compound command, such as the do of do for f in …, or
// its last word when it is nothing else.
func lead(w []string) []string {
	for len(w) > 1 && slices.Contains([]string{"do", "then", "else", "elif", "!"}, w[0]) {
		w = w[1:]
	}
	return w
}

// unroll reads each for loop of a command line, for NAME in WORD…; do …;
// done, as its body once for each word, with $NAME expanded to the word,
// so a command that names a directory through the loop's variable is
// seen to name it. A loop whose words the line does not hold, such as
// for f in $(ls), is left as it is, its body read once with $NAME
// unexpanded.
func unroll(ss []simple) []simple {
	var out []simple
	for i := 0; i < len(ss); i++ {
		w := lead(ss[i].words)
		if len(w) < 4 || w[0] != "for" || w[2] != "in" || !isAssignment(w[1]+"=") {
			out = append(out, ss[i])
			continue
		}
		end, depth := -1, 0
		for j := i + 1; j < len(ss) && end < 0; j++ {
			switch lead(ss[j].words)[0] {
			case "for", "while", "until":
				depth++
			case "done":
				if depth == 0 {
					end = j
				}
				depth--
			}
		}
		if end < 0 {
			out = append(out, ss[i])
			continue
		}
		for _, v := range w[3:] {
			var body []simple
			for _, b := range ss[i+1 : end] {
				body = append(body, b.expand(map[string]string{w[1]: v}))
			}
			out = append(out, unroll(body)...)
		}
		i = end
	}
	return out
}

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

// Command is one simple command of a shell command line as the scorer
// reads it: its words, with quotes removed, and the files it redirects to
// and from.
type Command struct{ Words, Outs, Ins []string }

// Name is the command's name, past any variable assignments and
// wrappers, and the words after it.
func (c Command) Name() (string, []string) { return simple{words: c.Words}.name() }

// Commands splits a Bash tool call's command line into its simple
// commands, in order, as the scorer reads them, so a check outside the
// scorer reads a session's commands the same way.
func Commands(src string) []Command {
	var out []Command
	for _, s := range parseShell(src) {
		out = append(out, Command{Words: s.words, Outs: s.outs, Ins: s.ins})
	}
	return out
}
