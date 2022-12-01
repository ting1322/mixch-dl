module mixch-dl

go 1.18

replace mixch => ./mixch

replace m3u8 => ./m3u8

replace twitcasting => ./twitcasting

replace inter => ./inter

require (
	inter v0.0.0-00010101000000-000000000000
	m3u8 v0.0.0-00010101000000-000000000000
	mixch v0.0.0-00010101000000-000000000000
	twitcasting v0.0.0-00010101000000-000000000000
)

require (
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/ting1322/chat-player/pkg/cplayer v0.0.0-20221201150455-8fe9771fe856 // indirect
	golang.org/x/crypto v0.3.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
