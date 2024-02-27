package chzzk

import "strings"

func Support(url string) bool {
	return strings.HasPrefix(url, "https://chzzk.naver.com/live/")
}
