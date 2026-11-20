// SPDX-License-Identifier: GPL-3.0-or-later

package situfile

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestErrorsErr(t *testing.T) {
	var none Errors
	if err := none.Err(); err != nil {
		t.Fatalf("nil Errors: got %v", err)
	}
	if err := (Errors{}).Err(); err != nil {
		t.Fatalf("empty Errors: got %v", err)
	}
	e := Errors{{"b", 1, "b1"}, {"a", 2, "a2"}, {"a", 1, "a1"}, {"b", 1, "b1bis"}}
	got, ok := e.Err().(Errors)
	if !ok {
		t.Fatalf("Err() is not Errors: %T", e.Err())
	}
	want := Errors{{"a", 1, "a1"}, {"a", 2, "a2"}, {"b", 1, "b1"}, {"b", 1, "b1bis"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestErrorsCollect(t *testing.T) {
	var e Errors
	if err := e.Collect(nil); err != nil || len(e) != 0 {
		t.Fatalf("nil: err=%v e=%+v", err, e)
	}
	if err := e.Collect(Errors{{"a", 1, "x"}}); err != nil {
		t.Fatalf("Errors: err=%v", err)
	}
	wrapped := fmt.Errorf("ctx: %w", Errors{{"b", 2, "y"}})
	if err := e.Collect(wrapped); err != nil {
		t.Fatalf("wrapped: err=%v", err)
	}
	want := Errors{{"a", 1, "x"}, {"b", 2, "y"}}
	if !reflect.DeepEqual(e, want) {
		t.Fatalf("got %+v", e)
	}
	sys := errors.New("disk on fire")
	if err := e.Collect(sys); err != sys {
		t.Fatalf("system error: got %v", err)
	}
	if !reflect.DeepEqual(e, want) {
		t.Fatalf("system error changed e: %+v", e)
	}
}

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
