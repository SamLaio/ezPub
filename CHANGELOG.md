# Change Log

## 0.3.1 - 2026-09-23

- TXT 匯入會先移除 XML 1.0 不允許的控制字元，避免舊式 Big5／CP950 來源在 EPUBCheck 產生致命 XML 錯誤；新增回歸測試。

## 0.3.0 - 2026-08-15

- 累積 TXT 製作 EPUB 流程修正，改善書庫整理時由 TXT 重建 EPUB3 的相容性。
- 保留並文件化 Big5/CP950 解碼參數、系列 metadata、章節處理、封面與字型相關行為。

## 2026-08-12

- TXT 解碼新增明確 `big5` / `cp950` / `big5hkscs` 選項，方便將早期
  Big5 繁體中文 TXT 製作成 EPUB；`auto` 仍維持既有 UTF / GB18030 判斷，
  避免影響舊簡體來源檔。

## 2026-08-06

- `build` 新增 `-series` 與 `-series-index` 參數，會輸出 EPUB 3 collection
  metadata，以及 Calibre 相容的 `calibre:series` /
  `calibre:series_index` metadata，方便系列書從 TXT 重建 EPUB 時保留系列資訊。
