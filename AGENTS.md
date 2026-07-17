# ezPub Agent Rules

本檔是 ezPub 的 agent / Codex 專案慣例檔。push 或 release 前，先依照這裡的規則檢查。

## 版本

- 目前版本：`0.1.1`
- 版本號來源：`internal/lang/lang.go`
- README 標題與內容要和目前版本一致。

## 核心功能

0.1 版核心功能如下：

- TXT 轉 EPUB
- Windows GUI：`ezpub gui`
- 讀取 EasyPub `config.xml`
- UTF-8、UTF-8 BOM、UTF-16、GBK、GB18030 文字解碼
- 依章節正則拆章
- 匯出、編輯並套用章節表 TSV
- 固定長度拆章
- EPUB 3 `nav.xhtml`
- EPUB 2 相容 `toc.ncx`
- 書內目錄頁 `book-toc.xhtml`
- 封面頁
- 書籍 description、publisher、subject、date、identifier、rights metadata
- 自訂 CSS
- 預設 CSS 使用 EasyPub 風格 class，例如 `.a`、`.titletoc`、`.tocl2`、`.titlel2std`
- 直排 CSS
- 內嵌字型
- 透過 release 隨附或 PATH 中的 `pyftsubset.exe` 子集化內嵌字型
- 插圖資源
- 自動掃描插圖並修正 raw HTML `<img src>` 引用
- EasyPub raw HTML 標記，預設 `##`
- EPUB 轉 TXT
- 顯示文字、預設中文規則與錯誤訊息集中於 `internal/lang/lang.go`

## 原程式對照

- 原 EasyPub 程式位置：`C:\PortableApps\easypub`
- 字型子集化參考：Sigil plugin `C:\Users\f8512\AppData\Local\sigil-ebook\sigil\plugins\SubsetFonts`
- 補齊或修改舊 EasyPub 已有功能時，要先對照原程式可觀察行為與設定檔，再實作。
- 優先參考：
  - `C:\PortableApps\easypub\config.xml`
  - `C:\PortableApps\easypub\CHANGELOG.txt`
  - `C:\PortableApps\easypub\ereaders.xml`
  - `C:\PortableApps\easypub\css\_easypub_autosave_config.xml.css`
  - `C:\Users\f8512\AppData\Local\sigil-ebook\sigil\plugins\SubsetFonts\plugin.py`
  - 原程式實際輸出結果
- 若 Go 版因 CLI 形式、EPUB 標準或外部工具限制而無法完全照舊，需在 README 或註解中說明差異。

## 測試素材

- 標準測試書使用維基文庫公共財文本《巴黎茶花女遺事 - 小仲馬》，無版權問題，可作為轉檔與相容性測試素材。
- 測試素材固定放在 `test_file/`：
  - `test_file/巴黎茶花女遺事 - 小仲馬.txt`
  - `test_file/巴黎茶花女遺事 - 小仲馬.epub`
  - `test_file/巴黎茶花女遺事 - 小仲馬2.epub`
  - `test_file/巴黎茶花女遺事 - 小仲馬3.epub`
  - `test_file/cover.jpg`
  - `test_file/images.jpg`
  - `test_file/single-fresh-red-strawberry-on-table-green-background-food-fruit-sweet-macro-juicy-plant-image-photo.jpg`
- `test_file/巴黎茶花女遺事 - 小仲馬.epub` 是使用原 EasyPub 製作的對照 EPUB。
- `test_file/巴黎茶花女遺事 - 小仲馬2.epub` 是無外部封面圖、使用文字封面的原 EasyPub 對照 EPUB。
- `test_file/巴黎茶花女遺事 - 小仲馬3.epub` 是無外部封面圖、且未勾選文字封面的原 EasyPub 對照 EPUB。
- `test_file/cover.jpg` 用於外部封面圖測試；`test_file/images.jpg` 與另一張長檔名草莓圖用於插圖功能測試。
- 後續測試、原程式行為對照、EPUB 抽 TXT 比對、Go 版輸出回歸測試，優先使用這本書。
- `test_file/` 中的公共財 fixture 是測試資料，不視為一般轉檔輸出；可例外納入版本庫。

## EPUB 內部檔案歸位

- EPUB 內部資源整理參考 Calibre Edit 的「工具 > 整理到資料夾中」邏輯。
- 預設歸位資料夾使用 Calibre Edit 對話框中的命名：
  - 文字 HTML / XHTML 檔案：`text/`
  - 樣式表 CSS 檔案：`styles/`
  - 圖片與封面圖：`images/`
  - 字型：`fonts/`
  - 音訊：`audio/`
  - 視訊：`video/`
  - OPF 檔案與 NCX 目錄檔：留空，放在 package root。
- 產生 EPUB 時應依資源類型歸位到上述穩定資料夾，方便 Calibre Edit、閱讀器與後續人工修改。
- 若為了 EPUB 3 `nav.xhtml` 或相容舊 EasyPub 需要和 Calibre Edit 歸位結果不同，需在 README 或 TODO 記錄差異與原因。

## TODO

- 本機 TODO 檔案使用 `TODO.md`。
- `TODO.md` 不進版本庫，已列入 `.gitignore`。
- README 要保留公開可見的「尚未完成」摘要。
- `TODO.md` 可保留更細的開發筆記、排序、臨時想法。
- 任何可能跨對話延續的事項，都要直接記進 `TODO.md`，不要只留在對話中。
- 原程式對照、相容性差異、待確認的 EasyPub 行為、已知偏差與後續修正項目，都要記進 `TODO.md`。
- 若使用者要求「之後記住」「換對話也要保留」「記到 todo」或同義事項，agent 應直接更新 `TODO.md`。

目前未完成項目：

- EPUB 驗證器整合
- Kindle Previewer 整合
- 多語系檔案格式擴充，從 Go 常數改成 JSON、TOML 或 embed 檔案

## Push 前檢查

每次 push 前都要做：

1. 執行 `go test ./...`。
2. 檢查目前程式功能是否和 `README.md` 一致。
3. 如果程式功能和 `README.md` 不一致，先修改 `README.md`。
4. 產生一份 release 說明草稿，檔名以版本號分界，例如 `release-notes/0.1.1.md`。
5. release 說明草稿不進版本庫，已列入 `.gitignore`。
6. 不提交 `.exe`，`.exe` 只在 release 時產生。

## Release 前檢查

每次 release 前都要做：

1. 確認版本號已更新。
2. 執行 `go test ./...`。
3. 檢查是否已有該版本 release 說明檔，例如 `release-notes/0.1.1.md`。
4. 如果沒有 release 說明檔，先產生一份。
5. 確認 `release/` 資料夾下已有該版本的 exe，例如 `release/ezpub-0.1.1-windows-amd64.exe`。
6. exe 一律輸出到 `release/`。
7. exe 檔名一律包含版本號與平台，例如 `ezpub-0.1.1-windows-amd64.exe`。
8. 若要支援 GUI 的「僅嵌入子集」，確認 `release/tools/pyftsubset.exe` 存在。
9. 如果 release 資料夾下沒有該版本 exe，先建置：

```powershell
go build -ldflags="-H=windowsgui" -o .\release\ezpub-0.1.1-windows-amd64.exe .\cmd\ezpub
```

10. 發布前，將該版本 exe 與整個 `release/tools/` 資料夾壓縮成 zip，例如 `release/ezpub-0.1.1-windows-amd64.zip`：

```powershell
Compress-Archive -Force -Path .\release\ezpub-0.1.1-windows-amd64.exe, .\release\tools -DestinationPath .\release\ezpub-0.1.1-windows-amd64.zip
```

11. 發布時，只上傳該版本 zip 作為 release 附件；不要再把 exe 與 `pyftsubset.exe` 分開上傳。
12. 發布內容使用該版本 release 說明檔。

## Build 輸出慣例

- 一般程式修改後只需執行測試，不主動重新產生 exe。
- 只有使用者明確要求「產 exe」「產出 exe」或同義指令時，才重新建置 exe。
- 重新建置 exe 前後要進行必要測試；至少執行 `go test ./...`，並確認輸出的 Windows GUI exe 不會開啟命令列視窗。
- 不使用 `bin/ezpub.exe` 作為正式產物。
- 所有 exe 都必須輸出到 `release/`，包含正式版、測試版、GUI 版、臨時驗證 build。
- 不得在專案根目錄、`cmd/`、`bin/` 或其它資料夾留下任何產出的 `.exe`。
- 所有 exe 檔名都必須包含版本號。
- `pyftsubset.exe` 一律輸出到 `release/tools/pyftsubset.exe`。
- Release 附件一律打包成 `release/ezpub-<版本號>-windows-amd64.zip`，zip 內包含該版本 exe 與 `tools/` 資料夾。
- GUI 一律顯示「內嵌字體」與字型選擇按鈕；只有在偵測到 `pyftsubset.exe` 時才顯示「僅嵌入子集」。
- 使用者可外部編輯或由 GUI 保存的設定集中放在 `setting/`；檔案可在第一次使用後才建立，也可以是空檔。
- 目前 `setting/` 檔案包含：`tagList.txt`（類別下拉，一行一個）、`custom.css`（自動保存定制 CSS）、`editor.txt`（TXT 編輯器路徑）。
- Windows amd64 檔名格式：

```text
release/ezpub-<版本號>-windows-amd64.exe
release/ezpub-<版本號>-windows-amd64.zip
```

範例：

```powershell
go build -ldflags="-H=windowsgui" -o .\release\ezpub-0.1.1-windows-amd64.exe .\cmd\ezpub
Compress-Archive -Force -Path .\release\ezpub-0.1.1-windows-amd64.exe, .\release\tools -DestinationPath .\release\ezpub-0.1.1-windows-amd64.zip
```

## Git Ignore 規則

以下內容不得提交：

- `*.exe`
- `bin/`
- `release/`
- `release-notes/`
- `setting/`
- `TODO.md`
- 轉檔輸出：`*.epub`
- 測試抽出的文字：`*.out.txt`
