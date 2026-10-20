// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/unitext"
)

// The output path codes (ADR-005 D12; ADR-005's interfaces name the
// family output.path). A render reports them over every planned path, and
// check over every path the manifest writes literally.
const (
	CodePathEncoding     finding.Code = "output.path.encoding"
	CodePathNotNFC       finding.Code = "output.path.not_nfc"
	CodePathShape        finding.Code = "output.path.shape"
	CodePathCharacter    finding.Code = "output.path.character"
	CodePathNotPortable  finding.Code = "output.path.not_portable"
	CodePathDuplicate    finding.Code = "output.path.duplicate"
	CodePathCaseCollide  finding.Code = "output.path.case_collision"
	CodePathFileDirClash finding.Code = "output.path.file_directory_conflict"
)

// PathCodes are the codes PathFinding and CollisionFinding report, beside
// limit.package and limit.exceeded.output.
func PathCodes() []finding.Code {
	return []finding.Code{CodePathEncoding, CodePathNotNFC, CodePathShape, CodePathCharacter,
		CodePathNotPortable, CodePathDuplicate, CodePathCaseCollide, CodePathFileDirClash}
}

// PathFinding is the finding of pr, one rule of ADR-005 D12 items 1 to 5
// an output path breaks, about subj at loc, with params, which hold the
// path and gain the rule's own: the output path codes, and for
// path-length and path-depth limit.package when the path is literal,
// since a manifest's path is the package's own content, and
// limit.exceeded.output when a path template rendered it, since that path
// is the render's (ADR-009 D19).
func PathFinding(pr unitext.Problem, subj finding.Subject, loc finding.Location, params finding.Params, literal bool, lim limits.Set) finding.Finding {
	var code finding.Code
	switch pr.Kind {
	case unitext.Encoding:
		code, params["reason"] = CodePathEncoding, finding.String(pr.Reason)
	case unitext.NotNFC:
		code = CodePathNotNFC
	case unitext.Shape:
		code, params["reason"] = CodePathShape, finding.String(pr.Reason)
	case unitext.Character:
		code, params["code_point"] = CodePathCharacter, finding.Int(int64(pr.CodePoint))
	case unitext.NotPortable:
		code, params["reason"] = CodePathNotPortable, finding.String(pr.Reason)
	case unitext.Length, unitext.Depth:
		id := limits.PathLength
		if pr.Kind == unitext.Depth {
			id = limits.PathDepth
		}
		if !literal {
			return lim.ExceededOutput(id, subj, loc, pr.Observed, true)
		}
		return lim.Breached(id, loc, pr.Observed, true)
	default:
		panic("template: an unknown path problem")
	}
	return finding.New(code, subj, &loc, params)
}

// CollisionFinding is the finding of c, ADR-005 D12 item 6 or 7, about
// subj's path at loc, with params, which hold the path and gain the
// colliding members; other is the location of the path it collides with.
func CollisionFinding(c unitext.Collision, subj finding.Subject, loc finding.Location, params finding.Params, other finding.Location) finding.Finding {
	switch c.Kind {
	case unitext.Duplicate:
		return finding.New(CodePathDuplicate, subj, &loc, params,
			finding.WithRelated(finding.Related{Location: other, Note: "first occurrence"}))
	case unitext.CaseCollision:
		params["member"], params["other"] = finding.String(c.Member), finding.String(c.OtherMember)
		return finding.New(CodePathCaseCollide, subj, &loc, params,
			finding.WithRelated(finding.Related{Location: other, Note: "collides with"}))
	}
	params["member"], params["other"] = finding.String(c.Member), finding.String(c.OtherMember)
	return finding.New(CodePathFileDirClash, subj, &loc, params,
		finding.WithRelated(finding.Related{Location: other, Note: "collides with"}))
}
