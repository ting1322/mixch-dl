//go:build !android

package inter

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all" // register cookie store finders!
)

func importCookie(client *http.Client, baseurl string) {
	// uses registered finders to find cookie store files in default locations
	// applies the passed filters "Valid", "DomainHasSuffix()" and "Name()" in order to the cookies
	burl, err := url.Parse(baseurl)
	if err != nil {
		LogMsg(false, fmt.Sprintf("setting cookie, parse base url with error: %v", err))
		return
	}
	cookies := kooky.ReadCookies(kooky.Valid, kooky.DomainHasSuffix(burl.Host))
	LogMsg(false, fmt.Sprintf("load cookies from browser, found: %v", len(cookies)))

	hcookies := make([]*http.Cookie, len(cookies))
	for idx, c := range cookies {
		hcookies[idx] = &c.Cookie
	}
	client.Jar.SetCookies(burl, hcookies)
}
