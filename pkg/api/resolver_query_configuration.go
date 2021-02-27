package api

import (
	"context"

	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *queryResolver) Configuration(ctx context.Context) (*models.ConfigResult, error) {
	return makeConfigResult(), nil
}

func (r *queryResolver) Directory(ctx context.Context, path *string) (*models.Directory, error) {
	var dirPath = ""
	if path != nil {
		dirPath = *path
	}
	currentDir := utils.GetDir(dirPath)

	return &models.Directory{
		Path:        currentDir,
		Parent:      utils.GetParent(currentDir),
		Directories: utils.ListDir(currentDir),
	}, nil
}

func makeConfigResult() *models.ConfigResult {
	return &models.ConfigResult{
		General:   makeConfigGeneralResult(),
		Interface: makeConfigInterfaceResult(),
	}
}

func makeConfigGeneralResult() *models.ConfigGeneralResult {
	logFile := config.GetLogFile()

	maxTranscodeSize := config.GetMaxTranscodeSize()
	maxStreamingTranscodeSize := config.GetMaxStreamingTranscodeSize()

	scraperUserAgent := config.GetScraperUserAgent()
	scraperCDPPath := config.GetScraperCDPPath()

	return &models.ConfigGeneralResult{
		Stashes:                       config.GetStashPaths(),
		DatabasePath:                  config.GetDatabasePath(),
		GeneratedPath:                 config.GetGeneratedPath(),
		CachePath:                     config.GetCachePath(),
		CalculateMd5:                  config.IsCalculateMD5(),
		VideoFileNamingAlgorithm:      config.GetVideoFileNamingAlgorithm(),
		ParallelTasks:                 config.GetParallelTasks(),
		PreviewSegments:               config.GetPreviewSegments(),
		PreviewSegmentDuration:        config.GetPreviewSegmentDuration(),
		PreviewExcludeStart:           config.GetPreviewExcludeStart(),
		PreviewExcludeEnd:             config.GetPreviewExcludeEnd(),
		PreviewPreset:                 config.GetPreviewPreset(),
		TranscodeHardwareAcceleration: config.GetTranscodeHardwareAcceleration(),
		MaxTranscodeSize:              &maxTranscodeSize,
		MaxStreamingTranscodeSize:     &maxStreamingTranscodeSize,
		Username:                      config.GetUsername(),
		Password:                      config.GetPasswordHash(),
		MaxSessionAge:                 config.GetMaxSessionAge(),
		LogFile:                       &logFile,
		LogOut:                        config.GetLogOut(),
		LogLevel:                      config.GetLogLevel(),
		LogAccess:                     config.GetLogAccess(),
		VideoExtensions:               config.GetVideoExtensions(),
		ImageExtensions:               config.GetImageExtensions(),
		GalleryExtensions:             config.GetGalleryExtensions(),
		CreateGalleriesFromFolders:    config.GetCreateGalleriesFromFolders(),
		Excludes:                      config.GetExcludes(),
		ImageExcludes:                 config.GetImageExcludes(),
		ScraperUserAgent:              &scraperUserAgent,
		ScraperCDPPath:                &scraperCDPPath,
		StashBoxes:                    config.GetStashBoxes(),
	}
}

func makeConfigInterfaceResult() *models.ConfigInterfaceResult {
	menuItems := config.GetMenuItems()
	soundOnPreview := config.GetSoundOnPreview()
	wallShowTitle := config.GetWallShowTitle()
	wallPlayback := config.GetWallPlayback()
	maximumLoopDuration := config.GetMaximumLoopDuration()
	autostartVideo := config.GetAutostartVideo()
	showStudioAsText := config.GetShowStudioAsText()
	css := config.GetCSS()
	cssEnabled := config.GetCSSEnabled()
	language := config.GetLanguage()

	return &models.ConfigInterfaceResult{
		MenuItems:           menuItems,
		SoundOnPreview:      &soundOnPreview,
		WallShowTitle:       &wallShowTitle,
		WallPlayback:        &wallPlayback,
		MaximumLoopDuration: &maximumLoopDuration,
		AutostartVideo:      &autostartVideo,
		ShowStudioAsText:    &showStudioAsText,
		CSS:                 &css,
		CSSEnabled:          &cssEnabled,
		Language:            &language,
	}
}
