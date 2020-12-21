package manager

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/disintegration/imaging"
	"github.com/fvbommel/sortorder"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/utils"
)

type SpriteGenerator struct {
	Info *GeneratorInfo

	VideoChecksum   string
	ImageOutputPath string
	VTTOutputPath   string
	Rows            int
	Columns         int

	Overwrite bool
}

func NewSpriteGenerator(videoFile ffmpeg.VideoFile, videoChecksum string, imageOutputPath string, vttOutputPath string, rows int, cols int) (*SpriteGenerator, error) {
	exists, err := utils.FileExists(videoFile.Path)
	if !exists {
		return nil, err
	}
	generator, err := newGeneratorInfo(videoFile)
	if err != nil {
		return nil, err
	}
	generator.ChunkCount = rows * cols
	if err := generator.configure(); err != nil {
		return nil, err
	}

	return &SpriteGenerator{
		Info:            generator,
		VideoChecksum:   videoChecksum,
		ImageOutputPath: imageOutputPath,
		VTTOutputPath:   vttOutputPath,
		Rows:            rows,
		Columns:         cols,
	}, nil
}

func (g *SpriteGenerator) Generate() (error, int) {
	encoder := ffmpeg.NewEncoder(instance.FFMPEGPath)

	err, ffmpegErrCount := g.generateSpriteImage(&encoder)
	if err != nil {
		return err, ffmpegErrCount
	}
	if err := g.generateSpriteVTT(&encoder); err != nil {
		return err, ffmpegErrCount
	}
	return nil, ffmpegErrCount
}

func (g *SpriteGenerator) generateSpriteImage(encoder *ffmpeg.Encoder) (error, int) {
	if !g.Overwrite && g.imageExists() {
		return nil, 0
	}
	logger.Infof("[generator] generating sprite image for %s", g.Info.VideoFile.Path)

	encodingErrorCount := 0
	// Create `this.chunkCount` thumbnails in the tmp directory
	stepSize := g.Info.VideoFile.Duration / float64(g.Info.ChunkCount)
	for i := 0; i < g.Info.ChunkCount; i++ {
		time := float64(i) * stepSize
		num := fmt.Sprintf("%.3d", i)
		filename := "thumbnail_" + g.VideoChecksum + "_" + num + ".jpg"

		options := ffmpeg.ScreenshotOptions{
			OutputPath: instance.Paths.Generated.GetTmpPath(filename),
			Time:       time,
			Width:      160,
		}
		ffmpegerr, _ := encoder.Screenshot(g.Info.VideoFile, options)
		if ffmpegerr != "" {
			encodingErrorCount += 1
		}
	}

	// Combine all of the thumbnails into a sprite image
	globPath := filepath.Join(instance.Paths.Generated.Tmp, fmt.Sprintf("thumbnail_%s_*.jpg", g.VideoChecksum))
	imagePaths, _ := doublestar.Glob(globPath)
	sort.Sort(sortorder.Natural(imagePaths))
	var images []image.Image
	for _, imagePath := range imagePaths {
		img, err := imaging.Open(imagePath)
		if err != nil {
			return err, encodingErrorCount
		}
		images = append(images, img)
	}

	if len(images) == 0 {
		return fmt.Errorf("images slice is empty, failed to generate sprite images for %s", g.Info.VideoFile.Path), encodingErrorCount
	}
	width := images[0].Bounds().Size().X
	height := images[0].Bounds().Size().Y
	canvasWidth := width * g.Columns
	canvasHeight := height * g.Rows
	montage := imaging.New(canvasWidth, canvasHeight, color.NRGBA{})
	for index := 0; index < len(images); index++ {
		x := width * (index % g.Columns)
		y := height * int(math.Floor(float64(index)/float64(g.Rows)))
		img := images[index]
		montage = imaging.Paste(montage, img, image.Pt(x, y))
	}

	return imaging.Save(montage, g.ImageOutputPath), encodingErrorCount
}

func (g *SpriteGenerator) generateSpriteVTT(encoder *ffmpeg.Encoder) error {
	if !g.Overwrite && g.vttExists() {
		return nil
	}
	logger.Infof("[generator] generating sprite vtt for %s", g.Info.VideoFile.Path)

	spriteImage, err := os.Open(g.ImageOutputPath)
	if err != nil {
		return err
	}
	defer spriteImage.Close()
	spriteImageName := filepath.Base(g.ImageOutputPath)
	image, _, err := image.DecodeConfig(spriteImage)
	width := image.Width / g.Columns
	height := image.Height / g.Rows

	stepSize := float64(g.Info.NthFrame) / g.Info.FrameRate

	vttLines := []string{"WEBVTT", ""}
	for index := 0; index < g.Info.ChunkCount; index++ {
		x := width * (index % g.Columns)
		y := height * int(math.Floor(float64(index)/float64(g.Rows)))
		startTime := utils.GetVTTTime(float64(index) * stepSize)
		endTime := utils.GetVTTTime(float64(index+1) * stepSize)

		vttLines = append(vttLines, startTime+" --> "+endTime)
		vttLines = append(vttLines, fmt.Sprintf("%s#xywh=%d,%d,%d,%d", spriteImageName, x, y, width, height))
		vttLines = append(vttLines, "")
	}
	vtt := strings.Join(vttLines, "\n")

	return ioutil.WriteFile(g.VTTOutputPath, []byte(vtt), 0644)
}

func (g *SpriteGenerator) imageExists() bool {
	exists, _ := utils.FileExists(g.ImageOutputPath)
	return exists
}

func (g *SpriteGenerator) vttExists() bool {
	exists, _ := utils.FileExists(g.VTTOutputPath)
	return exists
}
