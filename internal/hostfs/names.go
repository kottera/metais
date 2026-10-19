// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// The prefixes of ADR-005 D14's fixed names.
const (
	stagingPrefix = ".metais-staging-"
	tempPrefix    = ".metais-tmp-"
)

// StagingName is the name of the staging directory a render into a new
// output directory whose base name is base creates beside it, in the same
// parent: ".metais-staging-" and the first 16 lowercase hex digits of the
// SHA-256 of base (ADR-005 D14 item 1.2).
//
// The name is a function of base alone. It reads no random source and no
// process identity, so a killed render's leftover, or a concurrent render
// of the same output, has the same name and is found by the exclusive
// create that makes it.
func StagingName(base string) string {
	return stagingPrefix + nameHash(base)
}

// TempName is the name of the temporary file a render into an existing
// output directory writes a file named name through, in the target's own
// directory: ".metais-tmp-" and the first 16 lowercase hex digits of the
// SHA-256 of name, the target's file name (ADR-005 D14 item 2.1).
func TempName(name string) string {
	return tempPrefix + nameHash(name)
}

func nameHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// osName checks name, a relative "/"-separated name, and returns it in the
// host's form. It refuses every name that could resolve anywhere but
// beneath the directory it is resolved in, whatever the root does with it:
// an empty name, an absolute one, a "", "." or ".." segment, and a "\",
// ":" or NUL, which a Windows host reads as a separator, a drive or a
// stream and a unix host truncates at. "." alone names the directory
// itself.
func osName(name string) (string, bool) {
	if name == "." {
		return ".", true
	}
	if name == "" || strings.ContainsAny(name, "\\:\x00") {
		return "", false
	}
	for seg := range strings.SplitSeq(name, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", false
		}
	}
	return filepath.FromSlash(name), true
}

// split returns name's parent, "." at the top, and its last segment.
func split(name string) (dir, base string) {
	i := strings.LastIndexByte(name, '/')
	if i < 0 {
		return ".", name
	}
	return name[:i], name[i+1:]
}
