// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import "testing"

func TestRel(t *testing.T) {
	if got := Rel(id("a", "b")); got != "components/a/b" {
		t.Error(got)
	}
	if got := Rel(host("h")); got != "hosts/h" {
		t.Error(got)
	}
	r := Open("/r")
	if got := r.ModuleDir(host("h")); got != "/r/hosts/h" {
		t.Error(got)
	}
}
