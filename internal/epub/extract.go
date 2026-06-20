package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"ezpub/internal/lang"
)

func ExtractText(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	var files []*zip.File
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if strings.HasSuffix(strings.ToLower(name), ".xhtml") && strings.Contains(name, "/text/") && !strings.HasSuffix(name, "/cover.xhtml") {
			files = append(files, f)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	if len(files) == 0 {
		return "", fmt.Errorf(lang.ErrNoXHTMLText)
	}

	var out strings.Builder
	for _, f := range files {
		text, err := extractXHTMLText(f)
		if err != nil {
			return "", fmt.Errorf("%s: %w", f.Name, err)
		}
		out.WriteString(strings.TrimSpace(text))
		out.WriteString("\n\n")
	}
	return out.String(), nil
}

func extractXHTMLText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var out strings.Builder
	inBody := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if local(t.Name) == "body" {
				inBody = true
			}
			if inBody && local(t.Name) == "br" {
				out.WriteByte('\n')
			}
		case xml.EndElement:
			name := local(t.Name)
			if name == "body" {
				inBody = false
			}
			if inBody && (name == "p" || name == "h1" || name == "h2" || name == "h3" || name == "div" || name == "section") {
				out.WriteByte('\n')
			}
		case xml.CharData:
			if inBody {
				s := strings.TrimSpace(string(t))
				if s != "" {
					out.WriteString(s)
				}
			}
		}
	}
	return out.String(), nil
}

func local(name xml.Name) string {
	return strings.ToLower(name.Local)
}
