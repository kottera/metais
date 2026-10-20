# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# git-env.sh is sourced by the gate's scripts before they run git, so that
# nothing the host configures changes what git does for them, or what the
# go command's git does under them: every variable of git's own (GIT_*) the
# caller set is dropped, no system or global configuration or attributes
# file is read, no excludes file but the repository's own is read, and
# core.precomposeunicode is off. internal/testenv's GitEnv gives the git a
# test runs the same environment.
#
# The last is what a checkout needs on darwin, where git writes
# core.precomposeunicode=true into every repository it creates, a clone
# included. With it on, git converts each non-ASCII name it reads from a
# directory through iconv's UTF-8-MAC converter, and dies where iconv cannot
# open it. Variables given this way outrank every configuration file, the
# repository's own included.
#
# It is POSIX sh, because a script that sources it may run under a /bin/sh
# that is dash. The names come from env, whose every variable begins a line
# NAME=. A line inside a multi-line value can look like one too; unsetting
# a name that is not set does nothing.

for v in $(env | sed -n 's/^\(GIT_[A-Za-z0-9_]*\)=.*/\1/p'); do
	unset "$v"
done
export GIT_CONFIG_NOSYSTEM=1 GIT_ATTR_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_COUNT=3
export GIT_CONFIG_KEY_0=core.precomposeunicode GIT_CONFIG_VALUE_0=false
export GIT_CONFIG_KEY_1=core.excludesFile GIT_CONFIG_VALUE_1=/dev/null
export GIT_CONFIG_KEY_2=core.attributesFile GIT_CONFIG_VALUE_2=/dev/null
unset v
