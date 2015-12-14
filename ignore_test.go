package dockerignore

import (
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
