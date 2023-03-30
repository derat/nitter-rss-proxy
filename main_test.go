// Copyright 2023 Daniel Erat.
// All rights reserved.

package main

import (
	"net/url"
	"testing"
)

func TestRewriteContent(t *testing.T) {
	for _, tc := range []struct {
		loc, orig, want string
	}{
		{
			`https://nitter.1d4.us/user/status/123`,
			`<img src="https://nitter.1d4.us/pic/enc/bWVkaWEvRm1EaXZmTFhrQUlnREFYLmpwZw==" style="max-width:250px;" />`,
			`<img src="https://pbs.twimg.com/media/FmDivfLXkAIgDAX?format=jpg" style="max-width:250px;" />`,
		},
		{
			`https://nitter.net/user/status/123`,
			`<a href="https://nitter.net/foo/status/12345">nitter.net/foo/status/123…</a>`,
			`<a href="https://twitter.com/foo/status/12345">twitter.com/foo/status/123…</a>`,
		},
		{
			`https://nitter.net/user/status/123`,
			`<a href="https://nitter.net/foo/status/12345#m">nitter.net/foo/status/123…</a>`,
			`<a href="https://twitter.com/foo/status/12345">twitter.com/foo/status/123…</a>`,
		},
		{
			`https://nitter.net/user/status/123`,
			`<a href="https://nitter.net/i/web/status/12345">nitter.net/i/web/status/123…</a>`,
			`<a href="https://twitter.com/i/web/status/12345">twitter.com/i/web/status/123…</a>`,
		},
		{
			`https://nitter.mask.sh/user/status/123`,
			`<p></p><img src="https://nitter.mask.sh/pic/media%2FArpx24jXoAUzkc9.jpg" style="max-width:250px;" />`,
			`<p></p><img src="https://pbs.twimg.com/media/Arpx24jXoAUzkc9?format=jpg" style="max-width:250px;" />`,
		},
		{
			`https://nitter.kylrth.com/user/status/123`,
			`<p>Launch update: <a href="http://nitter.kylrth.com/NASA" title="NASA">@NASA</a> and ` +
				`<a href="http://nitter.kylrth.com/BoeingSpace" title="Boeing Space">@BoeingSpace</a>`,
			`<p>Launch update: <a href="https://twitter.com/NASA" title="NASA">@NASA</a> and ` +
				`<a href="https://twitter.com/BoeingSpace" title="Boeing Space">@BoeingSpace</a>`,
		},
		{
			`https://nitter.kylrth.com/user/status/123`,
			`The CST-100 <a href="http://nitter.kylrth.com/search?q=%23Starliner">#Starliner</a> flight`,
			`The CST-100 <a href="https://twitter.com/search?q=%23Starliner">#Starliner</a> flight`,
		},
		// TODO: Add more tests if I feel like it.
	} {
		loc, err := url.Parse(tc.loc)
		if err != nil {
			t.Error("Failed parsing location:", err)
		} else if got, err := rewriteContent(tc.orig, loc); err != nil {
			t.Errorf("rewriteContent(%q, %q) failed: %v", tc.orig, tc.loc, err)
		} else if got != tc.want {
			t.Errorf("rewriteContent(%q, %q) = %q; want %q", tc.orig, tc.loc, got, tc.want)
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
