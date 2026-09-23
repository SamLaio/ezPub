# ezPub 0.3.0

ezPub 是 EasyPub 工作流的 Go 重製版，目標是把已無源碼、無維護的舊工具改成可讀、可測、可持續維護的專案。

0.1 版先完成核心轉檔能力：把純文字小說轉成 EPUB，並盡量相容舊 EasyPub 的 `config.xml` 設定。

## 核心功能

- TXT 轉 EPUB
- Windows GUI：`ezpub gui`
- 讀取 EasyPub `config.xml`
- 支援 UTF-8、UTF-8 BOM、UTF-16、GBK、GB18030
- 依章節正則拆章
- 匯出、編輯並套用章節表 TSV
- 固定長度拆章
- 產生 EPUB 3 `nav.xhtml`
- 產生舊閱讀器相容的 `toc.ncx`
- 產生書內目錄頁 `book-toc.xhtml`
- 依 Calibre Edit 歸位邏輯整理 EPUB 內部檔案：`text/`、`styles/`、`images/`、`fonts/`
- 產生封面頁
- 外部封面圖會寫入 EPUB cover metadata、guide 與 NCX 封面項
- 輸出書籍 description、publisher、subject、date、identifier、rights metadata
- 支援自訂 CSS
- 預設 CSS 使用 EasyPub 風格 class，例如 `.a`、`.titletoc`、`.tocl2`、`.titlel2std`
- 支援直排 CSS
- 支援內嵌字型
- 可透過 `pyftsubset` 子集化內嵌字型
- 支援插圖資源
- 自動掃描插圖並修正 raw HTML `<img src>` 引用
- 支援 EasyPub raw HTML 標記，預設是 `##`
- EPUB 轉 TXT
- 顯示文字與預設中文規則集中在 `internal/lang/lang.go`

## 建置

```powershell
go build -ldflags="-H=windowsgui" -o .\release\ezpub-0.3.1-windows-amd64.exe .\cmd\ezpub
```

若要產生雙擊時不顯示命令列視窗的 GUI exe：

```powershell
go build -ldflags="-H=windowsgui" -o .\release\ezpub-0.3.1-windows-amd64.exe .\cmd\ezpub
```

查看版本：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe version
```

開啟 GUI：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe
```

也可以直接帶入 TXT：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe gui ".\test_file\巴黎茶花女遺事 - 小仲馬.txt"
```

GUI 依 EasyPub 的主要分頁工作流設計，包含章節、版式、字體、書籍信息、定制 CSS、插圖與高級選項；ezPub 目前只製作 EPUB，因此不提供 MOBI、AZW3、KindleGen、ASIN、壓縮方式或預設後綴等非 EPUB 欄位。

## TXT 轉 EPUB

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build .\examples\sample.txt -o .\examples\sample.epub
```

使用舊 EasyPub 設定檔：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -config C:\PortableApps\easypub\config.xml
```

若未指定 `-title` / `-author`，且檔名符合 `書名 - 作者.txt`，ezPub 會自動拆出書名與作者。

常用選項：

```powershell
-title "書名"
-author "作者"
-translator "譯者"
-cover cover.jpg
-css custom.css
-font font.ttf
-image illustration.jpg
-image-list images.tsv
-scan-images .\images
-chapter-regex "^\s*第[一二三四五六七八九十]+章.*"
-split-count 10
-split-length 12000
-publisher "出版社"
-description "簡介"
-subject "小說"
-series "系列名稱"
-series-index 1
-date "2026-06-20"
-isbn "978..."
-identifier "urn:uuid:..."
-rights "All rights reserved"
-subset-fonts
-encoding auto
-vertical
```

文字編碼可用 `-encoding` 指定；目前支援 `auto`、`utf-8`、`gbk`、`gb18030`、
`big5`、`cp950`、`big5hkscs`、`utf-16le`、`utf-16be`。若舊 TXT 是 Big5 /
CP950，建議明確使用 `-encoding big5` 或 `-encoding cp950`，避免被自動流程當成
GB18030 解碼。

若 EasyPub 設定 `RecentOptions.savecss=1`，ezPub 會把 `-css` 指定的內容保存到 `setting/custom.css`，之後未指定 `-css` 時自動讀回；不會寫入原 EasyPub 目錄。

## 章節手動編輯

先匯出章節表：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe chapters novel.txt -o novel.chapters.tsv
```

TSV 欄位為：

```text
start_line	level	title	skip_heading
```

改完後套用：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -chapters novel.chapters.tsv
```

`skip_heading` 設為 `1` 時，該行只當章節標題，不放進正文；序章或手動切出的純內容段落可設為 `0`。

固定長度拆章：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -split-length 12000
```

相容 EasyPub 的「按長度均分 x 章」：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -split-count 10
```

如果 `config.xml` 中 `RecentOptions.splitmode` 是 `2`，ezPub 會讀取 `RecentOptions.splitcount` 作為均分章數。

未指定 `-o` 時，ezPub 會依 EasyPub 設定決定輸出位置：`AdvancedOptions.outputtosrc` 為 `1` 時輸出到來源檔同目錄，否則優先使用 `RecentOptions.outputfolder`。

## 插圖與字型

建立可編輯的插圖清單：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe images .\illustrations -o images.tsv
```

清單中的圖片檔名不可重複，因為 EPUB 內會歸位到同一個 `images/` 資料夾。刪除 TSV 中某一行就等同於從本次 EPUB 插圖清單移除該圖，不會刪除來源檔。

產生並複製可貼入 TXT 的 raw HTML 引用：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe images .\illustrations -copy-ref 001.jpg
```

引用格式會和 EasyPub 一致：

```html
##<div class="centeredimage"><img src="images/001.jpg" alt="001.jpg" class="attpic" /></div>
```

用系統預設圖片查看程式開啟圖片：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe images .\illustrations -open 001.jpg
```

建書時套用插圖清單：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -image-list images.tsv
```

若文字中使用 raw HTML 標記：

```text
##<img src="illustrations/001.jpg" alt="插圖"/>
```

可以掃描插圖目錄，ezPub 會依檔名或相對路徑把引用修正成 EPUB 內部路徑並加入資源：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -scan-images .\illustrations
```

release 版會在 `tools/pyftsubset.exe` 隨附 fontTools 的 `pyftsubset`。找到這個工具時，可在輸出前子集化內嵌字型：

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe build novel.txt -o novel.epub -font .\font.ttf -subset-fonts
```

ezPub 會依序尋找 exe 同目錄的 `tools/pyftsubset.exe`、`pyftsubset.exe`，最後才找 PATH。CLI 仍可手動指定：

```powershell
-subset-tool C:\path\pyftsubset.exe
```

使用 EasyPub `config.xml` 時，若未指定 `-font`，只會在 `RecentOptions.fonttype=2`（原 EasyPub「內嵌字體」）時讀取 `RecentOptions.font_embedded`；內嵌字型會寫入 EPUB 並設為全書預設字體。若 `RecentOptions.font_subsetting` 為 `1`，會自動嘗試字型子集化。
GUI 一律顯示「內嵌字體」與字型選擇按鈕；只有在偵測到 `pyftsubset.exe` 時才顯示「僅嵌入子集」。
`RecentOptions.fonttype=3` 是「使用閱讀器預設字體」，不會嵌入 `font_embedded`。

使用者可外部編輯或由 GUI 保存的設定集中放在 `setting/`。檔案可在第一次使用後才建立，也可以是空檔：

- `setting/tagList.txt`：GUI「書籍信息 > 類別」下拉選單，一行一個類別。
- `setting/custom.css`：自動保存定制 CSS。
- `setting/editor.txt`：TXT 編輯器路徑。

## EPUB 轉 TXT

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe txt novel.epub -o novel.txt
```

EPUB 轉 TXT 會優先依 `META-INF/container.xml` 找到 OPF，並按 OPF `spine` 閱讀順序抽出 `.xhtml`、`.html`、`.htm` 正文檔；若 EPUB 缺少可解析的 spine，才退回掃描一般 HTML/XHTML 文字檔。

## 檢查 EasyPub 設定

```powershell
.\release\ezpub-0.3.1-windows-amd64.exe inspect-config C:\PortableApps\easypub\config.xml
```

會顯示目前讀到的章節正則、空行處理、raw HTML 標記與 EPUB 相關設定。

## 語言檔

0.1 版已把程式顯示文字、預設書名、預設語系、EPUB 目錄標題、前言章名、錯誤訊息、預設章節正則集中到：

```text
internal/lang/lang.go
```

之後若要改成繁中、簡中、英文，先改這份檔案即可。真正多語系切換可以在這個基礎上再擴充成 JSON、TOML 或 embed 檔案。

## 已相容的 EasyPub 設定

- `RecentOptions.full_reg`
- `RecentOptions.removeblankline`
- `RecentOptions.splitmode`
- `RecentOptions.splitcount`
- `RecentOptions.simple_reg_p1`
- `RecentOptions.simple_reg_p2`
- `RecentOptions.simple_reg_p3`
- `RecentOptions.simple_reg_ext`
- `RecentOptions.simple_reg_leadingspace`
- `RecentOptions.outputfolder`
- `RecentOptions.lineheight`
- `RecentOptions.fontsize`
- `RecentOptions.indent`
- `RecentOptions.textalign`
- `RecentOptions.top`
- `RecentOptions.bottom`
- `RecentOptions.left`
- `RecentOptions.right`
- `RecentOptions.pagetopunit`
- `RecentOptions.pagebottomunit`
- `RecentOptions.pageleftunit`
- `RecentOptions.pagerightunit`
- `RecentOptions.margintop`
- `RecentOptions.margintopunit`
- `RecentOptions.cssoverwrite`
- `RecentOptions.forcetextcover`
- `RecentOptions.addspace`
- `RecentOptions.addspacecount`
- `RecentOptions.fonttype`
- `RecentOptions.font_customized`
- `RecentOptions.machineid`
- `RecentOptions.font_embedded`
- `RecentOptions.font_subsetting`
- `eReadersConfig`
- `AdvancedOptions.description`
- `AdvancedOptions.publisher`
- `AdvancedOptions.date`
- `AdvancedOptions.identifier`
- `AdvancedOptions.rights`
- `AdvancedOptions.enable_htmlrawtag`
- `AdvancedOptions.htmlrawtag`
- `AdvancedOptions.tocspace`
- `AdvancedOptions.forceemptychapter`
- `AdvancedOptions.emptychapterstyle`
- `AdvancedOptions.flowsize`
- `AdvancedOptions.outputtosrc`

## 尚未完成

- EPUB 驗證器整合
- Kindle Previewer 整合

## 測試

```powershell
go test ./...
```

## 專案原則

ezPub 不反編譯舊 EasyPub，也不嘗試逐行移植舊 .NET 程式。這個專案重建的是使用者真正需要的工作流：讀文字、拆章、套版、產生標準 EPUB，並保留必要的 EasyPub 設定相容性。
