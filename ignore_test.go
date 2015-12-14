package dockerignore

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReadIgnoreFile(t *testing.T) {
	Convey("Read .gitignore should be OK", t, func() {
		patterns, err := ReadIgnoreFile(".gitignore")
		So(err, ShouldBeNil)

		for _, name := range []string{"ignore.exe", "6.out", "ignore.so"} {
			isSkip, err := Matches(name, patterns)
			Convey("Should ignore "+name, func() {
				So(err, ShouldBeNil)
				So(isSkip, ShouldEqual, true)
			})
		}

		for _, name := range []string{"ok.go", "ok.py", "ok.exe", "ok.a"} {
			Convey("Should include "+name, func() {
				isSkip, err := Matches(name, patterns)
				So(err, ShouldBeNil)
				So(isSkip, ShouldEqual, false)
			})
		}
	})
}

// Test lots of variants of patterns & strings
func TestMatches(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		pass    bool
	}{
		{"**", "file", true},
		{"**", "file/", true},
		{"**/", "file", true}, // weird one
		{"**/", "file/", true},
		{"**", "/", true},
		{"**/", "/", true},
		{"**", "dir/file", true},
		{"**/", "dir/file", false},
		{"**", "dir/file/", true},
		{"**/", "dir/file/", true},
		{"**/**", "dir/file", true},
		{"**/**", "dir/file/", true},
		{"dir/**", "dir/file", true},
		{"dir/**", "dir/file/", true},
		{"dir/**", "dir/dir2/file", true},
		{"dir/**", "dir/dir2/file/", true},
		{"**/dir2/*", "dir/dir2/file", true},
		{"**/dir2/*", "dir/dir2/file/", false},
		{"**/dir2/**", "dir/dir2/dir3/file", true},
		{"**/dir2/**", "dir/dir2/dir3/file/", true},
		{"**file", "file", true},
		{"**file", "dir/file", true},
		{"**/file", "dir/file", true},
		{"**file", "dir/dir/file", true},
		{"**/file", "dir/dir/file", true},
		{"**/file*", "dir/dir/file", true},
		{"**/file*", "dir/dir/file.txt", true},
		{"**/file*txt", "dir/dir/file.txt", true},
		{"**/file*.txt", "dir/dir/file.txt", true},
		{"**/file*.txt*", "dir/dir/file.txt", true},
		{"**/**/*.txt", "dir/dir/file.txt", true},
		{"**/**/*.txt2", "dir/dir/file.txt", false},
		{"**/*.txt", "file.txt", true},
		{"**/**/*.txt", "file.txt", true},
		{"a**/*.txt", "a/file.txt", true},
		{"a**/*.txt", "a/dir/file.txt", true},
		{"a**/*.txt", "a/dir/dir/file.txt", true},
		{"a/*.txt", "a/dir/file.txt", false},
		{"a/*.txt", "a/file.txt", true},
		{"a/*.txt**", "a/file.txt", true},
		{"a[b-d]e", "ae", false},
		{"a[b-d]e", "ace", true},
		{"a[b-d]e", "aae", false},
		{"a[^b-d]e", "aze", true},
		{".*", ".foo", true},
		{".*", "foo", false},
		{"abc.def", "abcdef", false},
		{"abc.def", "abc.def", true},
		{"abc.def", "abcZdef", false},
		{"abc?def", "abcZdef", true},
		{"abc?def", "abcdef", false},
		{"a\\*b", "a*b", true},
		{"a\\", "a", false},
		{"a\\", "a\\", false},
		{"a\\\\", "a\\", true},
		{"**/foo/bar", "foo/bar", true},
		{"**/foo/bar", "dir/foo/bar", true},
		{"**/foo/bar", "dir/dir2/foo/bar", true},
		{"abc/**", "abc", false},
		{"abc/**", "abc/def", true},
		{"abc/**", "abc/def/ghi", true},
	}

	for _, test := range tests {
		res, _ := regexpMatch(test.pattern, test.text)
		if res != test.pass {
			t.Fatalf("Failed: %v - res:%v", test, res)
		}
	}
}

// An empty string should return true from Empty.
func TestEmpty(t *testing.T) {
	empty := empty("")
	if !empty {
		t.Errorf("failed to get true for an empty string, got %v", empty)
	}
}

func TestCleanPatterns(t *testing.T) {
	cleaned, _, _, _ := cleanPatterns([]string{"docs", "config"})
	if len(cleaned) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(cleaned))
	}
}

func TestCleanPatternsStripEmptyPatterns(t *testing.T) {
	cleaned, _, _, _ := cleanPatterns([]string{"docs", "config", ""})
	if len(cleaned) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(cleaned))
	}
}

func TestCleanPatternsExceptionFlag(t *testing.T) {
	_, _, exceptions, _ := cleanPatterns([]string{"docs", "!docs/README.md"})
	if !exceptions {
		t.Errorf("expected exceptions to be true, got %v", exceptions)
	}
}

func TestCleanPatternsLeadingSpaceTrimmed(t *testing.T) {
	_, _, exceptions, _ := cleanPatterns([]string{"docs", "  !docs/README.md"})
	if !exceptions {
		t.Errorf("expected exceptions to be true, got %v", exceptions)
	}
}

func TestCleanPatternsTrailingSpaceTrimmed(t *testing.T) {
	_, _, exceptions, _ := cleanPatterns([]string{"docs", "!docs/README.md  "})
	if !exceptions {
		t.Errorf("expected exceptions to be true, got %v", exceptions)
	}
}

func TestCleanPatternsErrorSingleException(t *testing.T) {
	_, _, _, err := cleanPatterns([]string{"!"})
	if err == nil {
		t.Errorf("expected error on single exclamation point, got %v", err)
	}
}

func TestCleanPatternsFolderSplit(t *testing.T) {
	_, dirs, _, _ := cleanPatterns([]string{"docs/config/CONFIG.md"})
	if dirs[0][0] != "docs" {
		t.Errorf("expected first element in dirs slice to be docs, got %v", dirs[0][1])
	}
	if dirs[0][1] != "config" {
		t.Errorf("expected first element in dirs slice to be config, got %v", dirs[0][1])
	}
}

// These matchTests are stolen from go's filepath Match tests.
type matchTest struct {
	pattern, s string
	match      bool
	err        error
}

var matchTests = []matchTest{
	{"abc", "abc", true, nil},
	{"*", "abc", true, nil},
	{"*c", "abc", true, nil},
	{"a*", "a", true, nil},
	{"a*", "abc", true, nil},
	{"a*", "ab/c", false, nil},
	{"a*/b", "abc/b", true, nil},
	{"a*/b", "a/c/b", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
	{"ab[c]", "abc", true, nil},
	{"ab[b-d]", "abc", true, nil},
	{"ab[e-g]", "abc", false, nil},
	{"ab[^c]", "abc", false, nil},
	{"ab[^b-d]", "abc", false, nil},
	{"ab[^e-g]", "abc", true, nil},
	{"a\\*b", "a*b", true, nil},
	{"a\\*b", "ab", false, nil},
	{"a?b", "a☺b", true, nil},
	{"a[^a]b", "a☺b", true, nil},
	{"a???b", "a☺b", false, nil},
	{"a[^a][^a][^a]b", "a☺b", false, nil},
	{"[a-ζ]*", "α", true, nil},
	{"*[a-ζ]", "A", false, nil},
	{"a?b", "a/b", false, nil},
	{"a*b", "a/b", false, nil},
	{"[\\]a]", "]", true, nil},
	{"[\\-]", "-", true, nil},
	{"[x\\-]", "x", true, nil},
	{"[x\\-]", "-", true, nil},
	{"[x\\-]", "z", false, nil},
	{"[\\-x]", "x", true, nil},
	{"[\\-x]", "-", true, nil},
	{"[\\-x]", "a", false, nil},
	{"[]a]", "]", false, filepath.ErrBadPattern},
	{"[-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "x", false, filepath.ErrBadPattern},
	{"[x-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "z", false, filepath.ErrBadPattern},
	{"[-x]", "x", false, filepath.ErrBadPattern},
	{"[-x]", "-", false, filepath.ErrBadPattern},
	{"[-x]", "a", false, filepath.ErrBadPattern},
	{"\\", "a", false, filepath.ErrBadPattern},
	{"[a-b-c]", "a", false, filepath.ErrBadPattern},
	{"[", "a", false, filepath.ErrBadPattern},
	{"[^", "a", false, filepath.ErrBadPattern},
	{"[^bc", "a", false, filepath.ErrBadPattern},
	{"a[", "a", false, filepath.ErrBadPattern}, // was nil but IMO its wrong
	{"a[", "ab", false, filepath.ErrBadPattern},
	{"*x", "xxx", true, nil},
}

func errp(e error) string {
	if e == nil {
		return "<nil>"
	}
	return e.Error()
}

// TestMatch test's our version of filepath.Match, called regexpMatch.
func TestMatch(t *testing.T) {
	for _, tt := range matchTests {
		pattern := tt.pattern
		s := tt.s
		if runtime.GOOS == "windows" {
			if strings.Index(pattern, "\\") >= 0 {
				// no escape allowed on windows.
				continue
			}
			pattern = filepath.Clean(pattern)
			s = filepath.Clean(s)
		}
		ok, err := regexpMatch(pattern, s)
		if ok != tt.match || err != tt.err {
			t.Fatalf("Match(%#q, %#q) = %v, %q want %v, %q", pattern, s, ok, errp(err), tt.match, errp(tt.err))
		}
	}
}
