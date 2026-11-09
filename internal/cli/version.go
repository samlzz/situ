// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"runtime/debug"
	"slices"
)

func versionString(info *debug.BuildInfo, ok bool) string {
	if !ok || info == nil {
		return "situ (unknown)"
	}
	v := info.Main.Version
	if v == "" {
		v = "(devel)"
	}
	if v == "(devel)" {
		i := slices.IndexFunc(info.Settings, func(s debug.BuildSetting) bool {
			return s.Key == "vcs.revision" && s.Value != ""
		})
		if i >= 0 {
			return "situ " + v + " " + info.Settings[i].Value
		}
	}
	return "situ " + v
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	return versionString(info, ok)
}
