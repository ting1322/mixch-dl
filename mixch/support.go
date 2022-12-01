package mixch

import "strings"

func Support(url string) bool {
	return strings.HasPrefix(url, "https://mixch.tv/u/")
}