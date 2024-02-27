package chzzk

import (
	_ "embed"
	"testing"
)

var (
	//go:embed testdata/livedetail.json
	livedetail_json string

	//go:embed testdata/livePlayback.json
	livePlayback_json string
)

func TestParseLivePage(t *testing.T) {
	chzzk := Chzzk{}
	success := chzzk.parseLiveDetail(livedetail_json)
	if !success {
		t.Fatal("parseLiveDetail fail")
	}
	if chzzk.M3u8Url != "https://livecloud.akamaized.net/chzzk/lip2_kr/cflexnmss2u0002/wbqwer7oxuimnoc44mwl5wivh3n3nw5fg8/hls_playlist.m3u8?hdnts=st=1709036626~exp=1709069036~acl=*/wbqwer7oxuimnoc44mwl5wivh3n3nw5fg8/*~hmac=a73b8e4d74b5f3a52473bb64502856209d0cb30a3f3d361d579acf5349659380" {
		t.Fatal("video m3u8 url not match")
	}
	if chzzk.M3u8UrlA != "https://livecloud.akamaized.net/chzzk/lip2_kr/cflexnmss2u0002/wbqwer7oxuimnoc44mwl5wivh3n3nw5fg8/afragalow.stream_hls_playlist.m3u8?hdnts=st=1709036626~exp=1709069036~acl=*/wbqwer7oxuimnoc44mwl5wivh3n3nw5fg8/*~hmac=a73b8e4d74b5f3a52473bb64502856209d0cb30a3f3d361d579acf5349659380" {
		t.Fatal("audio m3u8 url not match")
	}
	if chzzk.status != "OPEN" {
		t.Fatal("status not match")
	}
}