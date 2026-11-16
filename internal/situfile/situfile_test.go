// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import "testing"

func TestEnumString(t *testing.T) {
	for typ, want := range map[EntryType]string{File: "file", Config: "config", Dir: "dir", Data: "data", 0: "invalid"} {
		if typ.String() != want {
			t.Errorf("EntryType %d: %q", typ, typ.String())
		}
	}
	for a, want := range map[UnitAction]string{Enable: "enable", Start: "start", 0: "invalid"} {
		if a.String() != want {
			t.Errorf("UnitAction %d: %q", a, a.String())
		}
	}
}

func TestModuleIDString(t *testing.T) {
	if s := (ModuleID{Component: "a"}).String(); s != "a" {
		t.Errorf("short: %q", s)
	}
	if s := (ModuleID{Category: "c", Component: "a"}).String(); s != "c/a" {
		t.Errorf("full: %q", s)
	}
}

func TestModuleIDIsProfile(t *testing.T) {
	for id, want := range map[ModuleID]bool{
		{Category: HostsDir, Component: "pve"}:      true,
		{Category: HostsDir}:                        true,
		{Component: HostsDir}:                       false,
		{Category: ComponentsDir, Component: "pve"}: false,
		{}: false,
	} {
		if got := id.IsProfile(); got != want {
			t.Errorf("%+v: %v", id, got)
		}
	}
}
