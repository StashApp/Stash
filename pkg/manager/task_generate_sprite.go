package manager

import (
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
	"sync"
)

type GenerateSpriteTask struct {
	Scene models.Scene
}

func (t *GenerateSpriteTask) Start(wg *sync.WaitGroup,spritesCh chan<- struct{},errorCh chan<- struct{}) {
	defer wg.Done()

	if t.doesSpriteExist(t.Scene.Checksum) {
		return
	}

	videoFile, err := ffmpeg.NewVideoFile(instance.FFProbePath, t.Scene.Path)
	if err != nil {
		logger.Errorf("error reading video file: %s", err.Error())
		errorCh <- struct{}{}
		return
	}

	imagePath := instance.Paths.Scene.GetSpriteImageFilePath(t.Scene.Checksum)
	vttPath := instance.Paths.Scene.GetSpriteVttFilePath(t.Scene.Checksum)
	generator, err := NewSpriteGenerator(*videoFile, imagePath, vttPath, 9, 9)
	if err != nil {
		logger.Errorf("error creating sprite generator: %s", err.Error())
		errorCh <- struct{}{}
		return
	}

	if err := generator.Generate(); err != nil {
		logger.Errorf("error generating sprite: %s", err.Error())
		errorCh <- struct{}{}
		return
	}
	spritesCh <- struct{}{} // since we got here that means we generated a new sprite
}

func (t *GenerateSpriteTask) doesSpriteExist(sceneChecksum string) bool {
	imageExists, _ := utils.FileExists(instance.Paths.Scene.GetSpriteImageFilePath(sceneChecksum))
	vttExists, _ := utils.FileExists(instance.Paths.Scene.GetSpriteVttFilePath(sceneChecksum))
	return imageExists && vttExists
}
