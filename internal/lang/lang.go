package lang

const Version = "0.3.2"

const (
	AppName           = "ezpub"
	AppDescription    = "maintainable EasyPub-style ebook builder"
	DefaultBookTitle  = "Untitled"
	DefaultLanguage   = "zh-TW"
	PrefaceTitle      = "正文"
	TOCTitle          = "目錄"
	FixedChapterTitle = "第%03d段"
)

const DefaultChapterRegex = `^\s*[第卷][0123456789一二三四五六七八九十零〇百千两兩]*[章回部节節集卷篇話话].*|^\s*Chapter\s*[0123456789]+.*`

const Usage = `ezpub 0.3.2 - maintainable EasyPub-style ebook builder

Usage:
  ezpub build input.txt -o output.epub [options]
  ezpub gui [input.txt]
  ezpub chapters input.txt -o chapters.tsv [options]
  ezpub images image-or-dir... -o images.tsv
  ezpub txt input.epub -o output.txt
  ezpub inspect-config C:\PortableApps\easypub\config.xml
  ezpub version

Common options:
  -config path          read EasyPub config.xml
  -title text           book title
  -author text          book author
  -translator text      book translator
  -cover path           cover image
  -css path             custom CSS
  -font path            embed font; repeat for more fonts
  -image path           embed image asset; repeat for more images
  -image-list path      read image TSV from the images command
  -scan-images dir      scan image assets and rewrite raw HTML img src by filename
  -chapters path        read manually edited chapter TSV
  -split-length n       split chapters by roughly n characters
  -publisher text       book publisher
  -description text     book description
  -subject text         subject tag; repeat for more tags
  -series text          book series
  -series-index number  book series index
  -date text            publication date
  -identifier text      book identifier
  -rights text          rights statement
  -subset-fonts         subset embedded fonts with pyftsubset
  -chapter-regex expr   override chapter matching regex
  -encoding name        auto, utf-8, gbk, gb18030, big5, cp950, utf-16le, utf-16be
  -split-count n        split into n even chapters, matching EasyPub length split
  -vertical             append vertical writing CSS
`

const (
	ErrMissingCommand      = "missing command"
	ErrBuildInputCount     = "build needs exactly one input txt file"
	ErrChaptersInputCount  = "chapters needs exactly one input txt file"
	ErrTXTInputCount       = "txt needs exactly one input epub file"
	ErrImagesInputCount    = "images needs at least one image file or directory"
	ErrInspectConfigCount  = "inspect-config needs a config.xml path"
	ErrOutputMustBeEPUB    = "output path must end with .epub"
	ErrUnsupportedEncoding = "unsupported encoding %q"
	ErrInvalidUTF8         = "input is not valid utf-8"
	ErrNilBook             = "book is nil"
	ErrEmptyBook           = "book has no chapters"
	ErrEmptyChapterPlan    = "chapter plan is empty"
	ErrInvalidChapterPlan  = "invalid chapter plan; expected start_line<TAB>level<TAB>title<TAB>skip_heading"
	ErrNoXHTMLText         = "no XHTML text files found in epub"
	ErrFontSubsetterNeeded = "font subsetting needs tools/pyftsubset.exe, pyftsubset.exe, PATH pyftsubset, or -subset-tool"
	ErrFontSubsetFailed    = "font subsetting failed: %w\n%s"
	MsgBuilt               = "built %s (%d chapters, text encoding %s)\n"
	MsgChaptersWritten     = "wrote %s (%d chapter marks, text encoding %s)\n"
)
