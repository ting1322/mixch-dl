package twitcasting

import (
	_ "embed"
	"testing"
	"bytes"
)

//go:embed testdata/chat1.json
var chatjson1 string

func TestParseChatData(t *testing.T) {
	chat := Chat{}
	buf := &bytes.Buffer{}
	chat.parseChatData([]byte(chatjson1), buf, 0)

	text := buf.String()
	pattern := `{"replayChatItemAction":{"actions":[{"addChatItemAction":{"item":{"liveChatTextMessageRenderer":{"authorName":{"simpleText":"ゆく"},"message":{"runs":[{"text":"なになになになに。LINE@の教えてください( т т )"}]}}}}}],"videoOffsetTimeMsec":"0"}}
`
	if text != pattern {
		t.Fatalf("text != pattern, text=%v", text)
	}
}