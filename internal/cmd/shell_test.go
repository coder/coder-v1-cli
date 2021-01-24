package cmd

import "testing"

func TestShellEscape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name    string
		Input   string
		Escaped string
	}{
		{
			Name:    "single space",
			Input:   "hello world",
			Escaped: `hello\ world`,
		},
		{
			Name:    "multiple spaces",
			Input:   "test message hello  world",
			Escaped: `test\ message\ hello\ \ world`,
		},
		{
			Name:    "mixed quotes",
			Input:   `"''"`,
			Escaped: `\"\'\'\"`,
		},
		{
			Name:    "mixed escaped quotes",
			Input:   `"'\"\"'"`,
			Escaped: `\"\'\\\"\\\"\'\"`,
		},
	}

	for _, test := range tests {
		if e, a := test.Escaped, shellEscape(test.Input); e != a {
			t.Fatalf("test %q failed; expected: %q, got %q (input: %q)",
				test.Name, test.Escaped, a, test.Input)
		}
	}
}
