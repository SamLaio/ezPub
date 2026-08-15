package textio

import (
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

func TestDecodeAutoUTF8(t *testing.T) {
	got, enc, err := Decode([]byte("第一章 測試"), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if enc != "utf-8" || got != "第一章 測試" {
		t.Fatalf("got %q/%s", got, enc)
	}
}

func TestDecodeAutoGB18030(t *testing.T) {
	var b strings.Builder
	w := transform.NewWriter(&b, simplifiedchinese.GB18030.NewEncoder())
	if _, err := w.Write([]byte("第一章 测试")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, enc, err := Decode([]byte(b.String()), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if enc != "gb18030" || got != "第一章 测试" {
		t.Fatalf("got %q/%s", got, enc)
	}
}

func TestDecodeExplicitBig5(t *testing.T) {
	var b strings.Builder
	w := transform.NewWriter(&b, traditionalchinese.Big5.NewEncoder())
	if _, err := w.Write([]byte("第一章 測試")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	got, enc, err := Decode([]byte(b.String()), "big5")
	if err != nil {
		t.Fatal(err)
	}
	if enc != "big5" || got != "第一章 測試" {
		t.Fatalf("got %q/%s", got, enc)
	}
}
