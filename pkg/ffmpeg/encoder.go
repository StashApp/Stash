package ffmpeg

import (
	"github.com/stashapp/stash/pkg/logger"
	"io/ioutil"
	"os/exec"
)

type Encoder struct {
	Path string
}

func NewEncoder(ffmpegPath string) Encoder {
	return Encoder{
		Path: ffmpegPath,
	}
}

func (e *Encoder) run(probeResult VideoFile, args []string) (string, error) {
	cmd := exec.Command(e.Path, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("FFMPEG stderr not available: " + err.Error())
	}

	stdout, err := cmd.StdoutPipe()
	if nil != err {
		logger.Error("FFMPEG stdout not available: " + err.Error())
	}

	if err = cmd.Start(); err != nil {
		return "", err
	}

	buf := make([]byte, 80)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			data := string(buf[0:n])
			time := GetTimeFromRegex(data)
			if time > 0 && probeResult.Duration > 0 {
				progress := time / probeResult.Duration
				logger.Infof("Progress %.2f", progress)
			}
		}
		if err != nil {
			break
		}
	}

	stdoutData, _ := ioutil.ReadAll(stdout)
	stdoutString := string(stdoutData)

	if err := cmd.Wait(); err != nil {
		return stdoutString, err
	}

	return stdoutString, nil
}
