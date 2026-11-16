// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"reflect"
	"testing"
)

func TestErrorsString(t *testing.T) {
	e := Errors{{"a", 1, "x"}, {"a", 2, "y"}}
	if e.Error() != "a:1: x\na:2: y" {
		t.Fatalf("%q", e.Error())
	}
}

func TestErrorsAdd(t *testing.T) {
	var e Errors
	e.add("P", 3, "bad %s %d", "x", 1)
	want := Errors{{"P", 3, "bad x 1"}}
	if !reflect.DeepEqual(e, want) {
		t.Fatalf("got %+v", e)
	}
}

func TestErrorsSortByLine(t *testing.T) {
	e := Errors{{"P", 3, "c1"}, {"P", 1, "a"}, {"P", 3, "c2"}, {"P", 2, "b"}}
	e.sortByLine()
	want := Errors{{"P", 1, "a"}, {"P", 2, "b"}, {"P", 3, "c1"}, {"P", 3, "c2"}}
	if !reflect.DeepEqual(e, want) {
		t.Fatalf("got %+v", e)
	}
}
