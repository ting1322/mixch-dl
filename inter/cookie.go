//go:build !android

package inter

import (
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/zellyn/kooky"
	_ "github.com/zellyn/kooky/browser/all" // register cookie store finders!
)

func importCookie(client *http.Client, baseurl string) {
	// uses registered finders to find cookie store files in default locations
	// applies the passed filters "Valid", "DomainHasSuffix()" and "Name()" in order to the cookies
	burl, err := url.Parse(baseurl)
	if err != nil {
		log.Fatal("setting cookie, parse base url with error:", err)
	}
	cookies := kooky.ReadCookies(kooky.Valid, kooky.DomainHasSuffix(burl.Host))
	log.Printf("load cookies from browser, found: %v", len(cookies))

	hcookies := make([]*http.Cookie, len(cookies))
	for idx, c := range cookies {
		hcookies[idx] = &c.Cookie
	}
	client.Jar, _ = cookiejar.New(nil)
	client.Jar.SetCookies(burl, hcookies)
}
