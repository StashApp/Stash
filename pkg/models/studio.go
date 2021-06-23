package models

type StudioReader interface {
	Find(id int) (*Studio, error)
	FindMany(ids []int) ([]*Studio, error)
	FindChildren(id int) ([]*Studio, error)
	FindByName(name string, nocase bool) (*Studio, error)
	FindByStashID(stashID string, stashboxEndpoint string) ([]*Studio, error)
	Count() (int, error)
	All() ([]*Studio, error)
	// TODO - this interface is temporary until the filter schema can fully
	// support the query needed
	QueryForAutoTag(words []string) ([]*Studio, error)
	Query(studioFilter *StudioFilterType, findFilter *FindFilterType) ([]*Studio, int, error)
	GetImage(studioID int) ([]byte, error)
	HasImage(studioID int) (bool, error)
	GetStashIDs(studioID int) ([]*StashID, error)
}

type StudioWriter interface {
	Create(newStudio Studio) (*Studio, error)
	Update(updatedStudio StudioPartial) (*Studio, error)
	UpdateFull(updatedStudio Studio) (*Studio, error)
	Destroy(id int) error
	UpdateImage(studioID int, image []byte) error
	DestroyImage(studioID int) error
	UpdateStashIDs(studioID int, stashIDs []StashID) error
}

type StudioReaderWriter interface {
	StudioReader
	StudioWriter
}
