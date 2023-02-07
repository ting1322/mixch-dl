module mixch-dl

go 1.18

replace mixch => ./mixch

replace m3u8 => ./m3u8

replace twitcasting => ./twitcasting

replace spoon => ./spoon

replace inter => ./inter

require (
	inter v0.0.0-00010101000000-000000000000
	m3u8 v0.0.0-00010101000000-000000000000
	mixch v0.0.0-00010101000000-000000000000
	spoon v0.0.0-00010101000000-000000000000
	twitcasting v0.0.0-00010101000000-000000000000
)

require (
	github.com/Velocidex/json v0.0.0-20220224052537-92f3c0326e5a // indirect
	github.com/Velocidex/ordereddict v0.0.0-20221110130714-6a7cb85851cd // indirect
	github.com/Velocidex/yaml/v2 v2.2.8 // indirect
	github.com/bobesa/go-domain-util v0.0.0-20190911083921-4033b5f7dd89 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-sqlite/sqlite3 v0.0.0-20180313105335-53dd8e640ee7 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gonuts/binary v0.2.0 // indirect
	github.com/keybase/go-keychain v0.0.0-20221221221913-9be78f6c498b // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/ting1322/chat-player/pkg/cplayer v0.0.0-20230204164700-fb69399a3f2c // indirect
	github.com/zalando/go-keyring v0.2.2 // indirect
	github.com/zellyn/kooky v0.0.0-20221025221128-3e66d684c4db // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
	www.velocidex.com/golang/go-ese v0.1.0 // indirect
)
