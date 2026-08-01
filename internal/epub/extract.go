package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
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

	filesByName := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		filesByName[name] = f
	}

	var files []*zip.File
	for _, name := range spineTextFiles(zr.File, filesByName) {
		if f := filesByName[name]; f != nil {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		for _, f := range zr.File {
			name := filepath.ToSlash(f.Name)
			if isTextDocument(name) && !isNonContentDocument(name) {
				files = append(files, f)
			}
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	}
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

type containerRootfiles struct {
	Rootfiles []containerRootfile `xml:"rootfiles>rootfile"`
}

type containerRootfile struct {
	FullPath string `xml:"full-path,attr"`
}

type opfPackage struct {
	Manifest []opfItem    `xml:"manifest>item"`
	Spine    []opfItemref `xml:"spine>itemref"`
}

type opfItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type opfItemref struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

func spineTextFiles(zipFiles []*zip.File, filesByName map[string]*zip.File) []string {
	opfName := rootfilePath(filesByName["META-INF/container.xml"])
	if opfName == "" {
		return nil
	}
	opfFile := filesByName[opfName]
	if opfFile == nil {
		return nil
	}
	pkg, err := readOPF(opfFile)
	if err != nil {
		return nil
	}
	opfDir := path.Dir(opfName)
	if opfDir == "." {
		opfDir = ""
	}
	items := make(map[string]opfItem, len(pkg.Manifest))
	for _, item := range pkg.Manifest {
		items[item.ID] = item
	}
	var ordered []string
	seen := map[string]bool{}
	for _, itemref := range pkg.Spine {
		item := items[itemref.IDRef]
		if item.Href == "" || strings.EqualFold(itemref.Linear, "no") {
			continue
		}
		if !isXHTMLManifestItem(item) {
			continue
		}
		name := cleanZipPath(path.Join(opfDir, item.Href))
		if name == "" || filesByName[name] == nil || seen[name] || isNonContentDocument(name) {
			continue
		}
		ordered = append(ordered, name)
		seen[name] = true
	}
	return ordered
}

func rootfilePath(f *zip.File) string {
	if f == nil {
		return ""
	}
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	var c containerRootfiles
	if err := xml.Unmarshal(data, &c); err != nil {
		return ""
	}
	for _, rootfile := range c.Rootfiles {
		if rootfile.FullPath != "" {
			return cleanZipPath(rootfile.FullPath)
		}
	}
	return ""
}

func readOPF(f *zip.File) (opfPackage, error) {
	rc, err := f.Open()
	if err != nil {
		return opfPackage{}, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return opfPackage{}, err
	}
	var pkg opfPackage
	return pkg, xml.Unmarshal(data, &pkg)
}

func cleanZipPath(name string) string {
	name = strings.TrimSpace(filepath.ToSlash(name))
	if name == "" {
		return ""
	}
	base, _, _ := strings.Cut(name, "#")
	if decoded, err := url.PathUnescape(base); err == nil {
		base = decoded
	}
	return strings.TrimPrefix(path.Clean(base), "./")
}

func isXHTMLManifestItem(item opfItem) bool {
	props := strings.Fields(strings.ToLower(item.Properties))
	for _, prop := range props {
		if prop == "nav" {
			return false
		}
	}
	mediaType := strings.ToLower(strings.TrimSpace(item.MediaType))
	return mediaType == "application/xhtml+xml" || mediaType == "text/html" || isTextDocument(item.Href)
}

func isTextDocument(name string) bool {
	ext := strings.ToLower(path.Ext(cleanZipPath(name)))
	return ext == ".xhtml" || ext == ".html" || ext == ".htm"
}

func isNonContentDocument(name string) bool {
	base := strings.ToLower(path.Base(cleanZipPath(name)))
	return base == "cover.xhtml" || base == "cover.html" || base == "nav.xhtml" || base == "toc.xhtml" || base == "book-toc.xhtml"
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
