package inter

import (
	"log"
	"os"
	"os/exec"
)

func FfmpegMerge(in, out string, fixts bool) {
	log.Printf("merge %v file by ffmpeg\n", out)
	var cmd *exec.Cmd
	if fixts {
		cmd = exec.Command(
			"ffmpeg", "-i", in,
			"-c", "copy",
			"-bsf", "setts=ts=TS-STARTPTS",
			"-map", "0",
			"-dn", "-ignore_unknown",
			"-ss", "1ms",
			"-movflags", "+faststart",
			out)
	} else {
		cmd = exec.Command(
			"ffmpeg", "-i", in,
			"-c", "copy",
			"-map", "0",
			"-dn", "-ignore_unknown",
			"-movflags", "+faststart",
			out)
	}
	err := cmd.Run()
	if err == nil {
		os.Remove(in)
		//os.Rename(in, in[:len(in)-5])
	}

}