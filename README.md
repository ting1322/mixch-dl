目的：下載 mixch 的影片，同時紀錄聊天室對話紀錄，並產生能夠播放聊天室文字的網頁檔案。

下載連結在右邊: https://github.com/ting1322/mixch-dl/releases/latest

# 使用說明

下載執行檔之後，用 cmd 執行，第一個參數是網址，第二個參數是預計執行的時間。
第二個參數可省略，會立刻開始執行。

ffmpeg 必須事先裝好，合併檔案需要 ffmpeg。
ffmpeg 通常會附贈 ffprobe，這個也需要。用來查詢影片時間，以便同步聊天室時間軸。

```
# 立刻開始下載影片
mixch-dl.exe https://mixch.tv/u/1234567

# 網址後面的 /live 可有可無
mixch-dl.exe https://mixch.tv/u/1234567/live

# 定時執行，如果預計晚上 19:00 開台，可以讓程式在 18:58 開始嘗試連網路
mixch-dl.exe https://mixch.tv/u/1234567/live 18:58

# twitcasting 的網址也有機會抓到，但影片經常會缺失片段
mixch-dl https://twitcasting.tv/c:annuuuu_cas

# spoon 的網址也有機會抓到 (網址 jp 限定)
mixch-dl https://www.spooncast.net/jp/live/@lala_ukulele

# twitcasting 需要密碼可以用 -pass 指定密碼
mixch-dl -pass THE_PASSWORD https://twitcasting.tv/quon01tama

# 查看目前程式版本
mixch-dl -version
```

## systemd service

在 linux 可以使用 systemd 來啟動程式，這邊有我的 service 定義： [systemd-script](systemd-script)

# 編譯

github 有上傳執行檔給 windows 64-bit, linux 64-bit。
我自己電腦主要是 linux 64-bit。
如果是更奇怪的系統 (Mac arm)，可能需要自己編譯。

1. 安裝 golang 的編譯器，讓 go 可以執行
2. 下載本專案原始碼
3. 下指令 go build

# 其他補充事項

如果不需要聊天室，請優先考慮使用 yt-dlp，那邊比較穩。

## 控制

下載中隨時可以按 Ctrl-C 中斷，收到訊號我會試著結束下載，
並把已經下載的檔案用 ffmpeg 轉為 mp4。
如果按第二次 Ctrl-C，會強制結束程式。

## 輸出檔案

下載後產生的檔案應該是 mp4 + htm。htm 點開會有聊天室，跟隨影片播放捲動。

如果看到 xxx.ts.part 檔案，表示程式正在執行中，如果沒有在執行，表示剛剛當掉了。
如果程式結束看到 xxx.ts 檔案，表示合併 mp4 的過程失敗，或許是 ffmpeg 沒有裝好。
這兩個問題可以手動執行 ffmpeg 產生 mp4 檔案

```
ffmpeg -i xxx.ts -c copy -map 0 -dn -ignore_unknown -movflags +faststart xxx.mp4
```

如果產生的 mp4 總時間不合理，比如說預期 30 分鐘，但播放軟體顯示 2 小時。
可以試著用 ffmpeg 去除最前面的一點點，重新設定時間。(twitcasting 經常發生)

```
ffmpeg -i xxx.ts -c copy -bsf setts=ts=TS-STARTPTS -map 0 -dn -ignore_unknown -ss 1ms -movflags +faststart xxx.mp4
```

如果程式結束看到 xxx.live_chat.json，這是正常現象。這是聊天是紀錄檔，可以刪掉。
留著 json 檔案，可以給另一支程式使用：https://github.com/ting1322/chat-player

## 行為

### 網路

如果執行的當下，指定的網址並沒有直播，會每隔 15 秒連一次網路，直到直播開始。
如果知道大概的開台時間，推薦使用第二個參數設定時間，節省網路資源。

### cookie

cookie 的部份，如果需要可以使用 `-cookies-from-browser` 會試著從瀏覽器
抓。如果 firefox + chrome 同時都有登入的狀況，似乎會抓到多個相同的
cookie？這我不太確定，如果怪怪的，建議只保留一個瀏覽器是登入狀態，其他
瀏覽器的 cookie 都清掉。

### mixch

mixch 的影片網址都類似下面這樣的格式:
```
https://d2ibghk7591fzs.cloudfront.net/hls/torte_u_17347373_s_17204820-5173.ts
https://d2ibghk7591fzs.cloudfront.net/hls/torte_u_17347373_s_17204820-5174.ts
```
m3u8 裡面只會有最新的兩個片段，每個片段 2 秒。
我們可以試著把數字往回推，猜測 m3u8 沒有給的更早的網址。但是伺服器那邊只會允許
往回抓 2 ~ 5 個片段，更往前會回應 403 錯誤。

### twitcasting

這邊的影片並不是 m3u8，而是使用 WebSocket 連線，伺服器會不斷的送影片資料過來。
只要把全部的資料寫入一個檔案，就可以丟給 ffmpeg 轉成 mp4。
但這也意味著我們只能連線上去，被動的等伺服器送資料過來。而不能像 mixch 一樣猜測
網址試著往回頭抓前面幾秒的影片。

### spoon

問題1: 明明沒有畫面，在 2023-09 之後抓到的 mp4 都變成有畫面

可以加上參數
`--prefer-fmt=audio_only` 強制抓沒有影像的 m3u8。

問題2: 限定登入者的直播，一直出現錯誤 `WSS (chat): in connectTry1: failed to get reader:...`

可以加上參數
`-spoon-chat=false` 強制不抓聊天室。
因為這隻程式沒辦法登入，在登入者限定的直播沒辦法連線到聊天室。


# 建議

如果知道大概的開台時間，推薦使用第二個參數設定時間，節省網路資源。
例如知道下午六點開台，可以指定程式從 18:00 開始試著連接網路
```
mixch-dl.exe https://mixch.tv/u/1234567/live 18:58
```

預設狀況，成功的下載一個影片之後，程式將會結束。如果直播中間短暫的關台，
然後又再次開始直播，後半場可能就抓不到了。這種狀況建議使用參數 `-loop`
這會讓程式無限的執行，一個直播結束就繼續等待下一個直播。
但是不斷的等待，等於是一個人很固定的每隔 15 秒按一次 F5 重整網頁，
說不定會被 ban 掉，這要自行斟酌。


# dependency

需要 ffmpeg 以及 ffprobe。ffmpeg 用來合併片段為完整影片檔案，
ffprobe 用來查詢目前片段累計的時間。

除了 golang 內建 library 之外，還有參考引用別人的專案

1. nhooyr.io/websocket 連接 mixch 的 聊天室，以及 twitcasting 的 影片＋聊天室
2. github.com/browserutils/kooky 用來讀取瀏覽器的 cookie
3. github.com/mattn/go-isatty 用來判斷目前的輸入，是 log 檔案還是終端機
