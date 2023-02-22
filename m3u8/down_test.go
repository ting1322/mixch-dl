package m3u8

import (
	"context"
	_ "embed"
	"math"
	"net/http"
	"testing"
)

var (
	//go:embed testdata/test2.m3u8
	m3u8Text string
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

func TestDownloadM3U8(t *testing.T) {
	conn := &FakeNetConn{}
	conn.content = m3u8Text
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := &Downloader{}
	m3u8, err := d.downloadM3U8(ctx, "123", conn)
	if err != nil {
		t.Fatal(err)
	}
	if m3u8.sequence != 5854 {
		t.Fatal("sequence != 5854")
	}
	if m3u8.targetDuration != 2 {
		t.Fatal("targetDuration != 2")
	}
	if len(m3u8.tsList) != 2 {
		t.Fatal("ts file count in m3u8 is 2")
	}
	if math.Abs(m3u8.tsList[0].duration-2) > 0.001 {
		t.Fatalf("math.Abs(m3u8.tsList[0].duration-2) < 0.001, math.Abs(m3u8.tsList[0].duration-2)=%v", math.Abs(m3u8.tsList[0].duration-2))
	}
	if m3u8.tsList[0].name != "torte_u_16487670_s_16998560-5854.ts" {
		t.Fatal("m3u8.tsList[1].name != \"torte_u_16487670_s_16998560-5854.ts\"")
	}
	if math.Abs(m3u8.tsList[1].duration-2.499) > 0.001 {
		t.Fatalf("math.Abs(m3u8.tsList[1].duration - 2.499) < 0.001, math.Abs(m3u8.tsList[1].duration-2.499)=%v", math.Abs(m3u8.tsList[1].duration-2.499))
	}
	if m3u8.tsList[1].name != "torte_u_16487670_s_16998560-5855.ts" {
		t.Fatal("m3u8.tsList[1].name != \"torte_u_16487670_s_16998560-5855.ts\"")
	}
}
