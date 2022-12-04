目的：下載 mixch 的影片，同時紀錄聊天室對話紀錄，並產生能夠播放聊天室文字的網頁檔案。

下載連結在右邊: https://github.com/ting1322/mixch-dl/releases/latest

# 使用說明

下載執行檔之後，用 cmd 執行，第一個參數是網址，第二個參數是預計執行的時間。
第二個參數可省略，會立刻開始執行。

ffmpeg 必須事先裝好，合併檔案需要 ffmpeg。

```
# 立刻開始下載影片
mixch-dl.exe https://mixch.tv/u/1234567

# 網址後面的 /live 可有可無
mixch-dl.exe https://mixch.tv/u/1234567/live

# 定時執行，如果預計晚上 19:00 開台，可以讓程式在 18:58 開始嘗試連網路
mixch-dl.exe https://mixch.tv/u/1234567/live 18:58

# twitcasting 的網址也有機會抓到，但影片經常會缺失片段
mixch-dl https://twitcasting.tv/c:annuuuu_cas
```

# 編譯

github 有上傳執行檔給 windows 64-bit, linux 64-bit。
我自己電腦主要是 linux 64-bit。
如果是更奇怪的系統 (Mac arm)，可能需要自己編譯。

1. 安裝 golang 的編譯器，讓 go 可以執行
2. 下載本專案原始碼
3. 下指令 go build

# 其他補充事項

如果不需要聊天室，請優先考慮使用 yt-dlp，那邊比較穩。

下載中隨時可以按 Ctrl-C 中斷，收到訊號我會試著結束下載，
並把已經下載的檔案用 ffmpeg 轉為 mp4。
如果按第二次 Ctrl-C，會強制結束程式。

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

如果執行的當下，指定的網址並沒有直播，會每隔 30 秒連一次網路，直到直播開始。
如果知道大概的開台時間，推薦使用第二個參數設定時間，節省網路資源。

cookie 的部份，會試著從瀏覽器抓。如果 firefox + chrome 同時都有登入的狀況，
似乎會抓到多個相同的 cookie？這我不太確定，如果怪怪的，建議只保留一個瀏覽器
是登入狀態，其他瀏覽器的 cookie 都清掉。

# dependency

除了 golang 內建 library 之外，還有參考引用別人的專案

1. nhooyr.io/websocket 連接 mixch 的 聊天室，以及 twitcasting 的 影片＋聊天室
2. github.com/zellyn/kooky 用來讀取瀏覽器的 cookie
