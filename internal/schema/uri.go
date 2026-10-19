// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import "strings"

// RFC 3986's character classes, as the grammar of §2 and Appendix A
// names them.
func isUnreserved(c byte) bool {
	return isAlpha(c) || isDigit(c) || c == '-' || c == '.' || c == '_' || c == '~'
}

func isSubDelim(c byte) bool { return strings.IndexByte("!$&'()*+,;=", c) >= 0 }

// validChars reports whether s is made of pct-encoded triplets and bytes
// for which allowed holds.
func validChars(s string, allowed func(byte) bool) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			if i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2]) {
				return false
			}
			i += 2
			continue
		}
		if !allowed(s[i]) {
			return false
		}
	}
	return true
}

// isPchar is pchar without its pct-encoded alternative.
func isPchar(c byte) bool { return isUnreserved(c) || isSubDelim(c) || c == ':' || c == '@' }

func isQueryChar(c byte) bool { return isPchar(c) || c == '/' || c == '?' }

// isURI is RFC 3986 §3's URI: scheme ":" hier-part ["?" query]
// ["#" fragment].
func isURI(s string) bool {
	colon := strings.IndexByte(s, ':')
	if colon < 1 || !isScheme(s[:colon]) {
		return false
	}
	return isRelativePart(s[colon+1:], true)
}

// isURIReference is RFC 3986 §4.1's URI-reference: a URI or a
// relative-ref.
func isURIReference(s string) bool {
	if isURI(s) {
		return true
	}
	return isRelativePart(s, false)
}

// isScheme is ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ).
func isScheme(s string) bool {
	if s == "" || !isAlpha(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isAlpha(s[i]) && !isDigit(s[i]) && s[i] != '+' && s[i] != '-' && s[i] != '.' {
			return false
		}
	}
	return true
}

// isRelativePart checks what follows a scheme's colon (hier-part, when
// absolute is set) or a whole relative-ref: the path forms, then the
// optional query and fragment. A relative-ref's first segment, when it
// has no authority and does not begin with "/", holds no colon
// (path-noscheme).
func isRelativePart(s string, absolute bool) bool {
	if hash := strings.IndexByte(s, '#'); hash >= 0 {
		if !validChars(s[hash+1:], isQueryChar) {
			return false
		}
		s = s[:hash]
	}
	if q := strings.IndexByte(s, '?'); q >= 0 {
		if !validChars(s[q+1:], isQueryChar) {
			return false
		}
		s = s[:q]
	}
	if rest, ok := strings.CutPrefix(s, "//"); ok {
		auth, path := rest, ""
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			auth, path = rest[:slash], rest[slash:]
		}
		return isAuthority(auth) && isPath(path)
	}
	if !absolute && !strings.HasPrefix(s, "/") {
		first, _, _ := strings.Cut(s, "/")
		if strings.IndexByte(first, ':') >= 0 {
			return false
		}
	}
	return isPath(s)
}

// isPath is a run of segments separated by "/", each *pchar.
func isPath(s string) bool {
	return validChars(s, func(c byte) bool { return isPchar(c) || c == '/' })
}

// isAuthority is [ userinfo "@" ] host [ ":" port ].
func isAuthority(s string) bool {
	if at := strings.LastIndexByte(s, '@'); at >= 0 {
		if !validChars(s[:at], func(c byte) bool { return isUnreserved(c) || isSubDelim(c) || c == ':' }) {
			return false
		}
		s = s[at+1:]
	}
	host, port := s, ""
	if strings.HasPrefix(s, "[") {
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return false
		}
		host, port = s[:end+1], s[end+1:]
		if port != "" {
			if port[0] != ':' {
				return false
			}
			port = port[1:]
		}
	} else if colon := strings.LastIndexByte(s, ':'); colon >= 0 {
		host, port = s[:colon], s[colon+1:]
	}
	for i := 0; i < len(port); i++ {
		if !isDigit(port[i]) {
			return false
		}
	}
	if strings.HasPrefix(host, "[") {
		return isIPLiteral(host[1 : len(host)-1])
	}
	// reg-name, which an IPv4address also matches.
	return validChars(host, func(c byte) bool { return isUnreserved(c) || isSubDelim(c) })
}

// isIPLiteral is IPv6address / IPvFuture.
func isIPLiteral(s string) bool {
	if len(s) > 1 && (s[0] == 'v' || s[0] == 'V') {
		ver, rest, ok := strings.Cut(s[1:], ".")
		if !ok || ver == "" || rest == "" {
			return false
		}
		for i := 0; i < len(ver); i++ {
			if !isHex(ver[i]) {
				return false
			}
		}
		for i := 0; i < len(rest); i++ {
			if !isUnreserved(rest[i]) && !isSubDelim(rest[i]) && rest[i] != ':' {
				return false
			}
		}
		return true
	}
	return isIPv6(s)
}

// isURITemplate is RFC 6570 §2's URI-Template: literals and expressions,
// an expression an optional operator and a comma-separated list of
// varspecs, each a varname with an optional prefix (":" 1 to 4 digits, the
// first not 0) or explode ("*") modifier.
func isURITemplate(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '{':
			end := strings.IndexByte(s[i:], '}')
			if end < 0 || !isExpression(s[i+1:i+end]) {
				return false
			}
			i += end
		case c == '%':
			if i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2]) {
				return false
			}
			i += 2
		case c <= 0x20 || c == 0x7F || strings.IndexByte("\"'<>\\^`|}", c) >= 0:
			return false
		}
	}
	return true
}

func isExpression(s string) bool {
	if s != "" && strings.IndexByte("+#./;?&=,!@|", s[0]) >= 0 {
		if strings.IndexByte("=,!@|", s[0]) >= 0 {
			// Reserved for future extensions (§2.2).
			return false
		}
		s = s[1:]
	}
	if s == "" {
		return false
	}
	for _, spec := range strings.Split(s, ",") {
		name := spec
		if n, ok := strings.CutSuffix(spec, "*"); ok {
			name = n
		} else if n, prefix, ok := strings.Cut(spec, ":"); ok {
			name = n
			if len(prefix) < 1 || len(prefix) > 4 || prefix[0] == '0' {
				return false
			}
			if _, ok := digits(prefix); !ok {
				return false
			}
		}
		if !isVarname(name) {
			return false
		}
	}
	return true
}

// isVarname is varchar *( ["."] varchar ), varchar ALPHA / DIGIT / "_" /
// pct-encoded.
func isVarname(s string) bool {
	if s == "" || s[0] == '.' || s[len(s)-1] == '.' || strings.Contains(s, "..") {
		return false
	}
	return validChars(s, func(c byte) bool { return isAlpha(c) || isDigit(c) || c == '_' || c == '.' })
}
