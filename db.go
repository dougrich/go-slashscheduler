package slashscheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-memdb"
)

const (
	TablenameSchedule = "schedule"
)

var (
	ErrGuildIDEmpty = fmt.Errorf("")
	MemDBSchema     = map[string]*memdb.TableSchema{
		"schedule": &memdb.TableSchema{
			Name: TablenameSchedule,
			Indexes: map[string]*memdb.IndexSchema{
				"id": &memdb.IndexSchema{
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "GuildID"},
				},
				"timestamp": &memdb.IndexSchema{
					Name:    "timestamp",
					Indexer: &memdb.IntFieldIndex{Field: "Timestamp"},
				},
			},
		},
	}
)

type slashSchedulerTxn struct {
	*memdb.Txn
}

func (db slashSchedulerTxn) get(guildID string) (*Schedule, error) {
	s, err := db.First(TablenameSchedule, "id", guildID)
	if err != nil && err != memdb.ErrNotFound {
		return nil, err
	} else if s == nil {
		return &Schedule{
			GuildID: guildID,
		}, nil
	} else {
		return s.(*Schedule), nil
	}
}

func (db slashSchedulerTxn) replace(s *Schedule, s2 Schedule) error {
	if s.GuildID == "" {
		return ErrGuildIDEmpty
	}
	err := db.Delete(TablenameSchedule, s)
	if err != nil && err != memdb.ErrNotFound {
		return err
	}
	return db.Insert(TablenameSchedule, &s2)
}

func (db slashSchedulerTxn) pending(from time.Time, until time.Time) (chan *Schedule, error) {
	it, err := db.LowerBound(TablenameSchedule, "timestamp", from.Unix())
	if err != nil {
		return nil, err
	}
	c := make(chan *Schedule)
	go func() {
		log.Print("slashscheduler:db: starting pending loop")
		for obj := it.Next(); obj != nil; obj = it.Next() {
			s := obj.(*Schedule)
			if s.Timestamp > until.Unix() {
				break
			}
			c <- s
		}
		log.Print("slashscheduler:db: finished pending loop")
		close(c)
	}()
	return c, nil
}
