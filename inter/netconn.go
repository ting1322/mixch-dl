package inter

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type INet interface {
	GetWebPage(ctx context.Context, url string) (string, error)
	GetFile(ctx context.Context, url string) ([]byte, error)
	PostForm(ctx context.Context, url string, data map[string]string) (string, error)
	Post(ctx context.Context, url string, data map[string]string) (string, error)
	GetHttpClient() *http.Client
	GetCookie(name, domain, path string) (string, error)
}

var (
	LogNetwork     bool = true
	AutoLoadCookie bool = true
)

type Net struct {
	client http.Client
}

func logNetln(v ...any) {
	if LogNetwork {
		log.Println(v...)
	}
}

func (this *Net) GetHttpClient() *http.Client {
	return &this.client
}

func NewNetConn(baseurl string) *Net {
	net := &Net{}
	net.client.Jar, _ = cookiejar.New(nil)

	if AutoLoadCookie {
		importCookie(&net.client, baseurl)
	}
	return net
}

func (this Net) Post(ctx context.Context, urltest string, data map[string]string) (string, error) {
	logNetln("POST:", urltest)
	var b bytes.Buffer
	buffer := bufio.NewWriter(&b)
	first := true
	for k, v := range data {
		if !first {
			buffer.WriteString("&")
		}
		first = false
		buffer.WriteString(k)
		buffer.WriteString("=")
		buffer.WriteString(v)
	}
	buffer.Flush()
	req, _ := http.NewRequest(http.MethodPost, urltest, &b)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := this.DoReq(ctx, req)
	return string(resp), err
}

func (this Net) PostForm(ctx context.Context, urltest string, data map[string]string) (string, error) {
	logNetln("POST-FORM:", urltest)
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, v := range data {
		fw, _ := w.CreateFormField(k)
		fw.Write([]byte(v))
	}
	w.Close()
	req, _ := http.NewRequest(http.MethodPost, urltest, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := this.DoReq(ctx, req)
	return string(resp), err
}

func (this Net) GetWebPage(ctx context.Context, url string) (string, error) {
	body, err := this.GetFile(ctx, url)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (this Net) GetFile(ctx context.Context, url string) ([]byte, error) {
	logNetln("GET:", url)
	req, _ := http.NewRequest("GET", url, nil)
	return this.DoReq(ctx, req)
}

func (this *Net) GetCookie(name, domain, path string) (string, error) {
	urlText, err := url.Parse(domain + path)
	if err != nil {
		return "", fmt.Errorf("get cookie: %w", err)
	}
	for _, cookie := range this.client.Jar.Cookies(urlText) {
		if cookie.Name == name {
			return cookie.Value, nil
		}
	}
	return "", errors.New("not found cookie")
}

func (this Net) DoReq(ctx context.Context, req *http.Request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/110.0")
	resp, err := this.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("http response: %v\n", resp.Status)
		return nil, fmt.Errorf("%w, Status Code: %v", ErrHttpNotOk, resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, nil
}
