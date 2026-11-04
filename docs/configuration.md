# EasyPub 設定相容性

[回到使用說明](../README.md) · [命令列操作](usage.md)

## 載入與檢查

```powershell
.\ezpub-0.3.5-windows-amd64.exe inspect-config C:\PortableApps\easypub\config.xml
.\ezpub-0.3.5-windows-amd64.exe build novel.txt -o novel.epub -config C:\PortableApps\easypub\config.xml
```

檢查指令顯示讀入的章節正則、空行、raw HTML 與 EPUB 相關設定。支援的 CLI 選項可覆寫對應建書設定；明確指定 `-o` 可避免輸出位置受舊設定影響。

## 常用設定的行為

GUI 尚未儲存字體選擇時，預選「使用閱讀器默認字體」，不採用自動載入的舊 EasyPub `fonttype`；已儲存的 ezPub 字體選擇會保留在 `setting/gui.json`，下次啟動恢復。字型路徑仍可由舊設定補入，供使用者手動選用。命令列明確使用 `-config` 時，仍依下表套用舊設定。

| 設定 | ezPub 行為 |
| --- | --- |
| `RecentOptions.splitmode=2` | 使用 `splitcount` 均分章節 |
| `AdvancedOptions.outputtosrc=1` | 未指定輸出時，放在來源檔旁 |
| `RecentOptions.outputfolder` | 非來源旁輸出時的優先目的資料夾 |
| `RecentOptions.savecss=1` | 保存指定 CSS 至 `setting/custom.css`，之後可讀回 |
| `RecentOptions.fonttype=2` | 未指定 `-font` 時讀取 `font_embedded`，嵌入字型 |
| `RecentOptions.fonttype=3` | 使用閱讀器字型，不嵌入 `font_embedded` |
| `RecentOptions.font_subsetting=1` | 嘗試子集化內嵌字型，需 pyftsubset |

GUI 一律顯示內嵌字型選擇，偵測到 pyftsubset 才顯示「僅嵌入子集」。工具依序尋找執行檔旁 `tools/pyftsubset.exe`、旁邊的 `pyftsubset.exe`，最後找 PATH；CLI 可用 `-subset-tool` 指定。

內嵌字型可多選並以分號分隔儲存；第一個連結內建CSS的內文，第二個連結標題。欄位順序會保留，下次啟動可繼續使用。只有一個字型時，標題沿用內文。

## 本工具保存的設定

可編輯設定集中在 `setting/`，可首次使用後才建立，也可以是空檔。不會寫入原 EasyPub 安裝目錄。

| 檔案 | 用途 |
| --- | --- |
| `tagList.txt` | GUI 類別／標籤清單，一行一個 |
| `custom.css` | 自動保存的定制 CSS |
| `editor.txt` | TXT 編輯器路徑 |

## 支援的 EasyPub 欄位

以下依設定名稱列出相容範圍；未列出的舊功能不能據此假定已完整重現。ezPub 只製作 EPUB，不提供 MOBI、AZW3、KindleGen、ASIN 或其他非 EPUB 欄位。


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
