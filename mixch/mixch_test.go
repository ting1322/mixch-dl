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

func (this *FakeNetConn) GetFile(ctx context.Context, url string) ([]byte, error) {
	text, err := this.GetWebPage(ctx, url)
	return []byte(text), err
}
func (this *FakeNetConn) GetWebPage(ctx context.Context, url string) (string, error) {
	this.url = url
	return this.content, nil
}
func (this *FakeNetConn) Post(ctx context.Context, url string, data map[string]string) (string, error) {
	return "", nil
}
func (this *FakeNetConn) PostForm(ctx context.Context, url string, data map[string]string) (string, error) {
	return "", nil
}

func (this *FakeNetConn) GetHttpClient() *http.Client {
	return nil
}

func (this *FakeNetConn) GetCookie(name, domain, path string) (string, error) {
	return "", nil
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
