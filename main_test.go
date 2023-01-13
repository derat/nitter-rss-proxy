// Copyright 2023 Daniel Erat.
// All rights reserved.

package main

import (
	"testing"
)

func TestRewriteContent(t *testing.T) {
	for _, tc := range []struct {
		orig, want string
	}{
		{
			`<img src="https://nitter.1d4.us/pic/enc/bWVkaWEvRm1EaXZmTFhrQUlnREFYLmpwZw==" style="max-width:250px;" />`,
			`<img src="https://pbs.twimg.com/media/FmDivfLXkAIgDAX?format=jpg" style="max-width:250px;" />`,
		},
		{
			`<a href="https://nitter.net/foo/status/12345">nitter.net/foo/status/123…</a>`,
			`<a href="https://twitter.com/foo/status/12345">twitter.com/foo/status/123…</a>`,
		},
		{
			`<a href="https://nitter.net/foo/status/12345#m">nitter.net/foo/status/123…</a>`,
			`<a href="https://twitter.com/foo/status/12345">twitter.com/foo/status/123…</a>`,
		},
		{
			`<a href="https://nitter.net/i/web/status/12345">nitter.net/i/web/status/123…</a>`,
			`<a href="https://twitter.com/i/web/status/12345">twitter.com/i/web/status/123…</a>`,
		},
		{
			`<p></p><img src="https://nitter.mask.sh/pic/media%2FArpx24jXoAUzkc9.jpg" style="max-width:250px;" />`,
			`<p></p><img src="https://pbs.twimg.com/media/Arpx24jXoAUzkc9?format=jpg" style="max-width:250px;" />`,
		},
		// TODO: Add more tests if I feel like it.
	} {
		if got, err := rewriteContent(tc.orig); err != nil {
			t.Errorf("rewriteContent(%q) failed: %v", tc.orig, err)
		} else if got != tc.want {
			t.Errorf("rewriteContent(%q) = %q; want %q", tc.orig, got, tc.want)
		}
	}
}

func TestRewriteIconURL(t *testing.T) {
	for _, tc := range []struct {
		orig, want string
	}{
		{
			`http://example.org/pic%2Fprofile_images%2F1591604213976530946%2F0CF-Esuh_400x400.jpg`,
			`https://pbs.twimg.com/profile_images/1591604213976530946/0CF-Esuh_400x400.jpg`,
		},
		{
			`http://example.org/pic/pbs.twimg.com%2Fprofile_images%2F1591604213976530946%2F0CF-Esuh_400x400.jpg`,
			`https://pbs.twimg.com/profile_images/1591604213976530946/0CF-Esuh_400x400.jpg`,
		},
	} {
		if got := rewriteIconURL(tc.orig); got != tc.want {
			t.Errorf("rewriteIconURL(%q) = %q; want %q", tc.orig, got, tc.want)
		}
	}
}
