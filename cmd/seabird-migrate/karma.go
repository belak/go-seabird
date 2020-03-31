package main

import (
	seabird "github.com/belak/go-seabird"
	"github.com/belak/nut"
	"xorm.io/xorm"
)

// Karma is the v1 xorm model for karma
type Karma struct {
	ID    int64
	Name  string `xorm:"unique"`
	Score int
}

func migrateKarma(b *seabird.Bot, ndb *nut.DB, xdb *xorm.Engine) error {
	l := seabird.CtxLogger(b.Context(), "migrate")

	// Migrate any relevant tables
	err := xdb.Sync(Karma{})
	if err != nil {
		return err
	}

	rowCount, err := xdb.Count(Karma{})
	if err != nil {
		return err
	}

	if rowCount != 0 {
		l.Warn("Skipping karma migration because target table is non-empty")
		return nil
	}

	// If a nut DB exists, we need to migrate all the data
	l.Info("Migrating karma from nut to xorm")

	// This is a bit gross, but it's the simplest way to get a transaction for both nut and xorm.
	return ndb.View(func(tx *nut.Tx) error {
		_, innerErr := xdb.Transaction(func(s *xorm.Session) (interface{}, error) {
			// We only need to migrate data if there's a karma bucket.
			bucket := tx.Bucket("karma")
			if bucket == nil {
				l.Info("Skipping karma migration because of missing bucket")
				return nil, nil
			}

			karma := &Karma{}

			c := bucket.Cursor()
			for k, e := c.First(&karma); e == nil; k, e = c.Next(&karma) {
				l.Infof("Migrating karma entry for %s", karma.Name)

				if karma.Name != k {
					l.Warnf("Karma name (%s) does not match key (%s)", karma.Name, k)
				}

				// Reset the ID before inserting
				karma.ID = 0

				// Actually insert
				_, err = s.InsertOne(karma)
				if err != nil {
					return nil, err
				}
			}

			return nil, err
		})

		return innerErr
	})
}
