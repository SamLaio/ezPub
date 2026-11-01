# ezPub

把純文字小說整理成 EPUB 的工具，提供 Windows 圖形介面與命令列。可偵測或手動編輯章節、加入封面與插圖、設定書籍資料、自訂 CSS、直排與內嵌字型，也能將 EPUB 抽出為 TXT。

ezPub 是 EasyPub 工作流的 Go 重製版，可讀取舊 EasyPub `config.xml` 中已支援的設定。它目前只製作 EPUB，不提供 MOBI／AZW3 或 KindleGen。

目前版本：**0.3.5**。版本異動見 [CHANGELOG](CHANGELOG.md)。

## 下載與快速開始

從 [Releases](https://github.com/SamLaio/ezPub/releases) 下載 Windows amd64 ZIP，完整解壓，保留執行檔旁的 `tools/` 資料夾。雙擊 `ezpub-0.3.5-windows-amd64.exe` 開啟 GUI；一般使用不需安裝 Go。

1. 開啟 TXT，確認文字編碼與書名、作者。
2. 檢查章節清單，修正誤判或漏掉的章節。
3. 視需要加入封面、書籍資料、版式、插圖及字型。
4. 輸出 EPUB，再用閱讀器及外部 EPUBCheck 檢查。

GUI 的主要工作區包含章節、版式、字體、書籍資料、定制 CSS、插圖與高級選項。標籤可用逗號、頓號或分號分隔，分別寫入 OPF subject。

## 命令列快速建書

在解壓目錄執行：

```powershell
.\ezpub-0.3.5-windows-amd64.exe build .\novel.txt -o .\novel.epub -title "書名" -author "作者"
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
go build -ldflags="-H=windowsgui" -o .\release\ezpub-0.3.5-windows-amd64.exe .\cmd\ezpub
```

GUI 版不開命令列視窗；需要查看 CLI help 時也可用 `go run`。顯示文字與預設規則集中在 `internal/lang/lang.go`。本專案重建工作流，不反編譯舊 EasyPub，也不逐行移植舊 .NET 程式。
