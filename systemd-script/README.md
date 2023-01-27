在 linux 上面讓 systemd 把 mixch-dl 當作 service 執行的方法。

# file path

- `*.service`: `/home/USER/.config/systemd/user/*.service`
- `*.sh`: `/home/USER/umi/*.sh`

# usage

## 單次執行

```sh
systemctl --user start mixch-dl-umi-mixch.service
systemctl --user start mixch-dl-umi-tc.service
```

## 每次開機執行

```sh
systemctl --user enable --now mixch-dl-umi-mixch.service
systemctl --user enable --now mixch-dl-umi-tc.service
loginctl enable-linger
```

## 停止

`start` 換成 `stop`，`enable` 換成 `disable`。

# description

`*.service` 定義 systemd service，啟動裡面寫的 `*.sh`，再由 `*.sh` 啟
動 mixch-dl 帶入正確參數、網址。

每次執行會產生 .log 的紀錄擋，檔名帶著啟動時間。