package slashscheduler

import (
	"fmt"

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
