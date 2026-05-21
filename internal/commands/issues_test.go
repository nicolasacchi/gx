package commands

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Fix the Login Bug!":          "fix-the-login-bug",
		"  Multiple   spaces & sym #": "multiple-spaces-sym",
		"ALLCAPS":                     "allcaps",
		"trailing---dashes---":        "trailing-dashes",
		"":                            "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q)=%q want %q", in, got, want)
		}
	}
	if got := slugify(strings.Repeat("a", 100)); len(got) > 50 {
		t.Errorf("slugify long: len=%d want <=50", len(got))
	}
}
