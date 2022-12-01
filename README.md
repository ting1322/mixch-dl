目的：下載 mixch 的影片，同時紀錄聊天室對話紀錄，並產生能夠播放聊天室文字的網頁檔案。

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

