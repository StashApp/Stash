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

	directory := &models.Directory{}
	var err error

	var dirPath = ""
	if path != nil {
		dirPath = *path
	}
	currentDir := utils.GetDir(dirPath)
	directories, err := utils.ListDir(currentDir)
	if err != nil {
		return directory, err
	}

	directory.Path = currentDir
	directory.Parent = utils.GetParent(currentDir)
	directory.Directories = directories

	return directory, err
}

func makeConfigResult() *models.ConfigResult {
	return &models.ConfigResult{
		General:   makeConfigGeneralResult(),
		Interface: makeConfigInterfaceResult(),
		Dlna:      makeConfigDLNAResult(),
	}
}

func makeConfigGeneralResult() *models.ConfigGeneralResult {
	config := config.GetInstance()
	logFile := config.GetLogFile()

	maxTranscodeSize := config.GetMaxTranscodeSize()
	maxStreamingTranscodeSize := config.GetMaxStreamingTranscodeSize()

	scraperUserAgent := config.GetScraperUserAgent()
	scraperCDPPath := config.GetScraperCDPPath()

	return &models.ConfigGeneralResult{
		Stashes:                    config.GetStashPaths(),
		DatabasePath:               config.GetDatabasePath(),
		GeneratedPath:              config.GetGeneratedPath(),
		ConfigFilePath:             config.GetConfigFilePath(),
		ScrapersPath:               config.GetScrapersPath(),
		CachePath:                  config.GetCachePath(),
		CalculateMd5:               config.IsCalculateMD5(),
		VideoFileNamingAlgorithm:   config.GetVideoFileNamingAlgorithm(),
		ParallelTasks:              config.GetParallelTasks(),
		PreviewAudio:               config.GetPreviewAudio(),
		PreviewSegments:            config.GetPreviewSegments(),
		PreviewSegmentDuration:     config.GetPreviewSegmentDuration(),
		PreviewExcludeStart:        config.GetPreviewExcludeStart(),
		PreviewExcludeEnd:          config.GetPreviewExcludeEnd(),
		PreviewPreset:              config.GetPreviewPreset(),
		MaxTranscodeSize:           &maxTranscodeSize,
		MaxStreamingTranscodeSize:  &maxStreamingTranscodeSize,
		APIKey:                     config.GetAPIKey(),
		Username:                   config.GetUsername(),
		Password:                   config.GetPasswordHash(),
		MaxSessionAge:              config.GetMaxSessionAge(),
		LogFile:                    &logFile,
		LogOut:                     config.GetLogOut(),
		LogLevel:                   config.GetLogLevel(),
		LogAccess:                  config.GetLogAccess(),
		VideoExtensions:            config.GetVideoExtensions(),
		ImageExtensions:            config.GetImageExtensions(),
		GalleryExtensions:          config.GetGalleryExtensions(),
		CreateGalleriesFromFolders: config.GetCreateGalleriesFromFolders(),
		Excludes:                   config.GetExcludes(),
		ImageExcludes:              config.GetImageExcludes(),
		ScraperUserAgent:           &scraperUserAgent,
		ScraperCertCheck:           config.GetScraperCertCheck(),
		ScraperCDPPath:             &scraperCDPPath,
		StashBoxes:                 config.GetStashBoxes(),
	}
}

func makeConfigInterfaceResult() *models.ConfigInterfaceResult {
	config := config.GetInstance()
	menuItems := config.GetMenuItems()
	soundOnPreview := config.GetSoundOnPreview()
	wallShowTitle := config.GetWallShowTitle()
	customPerformerImageLocation := config.GetCustomPerformerImageLocation()
	wallPlayback := config.GetWallPlayback()
	maximumLoopDuration := config.GetMaximumLoopDuration()
	autostartVideo := config.GetAutostartVideo()
	showStudioAsText := config.GetShowStudioAsText()
	css := config.GetCSS()
	cssEnabled := config.GetCSSEnabled()
	language := config.GetLanguage()
	slideshowDelay := config.GetSlideshowDelay()
	handyKey := config.GetHandyKey()

	return &models.ConfigInterfaceResult{
		MenuItems:           					menuItems,
		SoundOnPreview:      					&soundOnPreview,
		WallShowTitle:       					&wallShowTitle,
		CustomPerformerImageLocation:                            &customPerformerImageLocation,
		WallPlayback:       					&wallPlayback,
		MaximumLoopDuration:					&maximumLoopDuration,
		AutostartVideo:    					  &autostartVideo,
		ShowStudioAsText:   					&showStudioAsText,
		CSS:               					  &css,
		CSSEnabled:        					  &cssEnabled,
		Language:         					  &language,
		SlideshowDelay:  					    &slideshowDelay,
		HandyKey:       					    &handyKey,
	}
}

func makeConfigDLNAResult() *models.ConfigDLNAResult {
	config := config.GetInstance()

	return &models.ConfigDLNAResult{
		ServerName:     config.GetDLNAServerName(),
		Enabled:        config.GetDLNADefaultEnabled(),
		WhitelistedIPs: config.GetDLNADefaultIPWhitelist(),
		Interfaces:     config.GetDLNAInterfaces(),
	}
}
