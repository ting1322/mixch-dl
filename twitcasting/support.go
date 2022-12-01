package twitcasting


import "strings"

func Support(url string) bool {
	return strings.HasPrefix(url, "https://twitcasting.tv/")
}