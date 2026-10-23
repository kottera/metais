// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/tools/release/buildline"
	"metais.org/metais/internal/tools/site/mdhtml"
)

// repoURL is the repository's home, which a link of the specification to a
// file the site does not serve is made a link into.
const repoURL = "https://github.com/kottera/metais"

var (
	errSkill   = errors.New("metais skill --print printed no frontmatter")
	errSums    = errors.New("SHA256SUMS is not the release's")
	errRelease = errors.New("not a release version")
)

// skill is skill.json: the skill's text as metais skill --print prints it,
// and as an HTML fragment of its body, the text after its frontmatter.
type skill struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

// skillInput reads the skill metais skill --print printed.
func skillInput(text []byte) (skill, error) {
	rest, ok := bytes.CutPrefix(text, []byte("---\n"))
	if !ok {
		return skill{}, errSkill
	}
	_, body, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		return skill{}, errSkill
	}
	doc, err := mdhtml.Convert(body, nil)
	if err != nil {
		return skill{}, err
	}
	return skill{Text: string(text), HTML: doc.HTML}, nil
}

// spec is spec.json: the draft, from the checkout's own spec/, and each
// released revision, from its spec-<M>.<m> tag, oldest first.
type spec struct {
	Draft    revision   `json:"draft"`
	Releases []revision `json:"releases"`
}

// revision is one revision of the specification as the site serves it.
type revision struct {
	// Revision is "draft", or the release's <M>.<m>.
	Revision string `json:"revision"`
	// Ref is the git ref its files were read at: the checkout's commit for
	// the draft, and the tag for a release.
	Ref string `json:"ref"`
	// Path is the site's path of the revision, /spec/<revision>/.
	Path string `json:"path"`
	// Files are its Markdown files, spec/*.md, in byte order of the name.
	Files []specFile `json:"files"`
}

// specFile is one Markdown file of a revision.
type specFile struct {
	// File is its name in spec/.
	File string `json:"file"`
	// Path is the site's path of its page: the revision's path for
	// README.md, and the revision's path, the name without .md and a slash
	// for every other file.
	Path string `json:"path"`
	// Title is the plain text of its first level-1 heading.
	Title string `json:"title"`
	// HTML is the file as an HTML fragment, its links rewritten by specLink.
	HTML string `json:"html"`
	// Headings are its headings, each with the id its element carries.
	Headings []mdhtml.Heading `json:"headings"`
}

// specTag is a released revision's tag, ADR-011 A8 item 1's spec-<M>.<m>.
var specTag = regexp.MustCompile(`^spec-(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// specInput reads the draft from the checkout's spec/, and each released
// revision from its tag.
func (b *builder) specInput() (spec, error) {
	head, err := b.git("rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return spec{}, err
	}
	entries, err := os.ReadDir(filepath.Join(b.root, specDir))
	if err != nil {
		return spec{}, err
	}
	draft := map[string][]byte{}
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".md") {
			if draft[e.Name()], err = os.ReadFile(filepath.Join(b.root, specDir, e.Name())); err != nil {
				return spec{}, err
			}
		}
	}
	var out spec
	if out.Draft, err = convertRevision("draft", strings.TrimSpace(string(head)), draft); err != nil {
		return spec{}, err
	}

	refs, err := b.git("for-each-ref", "--format=%(refname:strip=2)", "refs/tags/spec-*")
	if err != nil {
		return spec{}, err
	}
	type tag struct {
		name         string
		major, minor int
	}
	var tags []tag
	for _, name := range strings.Fields(string(refs)) {
		m := specTag.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		major, err1 := strconv.Atoi(m[1])
		minor, err2 := strconv.Atoi(m[2])
		if err1 != nil || err2 != nil {
			continue
		}
		tags = append(tags, tag{name, major, minor})
	}
	slices.SortFunc(tags, func(a, b tag) int {
		return cmp.Or(cmp.Compare(a.major, b.major), cmp.Compare(a.minor, b.minor))
	})
	for _, t := range tags {
		files, err := b.tagFiles(t.name)
		if err != nil {
			return spec{}, err
		}
		r, err := convertRevision(strings.TrimPrefix(t.name, "spec-"), t.name, files)
		if err != nil {
			return spec{}, err
		}
		out.Releases = append(out.Releases, r)
	}
	if out.Releases == nil {
		out.Releases = []revision{}
	}
	return out, nil
}

// tagFiles reads the Markdown files of spec/ at tag, by name.
func (b *builder) tagFiles(tag string) (map[string][]byte, error) {
	commit := "refs/tags/" + tag + "^{commit}"
	list, err := b.git("ls-tree", "-z", commit, "--", specDir+"/")
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, line := range strings.Split(strings.TrimSuffix(string(list), "\x00"), "\x00") {
		// <mode> SP <type> SP <object> TAB <path>
		meta, p, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		name := strings.TrimPrefix(p, specDir+"/")
		if f := strings.Fields(meta); len(f) != 3 || f[1] != "blob" || !strings.HasSuffix(name, ".md") || strings.Contains(name, "/") {
			continue
		}
		if files[name], err = b.git("cat-file", "blob", commit+":"+p); err != nil {
			return nil, err
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("the tag %s holds no Markdown file in %s/", tag, specDir)
	}
	return files, nil
}

// convertRevision converts files, spec/'s Markdown files by name, as the
// revision rev read at ref.
func convertRevision(rev, ref string, files map[string][]byte) (revision, error) {
	r := revision{Revision: rev, Ref: ref, Path: "/spec/" + rev + "/", Files: []specFile{}}
	link := specLink(r.Path, ref, files)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		doc, err := mdhtml.Convert(files[name], link)
		if err != nil {
			return revision{}, fmt.Errorf("%s at %s: %w", name, ref, err)
		}
		r.Files = append(r.Files, specFile{
			File:     name,
			Path:     pagePath(r.Path, name),
			Title:    doc.Title,
			HTML:     doc.HTML,
			Headings: slices.Concat([]mdhtml.Heading{}, doc.Headings),
		})
	}
	return r, nil
}

// pagePath is the site's path of the page of name, a Markdown file of
// spec/, in the revision served at base.
func pagePath(base, name string) string {
	if name == "README.md" {
		return base
	}
	return base + strings.TrimSuffix(name, ".md") + "/"
}

// specLink rewrites a link of a file of spec/, in the revision served at
// base and read at ref, whose Markdown files are files:
//
//   - a link with a scheme, an absolute path or only a fragment is left as
//     it is;
//   - a link to one of files is made a link to its page, with its
//     fragment;
//   - any other relative link, to a file the site does not serve as a page,
//     is made a link to that file in the repository at ref: tree/ for a
//     link ending in a slash, blob/ otherwise.
func specLink(base, ref string, files map[string][]byte) func(string) string {
	return func(dest string) string {
		if dest == "" || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/") || hasScheme(dest) {
			return dest
		}
		p, frag, hasFrag := strings.Cut(dest, "#")
		if hasFrag {
			frag = "#" + frag
		}
		clean := path.Clean(specDir + "/" + p)
		if name, ok := strings.CutPrefix(clean, specDir+"/"); ok && !strings.Contains(name, "/") {
			if _, ok := files[name]; ok {
				return pagePath(base, name) + frag
			}
		}
		kind := "blob"
		if strings.HasSuffix(p, "/") {
			kind = "tree"
		}
		return repoURL + "/" + kind + "/" + ref + "/" + clean + frag
	}
}

// hasScheme reports whether dest begins with a URI scheme (RFC 3986 §3.1).
func hasScheme(dest string) bool {
	i := strings.IndexByte(dest, ':')
	if i <= 0 {
		return false
	}
	for j, c := range dest[:i] {
		letter := 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
		if !letter && (j == 0 || !('0' <= c && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return false
		}
	}
	return true
}

// release is release.json: published false for a build with no release,
// and otherwise the release's version, its tag, whether it is a
// prerelease, and each archive SHA256SUMS lists, in its order, where the
// release publishes it.
type release struct {
	Published  bool      `json:"published"`
	Version    string    `json:"version,omitempty"`
	Tag        string    `json:"tag,omitempty"`
	Prerelease bool      `json:"prerelease,omitempty"`
	SumsURL    string    `json:"sums_url,omitempty"`
	Archives   []archive `json:"archives,omitempty"`
}

// archive is one archive of a release.
type archive struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
}

// semver is a semantic version without a leading v (SemVer 2.0.0), with no
// build metadata, which a tag's name and an archive's cannot both carry.
var semver = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
	`(-((0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?$`)

// sumsLine is one line of SHA256SUMS, as sha256sum writes it.
var sumsLine = regexp.MustCompile(`^([0-9a-f]{64})  (\S+)$`)

// releaseInput reads the release the site is built for, from b.version and
// the SHA256SUMS file at b.sums.
func (b *builder) releaseInput() (release, error) {
	if b.version == "" {
		return release{Published: false}, nil
	}
	sums, err := os.ReadFile(b.sums)
	if err != nil {
		return release{}, err
	}
	return parseRelease(b.version, sums)
}

// parseRelease reads release version from sums, its SHA256SUMS: exactly
// one line per shipped target (buildline.Shipped), each naming the
// target's archive of version, in byte order of the name, and nothing
// else.
func parseRelease(version string, sums []byte) (release, error) {
	if !semver.MatchString(version) {
		return release{}, fmt.Errorf("%w: %q", errRelease, version)
	}
	tag := "v" + version
	r := release{
		Published:  true,
		Version:    version,
		Tag:        tag,
		Prerelease: strings.Contains(version, "-"),
		SumsURL:    repoURL + "/releases/download/" + tag + "/SHA256SUMS",
		Archives:   []archive{},
	}
	byName := map[string]buildline.Target{}
	for _, t := range buildline.Shipped {
		byName[buildline.ArchiveName(version, t)] = t
	}
	text, ok := strings.CutSuffix(string(sums), "\n")
	if !ok {
		return release{}, fmt.Errorf("%w: it does not end with a line feed", errSums)
	}
	var names []string
	for line := range strings.SplitSeq(text, "\n") {
		m := sumsLine.FindStringSubmatch(line)
		if m == nil {
			return release{}, fmt.Errorf("%w: %q is not a line sha256sum writes", errSums, line)
		}
		t, ok := byName[m[2]]
		if !ok {
			return release{}, fmt.Errorf("%w: %s is no archive of metais %s", errSums, m[2], version)
		}
		delete(byName, m[2])
		names = append(names, m[2])
		r.Archives = append(r.Archives, archive{
			Name: m[2], OS: t.OS, Arch: t.Arch, SHA256: m[1],
			URL: repoURL + "/releases/download/" + tag + "/" + m[2],
		})
	}
	if len(byName) > 0 {
		return release{}, fmt.Errorf("%w: it lists no %s", errSums, strings.Join(slices.Sorted(maps.Keys(byName)), ", "))
	}
	if !slices.IsSorted(names) {
		return release{}, fmt.Errorf("%w: its lines are not in byte order of the name", errSums)
	}
	return r, nil
}
