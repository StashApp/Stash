package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/manager"
	"github.com/stashapp/stash/pkg/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *mutationResolver) ConfigureGeneral(ctx context.Context, input models.ConfigGeneralInput) (*models.ConfigGeneralResult, error) {
	if len(input.Stashes) > 0 {
		for _, s := range input.Stashes {
			exists, err := utils.DirExists(s.Path)
			if !exists {
				return makeConfigGeneralResult(), err
			}
		}
		config.Set(config.Stash, input.Stashes)
	}

	if input.DatabasePath != nil {
		ext := filepath.Ext(*input.DatabasePath)
		if ext != ".db" && ext != ".sqlite" && ext != ".sqlite3" {
			return makeConfigGeneralResult(), fmt.Errorf("invalid database path, use extension db, sqlite, or sqlite3")
		}
		config.Set(config.Database, input.DatabasePath)
	}

	if input.GeneratedPath != nil {
		if err := utils.EnsureDir(*input.GeneratedPath); err != nil {
			return makeConfigGeneralResult(), err
		}
		config.Set(config.Generated, input.GeneratedPath)
	}

	if input.CachePath != nil {
		if err := utils.EnsureDir(*input.CachePath); err != nil {
			return makeConfigGeneralResult(), err
		}
		config.Set(config.Cache, input.CachePath)
	}

	if !input.CalculateMd5 && input.VideoFileNamingAlgorithm == models.HashAlgorithmMd5 {
		return makeConfigGeneralResult(), errors.New("calculateMD5 must be true if using MD5")
	}

	if input.VideoFileNamingAlgorithm != config.GetVideoFileNamingAlgorithm() {
		// validate changing VideoFileNamingAlgorithm
		if err := manager.ValidateVideoFileNamingAlgorithm(r.txnManager, input.VideoFileNamingAlgorithm); err != nil {
			return makeConfigGeneralResult(), err
		}

		config.Set(config.VideoFileNamingAlgorithm, input.VideoFileNamingAlgorithm)
	}

	config.Set(config.CalculateMD5, input.CalculateMd5)

	if input.ParallelTasks != nil {
		config.Set(config.ParallelTasks, *input.ParallelTasks)
	}
	if input.PreviewSegments != nil {
		config.Set(config.PreviewSegments, *input.PreviewSegments)
	}
	if input.PreviewSegmentDuration != nil {
		config.Set(config.PreviewSegmentDuration, *input.PreviewSegmentDuration)
	}
	if input.PreviewExcludeStart != nil {
		config.Set(config.PreviewExcludeStart, *input.PreviewExcludeStart)
	}
	if input.PreviewExcludeEnd != nil {
		config.Set(config.PreviewExcludeEnd, *input.PreviewExcludeEnd)
	}
	if input.PreviewPreset != nil {
		config.Set(config.PreviewPreset, input.PreviewPreset.String())
	}

	if input.TranscodeHardwareAcceleration != nil {
		config.Set(config.TranscodeHardwareAcceleration, *input.TranscodeHardwareAcceleration)
	}

	if input.MaxTranscodeSize != nil {
		config.Set(config.MaxTranscodeSize, input.MaxTranscodeSize.String())
	}

	if input.MaxStreamingTranscodeSize != nil {
		config.Set(config.MaxStreamingTranscodeSize, input.MaxStreamingTranscodeSize.String())
	}

	if input.Username != nil {
		config.Set(config.Username, input.Username)
	}

	if input.Password != nil {
		// bit of a hack - check if the passed in password is the same as the stored hash
		// and only set if they are different
		currentPWHash := config.GetPasswordHash()

		if *input.Password != currentPWHash {
			config.SetPassword(*input.Password)
		}
	}

	if input.MaxSessionAge != nil {
		config.Set(config.MaxSessionAge, *input.MaxSessionAge)
	}

	if input.LogFile != nil {
		config.Set(config.LogFile, input.LogFile)
	}

	config.Set(config.LogOut, input.LogOut)
	config.Set(config.LogAccess, input.LogAccess)

	if input.LogLevel != config.GetLogLevel() {
		config.Set(config.LogLevel, input.LogLevel)
		logger.SetLogLevel(input.LogLevel)
	}

	if input.Excludes != nil {
		config.Set(config.Exclude, input.Excludes)
	}

	if input.ImageExcludes != nil {
		config.Set(config.ImageExclude, input.ImageExcludes)
	}

	if input.VideoExtensions != nil {
		config.Set(config.VideoExtensions, input.VideoExtensions)
	}

	if input.ImageExtensions != nil {
		config.Set(config.ImageExtensions, input.ImageExtensions)
	}

	if input.GalleryExtensions != nil {
		config.Set(config.GalleryExtensions, input.GalleryExtensions)
	}

	config.Set(config.CreateGalleriesFromFolders, input.CreateGalleriesFromFolders)

	refreshScraperCache := false
	if input.ScraperUserAgent != nil {
		config.Set(config.ScraperUserAgent, input.ScraperUserAgent)
		refreshScraperCache = true
	}

	if input.ScraperCDPPath != nil {
		config.Set(config.ScraperCDPPath, input.ScraperCDPPath)
		refreshScraperCache = true
	}

	if input.StashBoxes != nil {
		if err := config.ValidateStashBoxes(input.StashBoxes); err != nil {
			return nil, err
		}
		config.Set(config.StashBoxes, input.StashBoxes)
	}

	if err := config.Write(); err != nil {
		return makeConfigGeneralResult(), err
	}

	manager.GetInstance().RefreshConfig()
	if refreshScraperCache {
		manager.GetInstance().RefreshScraperCache()
	}

	return makeConfigGeneralResult(), nil
}

func (r *mutationResolver) ConfigureInterface(ctx context.Context, input models.ConfigInterfaceInput) (*models.ConfigInterfaceResult, error) {
	if input.MenuItems != nil {
		config.Set(config.MenuItems, input.MenuItems)
	}

	if input.SoundOnPreview != nil {
		config.Set(config.SoundOnPreview, *input.SoundOnPreview)
	}

	if input.WallShowTitle != nil {
		config.Set(config.WallShowTitle, *input.WallShowTitle)
	}

	if input.WallPlayback != nil {
		config.Set(config.WallPlayback, *input.WallPlayback)
	}

	if input.MaximumLoopDuration != nil {
		config.Set(config.MaximumLoopDuration, *input.MaximumLoopDuration)
	}

	if input.AutostartVideo != nil {
		config.Set(config.AutostartVideo, *input.AutostartVideo)
	}

	if input.ShowStudioAsText != nil {
		config.Set(config.ShowStudioAsText, *input.ShowStudioAsText)
	}

	if input.Language != nil {
		config.Set(config.Language, *input.Language)
	}

	css := ""

	if input.CSS != nil {
		css = *input.CSS
	}

	config.SetCSS(css)

	if input.CSSEnabled != nil {
		config.Set(config.CSSEnabled, *input.CSSEnabled)
	}

	if err := config.Write(); err != nil {
		return makeConfigInterfaceResult(), err
	}

	return makeConfigInterfaceResult(), nil
}
