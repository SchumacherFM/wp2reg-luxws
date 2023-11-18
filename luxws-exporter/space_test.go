package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormalizeSpace(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{
			input: "",
			want:  "",
		},
		{
			input: "    ",
			want:  "",
		},
		{
			input: "\t\n\t",
			want:  "",
		},
		{
			input: "foobar",
			want:  "foobar",
		},
		{
			input: " -   foo   -   bar   - ",
			want:  "- foo - bar -",
		},
		{
			input: "02.02.11 08:00:00",
			want:  "02.02.11 08:00:00",
		},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeSpace(tc.input)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("normalizeSpace(%q) difference (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}

var benchmarkNormalizeSpace string

// BenchmarkNormalizeSpace-4   	 1000000	      1019 ns/op	      56 B/op	       4 allocs/op with REGEXP => \s+ => " "
// BenchmarkNormalizeSpace-4   	 6844600	       168.5 ns/op	      24 B/op	       2 allocs/op no REGEXP
func BenchmarkNormalizeSpace(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkNormalizeSpace = normalizeSpace(" -   foo   -   bar   - ")
	}
}
