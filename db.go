package slashscheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-memdb"
)

const (
	tablenameSchedule = "schedule"
)

var (
	ErrGuildIDEmpty = fmt.Errorf("")
	MemDBSchema     = map[string]*memdb.TableSchema{
		"schedule": &memdb.TableSchema{
			Name: tablenameSchedule,
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

func (db slashSchedulerTxn) get(guildID string) (*schedule, error) {
	s, err := db.First(tablenameSchedule, "id", guildID)
	if err != nil && err != memdb.ErrNotFound {
		return nil, err
	} else if s == nil {
		return &schedule{
			GuildID: guildID,
		}, nil
	} else {
		return s.(*schedule), nil
	}
}

func (db slashSchedulerTxn) replace(s *schedule, s2 schedule) error {
	if s.GuildID == "" {
		return ErrGuildIDEmpty
	}
	err := db.Delete(tablenameSchedule, s)
	if err != nil && err != memdb.ErrNotFound {
		return err
	}
	err = db.Insert(tablenameSchedule, &s2)
	if err != nil {
		return err
	}
	return nil
}

func (db slashSchedulerTxn) pending(from time.Time, until time.Time) (chan *schedule, error) {
	it, err := db.LowerBound(tablenameSchedule, "timestamp", from.Unix())
	if err != nil {
		return nil, err
	}
	c := make(chan *schedule)
	go func() {
		log.Print("slashscheduler:db: starting pending loop")
		for obj := it.Next(); obj != nil; obj = it.Next() {
			s := obj.(*schedule)
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
