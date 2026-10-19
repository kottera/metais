// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package yamlx keeps only-importers' exception: it may import the YAML
// library.
package yamlx

import "go.yaml.in/yaml/v3"

// Node is a node of the library's tree.
var Node yaml.Node
