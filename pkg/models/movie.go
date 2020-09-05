package models

import (
	"github.com/jmoiron/sqlx"
)

type MovieReader interface {
	Find(id int) (*Movie, error)
	FindMany(ids []int) ([]*Movie, error)
	// FindBySceneID(sceneID int) ([]*Movie, error)
	FindByName(name string, nocase bool) (*Movie, error)
	// FindByNames(names []string, nocase bool) ([]*Movie, error)
	All() ([]*Movie, error)
	// AllSlim() ([]*Movie, error)
	// Query(movieFilter *MovieFilterType, findFilter *FindFilterType) ([]*Movie, int)
	GetFrontImage(movieID int) ([]byte, error)
	GetBackImage(movieID int) ([]byte, error)
}

type MovieWriter interface {
	Create(newMovie Movie) (*Movie, error)
	Update(updatedMovie MoviePartial) (*Movie, error)
	UpdateFull(updatedMovie Movie) (*Movie, error)
	// Destroy(id string) error
	UpdateMovieImages(movieID int, frontImage []byte, backImage []byte) error
	// DestroyMovieImages(movieID int) error
}

type MovieReaderWriter interface {
	MovieReader
	MovieWriter
}

func NewMovieReaderWriter(tx *sqlx.Tx) MovieReaderWriter {
	return &movieReaderWriter{
		tx: tx,
		qb: NewMovieQueryBuilder(),
	}
}

type movieReaderWriter struct {
	tx *sqlx.Tx
	qb MovieQueryBuilder
}

func (t *movieReaderWriter) Find(id int) (*Movie, error) {
	return t.qb.Find(id, t.tx)
}

func (t *movieReaderWriter) FindMany(ids []int) ([]*Movie, error) {
	return t.qb.FindMany(ids)
}

func (t *movieReaderWriter) FindByName(name string, nocase bool) (*Movie, error) {
	return t.qb.FindByName(name, t.tx, nocase)
}

func (t *movieReaderWriter) All() ([]*Movie, error) {
	return t.qb.All()
}

func (t *movieReaderWriter) GetFrontImage(movieID int) ([]byte, error) {
	return t.qb.GetFrontImage(movieID, t.tx)
}

func (t *movieReaderWriter) GetBackImage(movieID int) ([]byte, error) {
	return t.qb.GetBackImage(movieID, t.tx)
}

func (t *movieReaderWriter) Create(newMovie Movie) (*Movie, error) {
	return t.qb.Create(newMovie, t.tx)
}

func (t *movieReaderWriter) Update(updatedMovie MoviePartial) (*Movie, error) {
	return t.qb.Update(updatedMovie, t.tx)
}

func (t *movieReaderWriter) UpdateFull(updatedMovie Movie) (*Movie, error) {
	return t.qb.UpdateFull(updatedMovie, t.tx)
}

func (t *movieReaderWriter) UpdateMovieImages(movieID int, frontImage []byte, backImage []byte) error {
	return t.qb.UpdateMovieImages(movieID, frontImage, backImage, t.tx)
}
