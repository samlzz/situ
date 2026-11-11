// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"runtime/debug"
	"testing"
)

func TestVersionString(t *testing.T) {
	rev := []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "true"}}
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"tagged", &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}, Settings: rev}, true, "situ v0.1.0"},
		{"devel with revision", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: rev}, true, "situ (devel) abc123"},
		{"devel without revision", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "situ (devel)"},
		{"empty version", &debug.BuildInfo{}, true, "situ (devel)"},
		{"pseudo-version as is", &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261101120000-abc123def456+dirty"}, Settings: rev}, true,
			"situ v0.0.0-20261101120000-abc123def456+dirty"},
		{"no build info", nil, false, "situ (unknown)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionString(tt.info, tt.ok); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
