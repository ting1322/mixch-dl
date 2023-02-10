package inter

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func FfmpegMerge(in, out string, fixts bool) {
	log.Printf("merge %v file by ffmpeg\n", out)
	cmdarg := []string{"-i", in, "-c", "copy"}
	if fixts {
		cmdarg = append(cmdarg, "-bsf", "setts=ts=TS-STARTPTS")
	}
	cmdarg = append(cmdarg, "-map", "0", "-dn", "-ignore_unknown")
	if fixts {
		cmdarg = append(cmdarg, "-ss", "1ms")
	}
	cmdarg = append(cmdarg, "-movflags", "+faststart", out)
	var cmd *exec.Cmd = exec.Command("ffmpeg", cmdarg...)
	log.Println(cmd)
	err := cmd.Run()
	if err == nil {
		os.Remove(in)
		//os.Rename(in, in[:len(in)-5])
	}
}

func FfmpegAttachThumbnail(video, img string, disposition int) {
	log.Printf("attach cover image to video")
	outfile := "temp-" + video
	cmdarg := []string{"-i", video, "-i", img,
		"-map", "0", "-map", "1",
		fmt.Sprintf("-disposition:%d", disposition), "attached_pic",
		"-c", "copy",
		"-movflags", "+faststart",
		outfile}
	var cmd *exec.Cmd = exec.Command("ffmpeg", cmdarg...)
	log.Println(cmd)
	err := cmd.Run()
	if err == nil {
		bak := video + ".bak"
		log.Printf("rename %v to %v\n", video, bak)
		err = os.Rename(video, bak)
		if err != nil {
			log.Println("RENAME FAIL", err)
			return
		}
		log.Printf("rename %v to %v\n", outfile, video)
		err = os.Rename(outfile, video)
		if err != nil {
			log.Println("RENAME FAIL", err)
			return
		}
		log.Printf("rm %v\n", bak)
		os.Remove(bak)
	}
}

func FfprobeTime(filename string) (time.Duration, error) {
	cmdarg := []string{"-v", "error",
		"-show_entries", "format=duration", "-of",
		"default=noprint_wrappers=1:nokey=1", filename}
	var cmd *exec.Cmd = exec.Command("ffprobe", cmdarg...)
	log.Println(cmd)
	outData, err := cmd.Output()
	text := strings.Trim(string(outData), " \r\n")
	log.Println("FFPROBE:", text)
	if err != nil {
		return 0, err
	}
	durationFloat, err := strconv.ParseFloat(text, 64)
	if err != nil {
		log.Println("FFPROBE ERR:", err)
		return 0, err
	}
	floatTime := durationFloat * float64(time.Second)
	return time.Duration(math.Round(floatTime)), nil
}
