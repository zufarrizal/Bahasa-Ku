//go:build !js || !wasm
// +build !js !wasm

package evaluator

import (
	_ "modernc.org/sqlite"
)
