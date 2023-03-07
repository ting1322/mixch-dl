package mixch

import (
	"fmt"
	"inter"
	"regexp"
	"strconv"
)

func guessTs(firstTs, baseurl string, downloadedIdx int) []string {
	urlList := make([]string, 0)
	re, _ := regexp.Compile(`(.+-)(\d+)\.ts(\?.\w+=\w+)?`)
	m := re.FindStringSubmatch(firstTs)
	if m == nil || len(m) < 2 {
		return urlList
	}
	curIdx, err := strconv.Atoi(m[2])
	if err != nil {
		return urlList
	}
	//d.tryDownloadLostFrag(ctx, tsw, baseurl, m[1], curIdx)
	startIdx := curIdx - 6
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx < downloadedIdx+1 {
		startIdx = downloadedIdx + 1
	}
	inter.LogMsg(false, fmt.Sprintf("REMEDY: downloaded video number:%v, current video number:%v, download:%v-%v\n", downloadedIdx, curIdx, startIdx, curIdx-1))
	suffix := ""
	if len(m) >= 3 {
		suffix = m[3]
	}
	for i := startIdx; i < curIdx; i++ {
		url := fmt.Sprintf("%v/%v%v.ts%v", baseurl, m[1], i, suffix)
		urlList = append(urlList, url)
	}
	return urlList
}
