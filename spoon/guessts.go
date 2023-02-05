package spoon

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
)

var (
	emedyEnable    bool = true
	emedyMaxNumber int  = 20
)

func guessTs(firstTs, baseurl string, downloadedIdx int) []string {
	urlList := make([]string, 0)
	if emedyEnable {
		re, _ := regexp.Compile(`(\d+)-(\d+)(\.ts)$`)
		m := re.FindStringSubmatch(firstTs)
		if m == nil || len(m) < 3 {
			return urlList
		}
		curIdx, err := strconv.Atoi(m[1])
		if err != nil {
			return urlList
		}
		idx2, err := strconv.Atoi(m[2])
		if err != nil {
			return urlList
		}
		startIdx := curIdx - emedyMaxNumber
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx < downloadedIdx+1 {
			startIdx = downloadedIdx + 1
		}
		log.Printf("REMEDY: downloaded video number:%v, current video number:%v, download:%v-%v\n", downloadedIdx, curIdx, startIdx, curIdx-1)
		for i := startIdx; i < curIdx; i++ {

			url := fmt.Sprintf("%v/%v-%v.ts", baseurl, i, idx2-(curIdx-i))
			urlList = append(urlList, url)
		}
	}
	return urlList
}
