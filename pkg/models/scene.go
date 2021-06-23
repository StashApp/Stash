package models

type SceneReader interface {
	Find(id int) (*Scene, error)
	FindMany(ids []int) ([]*Scene, error)
	FindByChecksum(checksum string) (*Scene, error)
	FindByOSHash(oshash string) (*Scene, error)
	FindByPath(path string) (*Scene, error)
	FindByPerformerID(performerID int) ([]*Scene, error)
	FindByGalleryID(performerID int) ([]*Scene, error)
	FindDuplicates(distance int) ([][]*Scene, error)
	FindByStashIDStatus(hasStashID bool, stashboxEndpoint string) ([]*Scene, error)
	CountByPerformerID(performerID int) (int, error)
	// FindByStudioID(studioID int) ([]*Scene, error)
	FindByMovieID(movieID int) ([]*Scene, error)
	CountByMovieID(movieID int) (int, error)
	Count() (int, error)
	Size() (float64, error)
	// SizeCount() (string, error)
	CountByStudioID(studioID int) (int, error)
	CountByTagID(tagID int) (int, error)
	CountMissingChecksum() (int, error)
	CountMissingOSHash() (int, error)
	Wall(q *string) ([]*Scene, error)
	All() ([]*Scene, error)
	Query(sceneFilter *SceneFilterType, findFilter *FindFilterType) ([]*Scene, int, error)
	GetCover(sceneID int) ([]byte, error)
	GetMovies(sceneID int) ([]MoviesScenes, error)
	GetTagIDs(sceneID int) ([]int, error)
	GetGalleryIDs(sceneID int) ([]int, error)
	GetPerformerIDs(sceneID int) ([]int, error)
	GetStashIDs(sceneID int) ([]*StashID, error)
}

type SceneWriter interface {
	Create(newScene Scene) (*Scene, error)
	Update(updatedScene ScenePartial) (*Scene, error)
	UpdateFull(updatedScene Scene) (*Scene, error)
	IncrementOCounter(id int) (int, error)
	DecrementOCounter(id int) (int, error)
	ResetOCounter(id int) (int, error)
	UpdateFileModTime(id int, modTime NullSQLiteTimestamp) error
	Destroy(id int) error
	UpdateCover(sceneID int, cover []byte) error
	DestroyCover(sceneID int) error
	UpdatePerformers(sceneID int, performerIDs []int) error
	UpdateTags(sceneID int, tagIDs []int) error
	UpdateGalleries(sceneID int, galleryIDs []int) error
	UpdateMovies(sceneID int, movies []MoviesScenes) error
	UpdateStashIDs(sceneID int, stashIDs []StashID) error
}

type SceneReaderWriter interface {
	SceneReader
	SceneWriter
}
