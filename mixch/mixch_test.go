package mixch

import (
	"context"
	_ "embed"
	"net/http"
	"testing"
)

var (
	//go:embed testdata/test1.htm
	test1htm string
	testUrl  string = "https://mixch.tv/u/16903258/live"
	testId   string = "16903258"
)

type FakeNetConn struct {
	url     string
	content string
}

func (me *FakeNetConn) GetFile(ctx context.Context, url string) ([]byte, error) {
	text, err := me.GetWebPage(ctx, url)
	return []byte(text), err
}
func (me *FakeNetConn) GetWebPage(ctx context.Context, url string) (string, error) {
	me.url = url
	return me.content, nil
}
func (me *FakeNetConn) Post(ctx context.Context, url string, data map[string]string) (string, error) {
	return "", nil
}

func (m *FakeNetConn) GetHttpClient() *http.Client {
	return nil
}

func TestLoadUserPage(t *testing.T) {
	mixch := Mixch{}
	conn := &FakeNetConn{}
	conn.content = test1htm
	mixch.Id = testId
	err := mixch.LoadUserPage(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	if mixch.M3u8Url != "https://d2ibghk7591fzs.cloudfront.net/hls/torte_16903258.m3u8" {
		t.Fatal("m3u8 url not match")
	}
}

func TestParseLivePage(t *testing.T) {
	mixch := Mixch{}
	success := mixch.parseLivePage(test1htm)
	if !success {
		t.Fatal("ParseLivePage fail")
	}
	if mixch.M3u8Url != "https://d2ibghk7591fzs.cloudfront.net/hls/torte_16903258.m3u8" {
		t.Fatal("m3u8 url not match")
	}
	if mixch.Chat != "wss://chat.mixch.tv/torte/room/16903258" {
		t.Fatal("chat url not match")
	}
}
