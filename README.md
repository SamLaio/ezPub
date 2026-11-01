# ezPub

把純文字小說整理成 EPUB 的工具，提供 Windows 圖形介面與命令列。可偵測或手動編輯章節、加入封面與插圖、設定書籍資料、自訂 CSS、直排與內嵌字型，也能將 EPUB 抽出為 TXT。

ezPub 是 EasyPub 工作流的 Go 重製版，可讀取舊 EasyPub `config.xml` 中已支援的設定。它目前只製作 EPUB，不提供 MOBI／AZW3 或 KindleGen。

目前版本：**0.3.6**。版本異動見 [CHANGELOG](CHANGELOG.md)。

## 下載與快速開始

從 [Releases](https://github.com/SamLaio/ezPub/releases) 下載 Windows amd64 ZIP，完整解壓，保留執行檔旁的 `tools/` 資料夾。雙擊 `ezpub-0.3.6-windows-amd64.exe` 開啟 GUI；一般使用不需安裝 Go。

1. 開啟 TXT，確認文字編碼與書名、作者。
2. 檢查章節清單，修正誤判或漏掉的章節。
3. 視需要加入封面、書籍資料、版式、插圖及字型。
4. 輸出 EPUB，再用閱讀器及外部 EPUBCheck 檢查。

GUI 的主要工作區包含章節、版式、字體、書籍資料、定制 CSS、插圖與高級選項。標籤可用逗號、頓號或分號分隔，分別寫入 OPF subject。

## 命令列快速建書

在解壓目錄執行：

```powershell
.\ezpub-0.3.6-windows-amd64.exe build .\novel.txt -o .\novel.epub -title "書名" -author "作者"
```

若檔名為 `書名 - 作者.txt`，未指定書名與作者時可由檔名拆出。舊 Big5／CP950 文字建議明確加 `-encoding big5` 或 `-encoding cp950`，避免自動辨識誤判。

## 依需求查閱

| 想做什麼 | 說明 |
| --- | --- |
| 設定書名、譯者、系列、直排與輸出 | [建書參數](docs/usage.md#建書與書籍資料) |
| 手動修章、固定長度或均分章節 | [章節處理](docs/usage.md#章節處理) |
| 插入圖片、調整 CSS、嵌入或子集化字型 | [插圖與字型](docs/usage.md#插圖與字型) |
| EPUB 抽成文字 | [EPUB 轉 TXT](docs/usage.md#epub-轉-txt) |
| 使用舊 EasyPub config.xml | [設定相容性](docs/configuration.md) |

## 輸出與限制

輸出包含 EPUB3 nav、相容用 NCX、書內目錄與封面。未指定識別碼時建立 UUID，OPF 與 NCX 共用。文字、樣式、圖片、字型分別歸位至 `text/`、`styles/`、`images/`、`fonts/`；OPF、NCX 留在 package root。

章節自動偵測需人工回讀，尤其原稿格式不一致時；可匯出 TSV 調整。程式不會自動簡轉正。嵌入字型前應確認使用權限；子集化需隨附或自行提供 pyftsubset。

尚未完成：內建 EPUB 驗證器、Kindle Previewer 整合與外部多語系檔案切換。產生檔案不代表已通過 EPUBCheck 或 Kindle 驗收，需另外執行驗證。

## 從原始碼執行與測試

使用 Go，版本需求見 [go.mod](go.mod)。在專案根目錄可直接執行：

```powershell
go run .\cmd\ezpub help
go run .\cmd\ezpub build .\examples\sample.txt -o .\examples\sample.epub
go test ./...
```

Windows GUI 建置：

```powershell
go build -ldflags="-H=windowsgui" -o .\release\ezpub-0.3.6-windows-amd64.exe .\cmd\ezpub
```

GUI 版不開命令列視窗；需要查看 CLI help 時也可用 `go run`。顯示文字與預設規則集中在 `internal/lang/lang.go`。本專案重建工作流，不反編譯舊 EasyPub，也不逐行移植舊 .NET 程式。

## 第三方來源與致謝

### 實際使用的 Go 套件

| 專案／作者 | 使用範圍 | 上游授權 |
| --- | --- | --- |
| [Walk／The Walk Authors](https://github.com/lxn/walk) | Windows GUI 控制項與宣告式介面 | BSD-3-Clause |
| [Win／The Win Authors](https://github.com/lxn/win) | Windows API 封裝 | BSD-3-Clause |
| [golang.org/x/text／The Go Authors](https://pkg.go.dev/golang.org/x/text) | 文字編碼與轉換 | BSD-3-Clause |
| [golang.org/x/sys／The Go Authors](https://pkg.go.dev/golang.org/x/sys) | Windows 系統呼叫相依 | BSD-3-Clause |
| [govaluate／George Lester](https://github.com/Knetic/govaluate) | Walk 使用的運算式相依 | MIT |

實際鎖定版本見 `go.mod`／`go.sum`；`indirect` 標記不代表未納入程式，也不代表可省略署名。

### 隨附工具與行為參考

- [fontTools／Just van Rossum、Behdad Esfahbod 與貢獻者](https://github.com/fonttools/fonttools)：Windows release 隨附 `tools/pyftsubset.exe`，用於內嵌字型子集化；fontTools 主要採 MIT，所含個別元件依其原授權。它是外部工具，不是 ezPub 自行實作的子集化引擎。
- EasyPub：TXT 建書流程、GUI 操作與 `config.xml` 相容性參考。依原程式可觀察行為與設定重建，沒有反編譯或逐行移植舊 .NET 程式；目前未有可核實的原作者／官方公開網址，保留工具名稱及相容範圍，不猜測署名。
- [Sigil](https://github.com/Sigil-Ebook/Sigil) 的 SubsetFonts plugin：子集化流程參考，非必須安裝的 plugin；未將其列為 ezPub 自有引擎。
- [calibre／Kovid Goyal 與貢獻者](https://github.com/kovidgoyal/calibre)：參考 Edit「整理到資料夾中」的 EPUB 資源歸位行為。

### 測試素材

`test_file/` 使用《巴黎茶花女遺事》（小仲馬原著、林紓／王壽昌譯述）的公共財文本與舊 EasyPub 對照 EPUB。文字來源為維基文庫；測試封面來源為台灣數位出版聯盟文章，頁面標註「國立師範大學圖書館藏書」。逐項來源如下：

| 檔案 | 來源 | 記錄範圍 |
| --- | --- | --- |
| `巴黎茶花女遺事 - 小仲馬.txt` | [維基文庫：《巴黎茶花女遺事》](https://zh.wikisource.org/zh-hant/巴黎茶花女遺事) | 頁面明列小仲馬、林紓與王壽昌；已補精確來源頁，未推定測試檔的歷史下載版本 |
| `cover.jpg` | [台灣數位出版聯盟：《茶花女》和《茶花女遺事》…](https://www.dpublishing.org.tw/2011/02/blog-post_04.html) | 已比對同一封面影像；文章標註「國立師範大學圖書館藏書」，未另推定攝影者或個別圖片授權 |
| `images.jpg` | [Alamy：Strawberries falling out of a red bucket on a black table，圖片編號 128210808](https://www.alamy.com/stock-photo-strawberries-falling-out-of-a-red-bucket-on-a-black-table-128210808.html?imageid=4410535B-F775-479C-8BB7-CC4369716AC6&pn=1&searchId=51ebe53bd100fd5732359eed8bdc05e0&searchtype=9) | 維護者提供來源網址；尚未記錄原攝影者及再散布授權 |
| `single-fresh-red-strawberry-on-table-green-background-food-fruit-sweet-macro-juicy-plant-image-photo.jpg` | [Facebook 群組貼文](https://www.facebook.com/groups/724616145130798/posts/2230003161258748/) | 維護者提供取得來源；貼文不等同原攝影者或授權聲明 |

上述兩個圖片網址本次未能讀取頁面內容，來源依維護者提供記錄；沒有將來源連結當成已核實的再散布授權，也不將文字的公共財狀態套用到全部圖片。

ezPub 本身採 [GPL v3](LICENSE)。本專案授權不取代套件、外部執行檔、測試素材或使用者嵌入字型的原有授權；上表是已確認的主要組件與參考來源，不宣稱已完整盤點 pyftsubset 打包內部的所有組件。
