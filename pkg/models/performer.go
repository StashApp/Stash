package models

import (
	"github.com/jmoiron/sqlx"
)

type PerformerReader interface {
	// Find(id int) (*Performer, error)
	// FindBySceneID(sceneID int) ([]*Performer, error)
	// FindNameBySceneID(sceneID int) ([]*Performer, error)
	// FindByNames(names []string, nocase bool) ([]*Performer, error)
	// Count() (int, error)
	// All() ([]*Performer, error)
	// AllSlim() ([]*Performer, error)
	// Query(performerFilter *PerformerFilterType, findFilter *FindFilterType) ([]*Performer, int)
	GetPerformerImage(performerID int) ([]byte, error)
}

type PerformerWriter interface {
	// Create(newPerformer Performer) (*Performer, error)
	// Update(updatedPerformer Performer) (*Performer, error)
	// Destroy(id string) error
	// UpdatePerformerImage(performerID int, image []byte) error
	// DestroyPerformerImage(performerID int) error
}

type PerformerReaderWriter interface {
	PerformerReader
	PerformerWriter
}

func NewPerformerReaderWriter(tx *sqlx.Tx) PerformerReaderWriter {
	return &performerReaderWriter{
		tx: tx,
		qb: NewPerformerQueryBuilder(),
	}
}

type performerReaderWriter struct {
	tx *sqlx.Tx
	qb PerformerQueryBuilder
}

func (t *performerReaderWriter) GetPerformerImage(performerID int) ([]byte, error) {
	return t.qb.GetPerformerImage(performerID, t.tx)
}
