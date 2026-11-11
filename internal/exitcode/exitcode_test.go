// SPDX-License-Identifier: GPL-3.0-or-later

package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestCodeOf(t *testing.T) {
	base := errors.New("boom")
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, OK},
		{"untyped", base, System},
		{"usage", New(Usage, "bad %s", "x"), Usage},
		{"drift", Wrap(Drift, base), Drift},
		{"wrapped by fmt", fmt.Errorf("ctx: %w", Wrap(Usage, base)), Usage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeOf(tt.err); got != tt.want {
				t.Fatalf("CodeOf = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	if Wrap(Usage, nil) != nil {
		t.Fatal("Wrap(nil) must be nil")
	}
	base := errors.New("boom")
	err := Wrap(Usage, base)
	if !errors.Is(err, base) {
		t.Fatal("Wrap must unwrap to the cause")
	}
	if err.Error() != "boom" {
		t.Fatalf("message = %q", err.Error())
	}
}

func TestNew(t *testing.T) {
	err := New(System, "n=%d", 3)
	if err.Error() != "n=3" || CodeOf(err) != System {
		t.Fatalf("got %q code %d", err.Error(), CodeOf(err))
	}
}
