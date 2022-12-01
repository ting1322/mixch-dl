package twitcasting

import (
	_ "embed"
	"testing"
)

//go:embed testdata/stream1.json
var jsonText string

func TestParseStreamInfo(t *testing.T) {
	live := &Live{}
	err := parseStreamInfo(live, jsonText)
	if err != nil {
		t.Fatal(err)
	}
	if live.MovieId != "752620563" {
		t.Fatalf(`live.MovieId != "752620563", id=%v`, live.MovieId)
	}
	if !live.IsLive {
		t.Fatal("is live")
	}
	if live.VideoUrl != "wss://202-218-171-202.twitcasting.tv/tc.edge/v1/streams/752620563.138.96/fmp4" {
		t.Fatalf("video url error, %v", live.VideoUrl)
	}
}

func TestParseChatInfo(t *testing.T) {
	text := `{"url":"wss:\/\/s202218175202.twitcasting.tv\/event.pubsub\/v1\/streams\/752691856\/events?token=NzUyNjkxODU2%3A%3A%3A1669825734%3A2a714b775193997f&n=46684e3bf41b67ca"}`

	live := &Live{}
	err := parseChatInfo(live, text)
	if err != nil {
		t.Fatal(err)
	}

	if live.Chat != "wss://s202218175202.twitcasting.tv/event.pubsub/v1/streams/752691856/events?token=NzUyNjkxODU2%3A%3A%3A1669825734%3A2a714b775193997f&n=46684e3bf41b67ca" {
		t.Fatal("wss url")
	}
}
