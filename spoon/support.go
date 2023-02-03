package spoon

import "strings"

func Support(url string) bool {
	return strings.HasPrefix(url, "https://www.spooncast.net/jp/live/@")
}
