// SPDX-License-Identifier: GPL-3.0-or-later

// Package help holds the embedded help topics.
package help

import (
	"embed"
	"strings"
)

//go:embed topics/*.txt
var topics embed.FS

const stubText = "not implemented"

func Topics() []string {
	entries, err := topics.ReadDir("topics")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".txt"))
	}
	return names
}

func Topic(name string) (string, bool) {
	if strings.ContainsAny(name, "/.") {
		return "", false
	}
	b, err := topics.ReadFile("topics/" + name + ".txt")
	if err != nil {
		return "", false
	}
	return string(b), true
}

func IsStub(name string) bool {
	text, ok := Topic(name)
	return ok && strings.TrimSpace(text) == stubText
}
