# ezPub 命令列使用手冊

[回到使用說明](../README.md)

以下範例在 PowerShell 執行，先將變數設為實際解壓位置：

```powershell
$ezpub = "C:\path\to\ezpub-0.3.8-windows-amd64.exe"
& $ezpub version
& $ezpub gui .\novel.txt
```

## 建書與書籍資料


```powershell
& $ezpub build .\examples\sample.txt -o .\examples\sample.epub
```

使用舊 EasyPub 設定檔：

```powershell
& $ezpub build novel.txt -o novel.epub -config C:\PortableApps\easypub\config.xml
```

若未指定 `-title` / `-author`，且檔名符合 `書名 - 作者.txt`，ezPub 會自動拆出書名與作者。

常用選項（附加於 `build input.txt -o output.epub` 後，範例路徑需換成實際檔案）：

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

使用 `-css style.css -css-mode 2` 完整替換預設 CSS 時，自訂的內文與標題字型設定會保留，不以自動選用的字型覆蓋。搭配 `-font` 時仍會嵌入字型資源及附加字型宣告；CSS 中的資源路徑須對應包內檔名。

`-css-mode 1` 將自訂CSS附加於內建CSS之後；`2` 以非空白自訂CSS完整替換，空白內容仍沿用內建CSS。模式 `0` 原則上忽略自訂CSS，但明確指定 `-css` 檔案時，建書入口會將模式0改為附加模式；要完整替換請明確傳入2，不能只選檔案就假定已替換。

內建 CSS 與嵌入字型依選取順序連結：第一個字型用於內文，第二個用於章節標題、目錄標題及文字封面標題；只有一個字型時，標題沿用內文字型。命令列可重複指定 `-font body.otf -font heading.ttf`；GUI 的字型選檔支援多選，欄位以分號分隔，可調整先後順序。附加自訂 CSS 時，自訂規則優先；完整替換時完全採用自訂規則。子集化保留原檔名，所以 `../fonts/body.otf` 等自訂字型路徑仍然有效。

### 自訂CSS如何引用嵌入字型

工具為選取字型依序附加 `ezpub-font-1`、`ezpub-font-2` 等 `@font-face` 別名。完整替換模式保留你自己的選擇器，不會自動替你補上內文與標題的字型設定。可在自訂CSS中使用這些別名：

```css
body { font-family: "ezpub-font-1", serif; }
h1, h2, h3, h4, h5, h6, .booktitle, .bookauthor, .titletoc, .title {
  font-family: "ezpub-font-2", "ezpub-font-1", serif;
}
```

若改用字型自己的家族名稱，就需自行撰寫相符的 `@font-face`，例如在包內 `styles/style.css` 以 `../fonts/body.otf` 引用檔案。嵌入成功不等於CSS已套用，應核對字型別名／家族、相對路徑與正文實際使用的選擇器。ezPub沒有固定採用個人字型資料夾或CSS的全域預設。

## 章節處理

先匯出章節表：

```powershell
& $ezpub chapters novel.txt -o novel.chapters.tsv
```

TSV 欄位為：

```text
start_line	level	title	skip_heading
```

改完後套用：

```powershell
& $ezpub build novel.txt -o novel.epub -chapters novel.chapters.tsv
```

`skip_heading` 設為 `1` 時，該行只當章節標題，不放進正文；序章或手動切出的純內容段落可設為 `0`。

固定長度拆章：

```powershell
& $ezpub build novel.txt -o novel.epub -split-length 12000
```

相容 EasyPub 的「按長度均分 x 章」：

```powershell
& $ezpub build novel.txt -o novel.epub -split-count 10
```

如果 `config.xml` 中 `RecentOptions.splitmode` 是 `2`，ezPub 會讀取 `RecentOptions.splitcount` 作為均分章數。

未指定 `-o` 時，ezPub 會依 EasyPub 設定決定輸出位置：`AdvancedOptions.outputtosrc` 為 `1` 時輸出到來源檔同目錄，否則優先使用 `RecentOptions.outputfolder`。

## 插圖與字型

建立可編輯的插圖清單：

```powershell
& $ezpub images .\illustrations -o images.tsv
```

清單中的圖片檔名不可重複，因為 EPUB 內會歸位到同一個 `images/` 資料夾。刪除 TSV 中某一行就等同於從本次 EPUB 插圖清單移除該圖，不會刪除來源檔。

產生並複製可貼入 TXT 的 raw HTML 引用：

```powershell
& $ezpub images .\illustrations -copy-ref 001.jpg
```

引用格式會和 EasyPub 一致：

```html
##<div class="centeredimage"><img src="images/001.jpg" alt="001.jpg" class="attpic" /></div>
```

用系統預設圖片查看程式開啟圖片：

```powershell
& $ezpub images .\illustrations -open 001.jpg
```

建書時套用插圖清單：

```powershell
& $ezpub build novel.txt -o novel.epub -image-list images.tsv
```

若文字中使用 raw HTML 標記：

```text
##<img src="illustrations/001.jpg" alt="插圖"/>
```

可以掃描插圖目錄，ezPub 會依檔名或相對路徑把引用修正成 EPUB 內部路徑並加入資源：

```powershell
& $ezpub build novel.txt -o novel.epub -scan-images .\illustrations
```

release 版會在 `tools/pyftsubset.exe` 隨附 fontTools 的 `pyftsubset`。找到這個工具時，可在輸出前子集化內嵌字型：

```powershell
& $ezpub build novel.txt -o novel.epub -font .\font.ttf -subset-fonts
```

CLI明確指定 `-subset-tool` 時只檢查該路徑；不存在則子集化失敗，不再自動尋找。未指定時，依序找exe旁的 `tools/pyftsubset.exe`、`pyftsubset.exe`，再找目前工作目錄下的 `release/tools/pyftsubset.exe`、`release/pyftsubset.exe`、`tools/pyftsubset.exe`、`pyftsubset.exe`，最後才找PATH中的 `pyftsubset.exe`／`pyftsubset`。可手動指定：

```powershell
-subset-tool C:\path\pyftsubset.exe
```

EasyPub 的字型設定、GUI 子集化選項及 `setting/` 檔案，集中於 [設定相容性](configuration.md)。

## EPUB 轉 TXT

```powershell
& $ezpub txt novel.epub -o novel.txt
```

EPUB 轉 TXT 會優先依 `META-INF/container.xml` 找到 OPF，並按 OPF `spine` 閱讀順序抽出 `.xhtml`、`.html`、`.htm` 正文檔；若 EPUB 缺少可解析的 spine，才退回掃描一般 HTML/XHTML 文字檔。

## 驗證與失敗處理

建書後先確認輸出可開啟，再核對章節順序、目錄、書籍資料、圖片與字型。可用外部 EPUBCheck 驗證；工具尚未內建驗證器。無法偵測的章節應改用手動 TSV，編碼錯誤則先指定原稿編碼，不要在亂碼內容上繼續建書。
