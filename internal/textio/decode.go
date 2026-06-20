package textio

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"ezpub/internal/lang"
)

func DecodeFile(path, encodingName string) (text string, detected string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return Decode(data, encodingName)
}

func Decode(data []byte, encodingName string) (text string, detected string, err error) {
	name := strings.ToLower(strings.TrimSpace(encodingName))
	if name == "" {
		name = "auto"
	}

	if name != "auto" {
		text, err := decodeAs(data, name)
		return text, name, err
	}

	switch {
	case bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}):
		return string(data[3:]), "utf-8-bom", nil
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
		text, err := decodeAs(data, "utf-16le")
		return text, "utf-16le", err
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		text, err := decodeAs(data, "utf-16be")
		return text, "utf-16be", err
	case utf8.Valid(data):
		return string(data), "utf-8", nil
	default:
		text, err := decodeAs(data, "gb18030")
		return text, "gb18030", err
	}
}

func decodeAs(data []byte, name string) (string, error) {
	switch strings.ToLower(name) {
	case "utf-8", "utf8":
		if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
			data = data[3:]
		}
		if !utf8.Valid(data) {
			return "", fmt.Errorf(lang.ErrInvalidUTF8)
		}
		return string(data), nil
	case "gbk", "cp936":
		return transformOnly(simplifiedchinese.GBK.NewDecoder(), data)
	case "gb18030":
		return transformOnly(simplifiedchinese.GB18030.NewDecoder(), data)
	case "utf-16le":
		return transformOnly(textunicode.UTF16(textunicode.LittleEndian, textunicode.ExpectBOM).NewDecoder(), data)
	case "utf-16be":
		return transformOnly(textunicode.UTF16(textunicode.BigEndian, textunicode.ExpectBOM).NewDecoder(), data)
	default:
		return "", fmt.Errorf(lang.ErrUnsupportedEncoding, name)
	}
}

func transformOnly(t transform.Transformer, data []byte) (string, error) {
	s, _, err := transform.String(t, string(data))
	return s, err
}
