#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# release.sh <step> <work> runs one step of release.yml, which publishes a
# draft GitHub release that the release procedure created
# (docs/dev/release.md, ADR-013 D7 item 3). It runs from a whole clone of
# the repository whose origin is GitHub's, and keeps what one step hands
# the next in <work>, a directory prepare creates.
#
#   prepare    checks everything and pushes nothing. The draft is a draft,
#              its target is a commit's full SHA, its assets are exactly
#              SHA256SUMS and the shipped targets' archives, each with its
#              line's SHA-256, and the tool of this host's target reports
#              the version (release verify). The specification release the draft
#              names, if any, is a released commit that is an ancestor of
#              the target, and a prerelease names none. A release that is
#              not a prerelease implements only released revisions. No tag
#              it would push exists at another commit. It then makes the
#              annotated tags in this clone, and the tap's commit of the
#              generated formula in a clone of the tap in <work>.
#   push-tags  pushes the annotated v<version> tag, and spec-<M>.<m> when
#              the draft names it, to origin, in one atomic push.
#   publish    publishes the draft, marked a prerelease when the version
#              has a prerelease suffix. A release that is not one is
#              marked GitHub's latest release unless it is older than
#              the newest such release whose tag origin has.
#   push-tap   pushes the tap's commit to its main branch, with TAP_TOKEN,
#              unless the release does not update the tap: a prerelease
#              updates it only before a release that is not a prerelease
#              exists (ADR-013 D7 item 5), and a release older than the
#              newest one never moves the tap back (D6 item 1).
#
# A dry run (DRY_RUN=true) runs prepare whole, and every other step
# refuses: it is how the workflow is proved on a draft without publishing
# anything. A run that is not dry refuses, in prepare and before anything
# is pushed, a repository that is private or whose release environment
# requires no reviewer, since Ed's approval in that environment is the
# release act, and a release that updates the tap without TAP_TOKEN.
#
# The draft names the specification's release commit (ADR-011 D5 item 2)
# by one line of its body:
#
#   Specification release: spec-<M>.<m> <the commit's full SHA>
#
# Its environment: TAG, the draft's tag name v<version>; DRY_RUN, true or
# false; RELEASE_TOOL, the built internal/tools/release; GITHUB_REPOSITORY;
# ACTOR and ACTOR_ID, the GitHub account that dispatched the run, whose
# name and no-reply address tag the release and sign off the tap's commit;
# TAP_TOKEN_SET, true when the release environment holds TAP_TOKEN, and
# TAP_TOKEN itself for push-tap; and TAP_REPO, the tap's repository, which
# is kottera/homebrew-tap's on GitHub unless set. gh reads GH_TOKEN.
set -eu

usage() {
	echo "usage: .github/scripts/release.sh prepare|push-tags|publish|push-tap <work>" >&2
	exit 2
}

die() {
	echo "release: $*" >&2
	exit 1
}

note() {
	echo "release: $*"
	if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
		echo "- $*" >>"$GITHUB_STEP_SUMMARY"
	fi
}

[ "$#" -eq 2 ] || usage
step=$1
work=$2
case $work in
"") usage ;;
/*) ;;
*) work=$PWD/$work ;;
esac
: "${TAG:?TAG names the draft}" "${DRY_RUN:?DRY_RUN is true or false}"
case $DRY_RUN in
true | false) ;;
*) die "DRY_RUN is $DRY_RUN, not true or false" ;;
esac
TAP_REPO=${TAP_REPO:-https://github.com/kottera/homebrew-tap.git}

# commit_sha reports whether $1 is a commit's full SHA.
commit_sha() {
	printf '%s\n' "$1" | grep -Eq '^[0-9a-f]{40}$'
}

# remote_commit prints the commit the tag $1 names on origin, as listed in
# $work/remote-tags, or nothing when origin has no such tag.
remote_commit() {
	c=$(awk -v r="refs/tags/$1^{}" '$2 == r { print $1 }' "$work/remote-tags")
	[ -n "$c" ] || c=$(awk -v r="refs/tags/$1" '$2 == r { print $1 }' "$work/remote-tags")
	printf '%s' "$c"
}

prepare() {
	[ ! -e "$work" ] || die "$work exists; prepare starts from nothing"
	case $TAG in
	v[0-9]*) ;;
	*) die "the draft's tag, $TAG, is not v<version>" ;;
	esac
	version=${TAG#v}
	case $version in
	*[!0-9A-Za-z.-]*) die "the draft's tag, $TAG, is not v<version>" ;;
	esac
	: "${RELEASE_TOOL:?}" "${GITHUB_REPOSITORY:?}" "${ACTOR:?}" "${ACTOR_ID:?}"
	mkdir -p "$work/assets"

	draft=$(gh release view "$TAG" --json isDraft --jq .isDraft) ||
		die "there is no release $TAG"
	[ "$draft" = true ] || die "the release $TAG is published already; release.yml publishes a draft"
	target=$(gh release view "$TAG" --json targetCommitish --jq .targetCommitish)
	commit_sha "$target" ||
		die "the draft's target, $target, is not a commit's full SHA; the release procedure creates the draft with --target <the tagged commit's SHA>"
	git cat-file -e "$target^{commit}" 2>/dev/null || die "this repository holds no commit $target, the draft's target"

	gh release download "$TAG" --dir "$work/assets"
	case $(uname -s) in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) die "$(uname -s) runs no shipped target's metais" ;;
	esac
	case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) die "$(uname -m) runs no shipped target's metais" ;;
	esac
	implements=$("$RELEASE_TOOL" verify -version "$version" -dir "$work/assets" -run "$os/$arch") ||
		die "the draft's assets are refused"
	case $version in
	*-*) prerelease=true ;;
	*) prerelease=false ;;
	esac

	git ls-remote --tags origin >"$work/remote-tags"

	gh release view "$TAG" --json body --jq .body | tr -d '\r' >"$work/body"
	grep '^Specification release:' "$work/body" >"$work/spec-line" || true
	spec_tag=
	spec_commit=
	if [ -s "$work/spec-line" ]; then
		[ "$(wc -l <"$work/spec-line")" -eq 1 ] || die "the draft names more than one specification release"
		[ "$(awk '{ print NF }' "$work/spec-line")" -eq 4 ] ||
			die "the draft's line \"$(cat "$work/spec-line")\" is not \"Specification release: spec-<M>.<m> <commit>\""
		spec_tag=$(awk '{ print $3 }' "$work/spec-line")
		spec_commit=$(awk '{ print $4 }' "$work/spec-line")
		printf '%s\n' "$spec_tag" | grep -Eq '^spec-(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' ||
			die "the draft names $spec_tag, which is not spec-<M>.<m>"
		commit_sha "$spec_commit" || die "the draft names $spec_tag at $spec_commit, which is not a commit's full SHA"
	fi
	spec_rev=${spec_tag#spec-}

	if [ -n "$spec_tag" ]; then
		[ "$prerelease" = false ] ||
			die "the draft of the prerelease $TAG names $spec_tag; a prerelease never carries a specification release (ADR-011 A8 item 5)"
		git cat-file -e "$spec_commit^{commit}" 2>/dev/null || die "this repository holds no commit $spec_commit, where the draft names $spec_tag"
		git merge-base --is-ancestor "$spec_commit" "$target" ||
			die "$spec_commit, where the draft names $spec_tag, is not an ancestor of the draft's target $target"
		git cat-file -e "$spec_commit:spec/conformance/released/$spec_rev.sha256" 2>/dev/null ||
			die "$spec_commit, where the draft names $spec_tag, holds no spec/conformance/released/$spec_rev.sha256, so it is not the revision's release commit (ADR-011 D5 item 2)"
		printf '%s\n' "$implements" | grep -Fqx "$spec_rev" ||
			die "the draft names $spec_tag, and the release's tool implements only $(echo "$implements" | tr '\n' ' ')"
	fi
	# shellcheck disable=SC2086 # one revision M.m per line
	for rev in $implements; do
		[ "$rev" != "$spec_rev" ] || continue
		c=$(remote_commit "spec-$rev")
		if [ -z "$c" ]; then
			[ "$prerelease" = true ] ||
				die "the release $TAG implements $rev, which is not released: origin has no spec-$rev and the draft names none (ADR-011 D5 item 6)"
			note "the prerelease $TAG implements $rev, which is not released (ADR-011 A8 item 5)"
		elif ! git merge-base --is-ancestor "$c" "$target"; then
			die "the release $TAG implements $rev, and spec-$rev is not on its target's history"
		fi
	done

	push_v=true
	c=$(remote_commit "$TAG")
	if [ -n "$c" ]; then
		[ "$c" = "$target" ] || die "origin has $TAG already, at $c, not at the draft's target $target; a tag never moves"
		push_v=false
		note "origin has $TAG at $target already; it is not pushed again"
	fi
	push_spec=false
	if [ -n "$spec_tag" ]; then
		push_spec=true
		c=$(remote_commit "$spec_tag")
		if [ -n "$c" ]; then
			[ "$c" = "$spec_commit" ] || die "origin has $spec_tag already, at $c, not at $spec_commit; a tag never moves"
			push_spec=false
			note "origin has $spec_tag at $spec_commit already; it is not pushed again"
		fi
	fi

	# The tap, and GitHub's latest release, which install.sh installs,
	# follow the newest release that is not a prerelease (ADR-013 D6 item 1,
	# D7 item 5). A prerelease updates the tap only while origin has no
	# such release's tag, and a release older than the newest one already
	# tagged updates neither.
	newest=$(grep -E '[[:space:]]refs/tags/v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' "$work/remote-tags" |
		sed 's,.*refs/tags/v,,' | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)
	tap=true
	latest=
	tap_why=
	if [ "$prerelease" = true ]; then
		if [ -n "$newest" ]; then
			tap=false
			tap_why="$TAG is a prerelease, and the release v$newest is not one"
		fi
	else
		latest=true
		if [ -n "$newest" ] &&
			[ "$(printf '%s\n%s\n' "$newest" "$version" | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)" != "$version" ]; then
			tap=false
			latest=false
			tap_why="$TAG is older than v$newest, the newest release that is not a prerelease"
		fi
	fi

	private=$(gh api "repos/$GITHUB_REPOSITORY" --jq .private)
	reviewers=$(gh api "repos/$GITHUB_REPOSITORY/environments/release" \
		--jq '[.protection_rules[]? | select(.type == "required_reviewers")] | length' 2>/dev/null) || reviewers=0
	if [ "$DRY_RUN" = false ]; then
		[ "$private" = false ] ||
			die "$GITHUB_REPOSITORY is private, where GitHub's free plan enforces no required reviewer, so nothing approved this run; make it public first (docs/dev/release.md)"
		[ "$reviewers" -gt 0 ] ||
			die "the release environment requires no reviewer, so nothing approved this run; add Ed as its required reviewer (docs/dev/release.md)"
		[ "$tap" = false ] || [ "${TAP_TOKEN_SET:-}" = true ] ||
			die "the release environment holds no TAP_TOKEN, and $TAG updates the tap"
	else
		note "dry run: $GITHUB_REPOSITORY is private: $private; required reviewer rules on the release environment: $reviewers; TAP_TOKEN held: ${TAP_TOKEN_SET:-false}"
	fi

	name=$(gh api "users/$ACTOR" --jq '.name // ""' 2>/dev/null) || name=
	[ -n "$name" ] || name=$ACTOR
	email="$ACTOR_ID+$ACTOR@users.noreply.github.com"

	if [ "$push_v" = true ]; then
		git -c user.name="$name" -c user.email="$email" tag -a -m "metais $version" "$TAG" "$target"
	fi
	if [ "$push_spec" = true ]; then
		git -c user.name="$name" -c user.email="$email" tag -a -m "METAIS specification $spec_rev" "$spec_tag" "$spec_commit"
	fi

	if [ "$tap" = true ]; then
		"$RELEASE_TOOL" formula -version "$version" -sums "$work/assets/SHA256SUMS" >"$work/metais.rb" ||
			die "the formula of $TAG is refused"
		git clone --quiet "$TAP_REPO" "$work/tap"
		mkdir -p "$work/tap/Formula"
		cp "$work/metais.rb" "$work/tap/Formula/metais.rb"
		git -C "$work/tap" add Formula/metais.rb
		printf 'metais %s\n\nGenerated from the SHA256SUMS of the release v%s of kottera/metais\nby its go run ./internal/tools/release formula.\n' \
			"$version" "$version" >"$work/tap-message"
		git -C "$work/tap" -c user.name="$name" -c user.email="$email" commit --quiet -s -F "$work/tap-message"
		git -C "$work/tap" show --stat --format=fuller HEAD
	fi

	cat >"$work/state" <<EOF
dry_run=$DRY_RUN
version=$version
target=$target
prerelease=$prerelease
push_v=$push_v
spec_tag=$spec_tag
spec_commit=$spec_commit
push_spec=$push_spec
tap=$tap
latest=$latest
EOF
	if [ -n "${GITHUB_OUTPUT:-}" ]; then
		echo "tap=$tap" >>"$GITHUB_OUTPUT"
	fi

	kind=release
	[ "$prerelease" = false ] || kind=prerelease
	note "$TAG at $target is checked: a $kind, whose assets are the release's"
	[ "$push_v" = false ] || note "push-tags pushes the annotated tag $TAG at $target"
	[ "$push_spec" = false ] || note "push-tags pushes the annotated tag $spec_tag at $spec_commit"
	note "publish publishes the draft $TAG as a $kind"
	[ "$latest" != false ] || note "publish does not mark $TAG GitHub's latest release, which v$newest is"
	if [ "$tap" = true ]; then
		note "push-tap pushes Formula/metais.rb of $version to $TAP_REPO's main"
	else
		note "the tap is not updated: $tap_why"
	fi
	[ "$DRY_RUN" = false ] || note "dry run: nothing is pushed or published"
}

# load reads prepare's state, and refuses a dry run.
load() {
	[ -f "$work/state" ] || die "$work holds no state; prepare runs first"
	dry_run=true
	# shellcheck source=/dev/null
	. "$work/state"
	if [ "$DRY_RUN" != false ] || [ "$dry_run" != false ]; then
		die "this is a dry run, and $step pushes; a dry run pushes nothing"
	fi
}

push_tags() {
	load
	set --
	[ "$push_v" = false ] || set -- "$@" "refs/tags/$TAG"
	[ "$push_spec" = false ] || set -- "$@" "refs/tags/$spec_tag"
	if [ "$#" -eq 0 ]; then
		note "origin has every tag of $TAG already"
		return
	fi
	git push --atomic origin "$@"
	note "pushed $*"
}

publish() {
	load
	set -- --draft=false --prerelease="$prerelease"
	[ -z "$latest" ] || set -- "$@" --latest="$latest"
	gh release edit "$TAG" "$@" --verify-tag >/dev/null
	draft=$(gh release view "$TAG" --json isDraft --jq .isDraft)
	[ "$draft" = false ] || die "the release $TAG is still a draft"
	note "published $TAG"
}

push_tap() {
	load
	if [ "$tap" = false ]; then
		note "the tap is not updated"
		return
	fi
	: "${TAP_TOKEN:?TAP_TOKEN pushes to the tap}"
	case $TAP_REPO in
	https://github.com/*) url="https://x-access-token:$TAP_TOKEN@${TAP_REPO#https://}" ;;
	*) url=$TAP_REPO ;;
	esac
	git -C "$work/tap" push --quiet "$url" HEAD:refs/heads/main
	note "pushed Formula/metais.rb of $version to the tap"
}

case $step in
prepare) prepare ;;
push-tags) push_tags ;;
publish) publish ;;
push-tap) push_tap ;;
*) usage ;;
esac
