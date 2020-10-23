// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httppattern

import (
	"testing"
)

func TestWildcardMatcherForPath(t *testing.T) {
	testData := []struct {
		desc        string
		uri         string
		wantMatcher string
	}{
		{
			desc:        "No path params",
			uri:         "/shelves",
			wantMatcher: "",
		},
		{
			desc:        "Path params with fieldpath-only bindings",
			uri:         "/shelves/{shelf_id}/books/{book.id}",
			wantMatcher: `^/shelves/[^\/]+/books/[^\/]+$`,
		},
		{
			desc:        "Path params with fieldpath-only bindings and verb",
			uri:         "/shelves/{shelf_id}/books/{book.id}:checkout",
			wantMatcher: `^/shelves/[^\/]+/books/[^\/]+:checkout$`,
		},
		{
			desc:        "Path param with wildcard segments",
			uri:         "/test/*/test/**",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path param with wildcard segments and verb",
			uri:         "/test/*/test/**:upload",
			wantMatcher: `^/test/[^\/]+/test/.*:upload$`,
		},
		{
			desc:        "Path param with wildcard in segment binding",
			uri:         "/test/{x=*}/test/{y=**}",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path param with mixed wildcards",
			uri:         "/test/{name=*}/test/**",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Invalid http template, not preceded by '/' ",
			uri:         "**",
			wantMatcher: "",
		},
		{
			desc:        "Path params with full segment binding",
			uri:         "/v1/{name=books/*}",
			wantMatcher: `^/v1/books/[^\/]+$`,
		},
		{
			desc:        "Path params with multiple field path segment bindings",
			uri:         "/v1/{test=a/b/*}/route/{resource_id=shelves/*/books/**}:upload",
			wantMatcher: `^/v1/a/b/[^\/]+/route/shelves/[^\/]+/books/.*:upload$`,
		},
		{
			// TODO(nareddyt): How can we improve validation once we remove path matcher?
			desc:        "BUG - Incorrect http template syntax is not validated",
			uri:         "/v1/{name=/books/*}",
			wantMatcher: `^/v1//books/[^\/]+$`,
		},
	}

	for _, tc := range testData {
		got := WildcardMatcherForPath(tc.uri)

		if tc.wantMatcher != got {
			t.Errorf("Test (%v): \n got %v \nwant %v", tc.desc, got, tc.wantMatcher)
		}
	}
}