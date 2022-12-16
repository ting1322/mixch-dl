package inter

import (
	"log"
	"os"
	"os/exec"
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
