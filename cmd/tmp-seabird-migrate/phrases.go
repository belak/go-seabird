package main

import (
	seabird "github.com/belak/go-seabird"
	"github.com/belak/nut"
	"github.com/go-xorm/xorm"
)

// Phrase is the v1 xorm model for phrases
type Phrase struct {
	ID        int64
	Name      string `xorm:"index"`
	Value     string
	Submitter string
	Deleted   bool
}

// phraseBucket is the old nut.DB phrase store
type phraseBucket struct {
	Key     string
	Entries []struct {
		Value     string
		Submitter string
		Deleted   bool
	}
}

func migratePhrases(b *seabird.Bot, ndb *nut.DB, xdb *xorm.Engine) error {
	l := b.GetLogger()

	err := xdb.Sync(Phrase{})
	if err != nil {
		return err
	}

	rowCount, err := xdb.Count(Phrase{})
	if err != nil {
		return err
	}

	if rowCount != 0 {
		l.Warn("Skipping phrases migration because target table is non-empty")
		return nil
	}

	l.Info("Migrating phrases from nut to xorm")

	// This is a bit gross, but it's the simplest way to get a transaction for both nut and xorm.
	l.Info("Migrating phrases from nut to xorm")

	// This is a bit gross, but it's the simplest way to get a transaction for both nut and xorm.
	return ndb.View(func(tx *nut.Tx) error {
		_, innerErr := xdb.Transaction(func(s *xorm.Session) (interface{}, error) {
			bucket := tx.Bucket("phrases")
			if bucket == nil {
				l.Info("Skipping phrases migration because of missing bucket")
				return nil, nil
			}

			data := &phraseBucket{}
			c := bucket.Cursor()
			for k, e := c.First(&data); e == nil; k, e = c.Next(&data) {
				l.Infof("Migrating phrase entry for %s", data.Key)

				if data.Key != k {
					l.Warnf("Phrase name (%s) does not match key (%s)", data.Key, k)
				}

				for _, entry := range data.Entries {
					phrase := Phrase{
						Name:      data.Key,
						Value:     entry.Value,
						Submitter: entry.Submitter,
						Deleted:   entry.Deleted,
					}

					_, err = s.InsertOne(phrase)
					if err != nil {
						return nil, err
					}
				}
			}

			return nil, err
		})

		return innerErr
	})
}
