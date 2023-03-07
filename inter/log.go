package inter

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

var (
	JsonOutput    bool = false
	VerboseOutput bool = false
)

type downloadProgressData struct {
	MsgType   string `json:"type"`
	VideoFrag int    `json:"video_frag"`
	ChatCount int    `json:"chat_count"`
	Duration  string `json:"duration"`
}

func LogProgress(vfrag, chatCount int, duration time.Duration) {
	if JsonOutput {
		d := downloadProgressData{
			MsgType:   "progress",
			VideoFrag: vfrag,
			ChatCount: chatCount,
		}
		if duration > 0 {
			d.Duration = duration.String()
		}
		jtext, _ := json.Marshal(d)
		fmt.Println(string(jtext))
	} else {
		fmt.Printf("downloaded video fragment: %d, duration: %v, chat: %v\n", vfrag, duration, chatCount)
	}
}

type StatusCode int

const (
	STATUS_WaitStream StatusCode = iota
	STATUS_Downloading
	STATUS_Finish
)

func (status StatusCode) String() string {
	return [...]string{
		"wait stream start",
		"downloading",
		"download finish",
	}[status]
}

func (status StatusCode) JsonStr() string {
	return [...]string{
		"waiting",
		"downloading",
		"finish",
	}[status]
}

type statusData struct {
	MsgType string `json:"type"`
	Status  string `json:"status"`
}

func LogStatus(status StatusCode) {
	if JsonOutput {
		d := statusData{
			MsgType: "status",
			Status:  status.JsonStr(),
		}
		jtext, _ := json.Marshal(d)
		fmt.Println(string(jtext))
	} else {
		fmt.Println()
		fmt.Println(status.String())
	}
}

type msgData struct {
	MsgType string `json:"type"`
	Msg     string `json:"msg"`
	Debug   bool   `json:"debug"`
}

func LogMsg(debug bool, msg string) {
	if JsonOutput {
		d := msgData{
			MsgType: "msg",
			Debug:   debug,
			Msg:     msg,
		}
		jtext, _ := json.Marshal(d)
		fmt.Println(string(jtext))
	} else {
		if !debug || (debug && VerboseOutput) {
			log.Println(msg)
		}
	}
}
