package inter

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/zellyn/kooky"
	_ "github.com/zellyn/kooky/browser/all" // register cookie store finders!
)

type INet interface {
	GetWebPage(ctx context.Context, url string) (string, error)
	GetFile(ctx context.Context, url string) ([]byte, error)
	Post(ctx context.Context, url string, data map[string]string) (string, error)
}

type Net struct {
	client http.Client
}

func NewNetConn(baseurl string) *Net {
	net := &Net{}

	// uses registered finders to find cookie store files in default locations
	// applies the passed filters "Valid", "DomainHasSuffix()" and "Name()" in order to the cookies
	burl, err := url.Parse(baseurl)
	if err != nil {
		log.Fatal("setting cookie, parse base url with error:", err)
	}
	cookies := kooky.ReadCookies(kooky.Valid, kooky.DomainHasSuffix(burl.Host))
	log.Printf("try load cookies from browser, fouund: %v", len(cookies))

	hcookies := make([]*http.Cookie, len(cookies))
	for idx, c := range cookies {
		hcookies[idx] = &c.Cookie
	}
	net.client.Jar, _ = cookiejar.New(nil)
	net.client.Jar.SetCookies(burl, hcookies)
	return net
}

func (m Net) Post(ctx context.Context, urltest string, data map[string]string) (string, error) {
	log.Println("POST:", urltest)
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, v := range data {
		fw, _ := w.CreateFormField(k)
		fw.Write([]byte(v))
	}
	w.Close()
	req, _ := http.NewRequest(http.MethodPost, urltest, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := m.DoReq(ctx, req)
	return string(resp), err
}

func (m Net) GetWebPage(ctx context.Context, url string) (string, error) {
	body, err := m.GetFile(ctx, url)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (m Net) GetFile(ctx context.Context, url string) ([]byte, error) {
	log.Println("GET:", url)
	req, _ := http.NewRequest("GET", url, nil)
	return m.DoReq(ctx, req)
}

func (m Net) DoReq(ctx context.Context, req *http.Request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:106.0) Gecko/20100101 Firefox/106.0")
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("http response: %v\n", resp.Status)
		return nil, errors.New("http response not OK")
	}
	//log.Println(resp)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, nil
}
