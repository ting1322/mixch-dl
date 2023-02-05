在 linux 上面讓 systemd 把 mixch-dl 當作 service 執行的方法。

~~windows 就不用看了~~

# file path

- `*.service`: `/home/USER/.config/systemd/user/*.service`
- `*.target`: `/home/USER/.config/systemd/user/*.target`

# usage

`*.service` 之中的路徑要先建立，比如我的家目錄會有 umi 資料夾。

檔案放好之後，執行 `systemctl --user daemon-reload`

## 單次執行

```sh
systemctl --user start mixch-dl-umi-mixch.service
systemctl --user start mixch-dl-umi-tc.service
```

## 每次開機執行

首先為了讓電腦沒有登入的時候也能執行程式，需要下面指令，否則只會在打密
碼進桌面之後執行。

```sh
loginctl enable-linger
```

開啟並執行 service

```sh
systemctl --user enable --now mixch-dl-umi-mixch.service
systemctl --user enable --now mixch-dl-umi-tc.service
```

或是使用 target 指定全部

```sh
systemctl --user enable --now mixchdl.target
```

## 停止

`start` 換成 `stop`，`enable` 換成 `disable`。

# description

`*.service` 定義 systemd service，指定程式執行的參數（網址）。使用
user service 而不是 system service，因為檔案放在家目錄比較方便。
