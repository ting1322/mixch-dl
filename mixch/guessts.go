package mixch

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
)

func guessTs(firstTs, baseurl string, downloadedIdx int) []string {
	urlList := make([]string, 0)
	re, _ := regexp.Compile(`(.+-)(\d+)\.ts$`)
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
	log.Printf("REMEDY: downloaded video number:%v, current video number:%v, download:%v-%v\n", downloadedIdx, curIdx, startIdx, curIdx-1)
	for i := startIdx; i < curIdx; i++ {
		url := fmt.Sprintf("%v/%v%v.ts", baseurl, m[1], i)
		urlList = append(urlList, url)
	}
	return urlList
}
