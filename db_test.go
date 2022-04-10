package slashscheduler

import (
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)
	// Create a new data base
	db, err := memdb.NewMemDB(&memdb.DBSchema{
		Tables: MemDBSchema,
	})
	if err != nil {
		panic(err)
	}

	// Create a write transaction
	txn := slashSchedulerTxn{db.Txn(true)}

	s, err := txn.get("5")
	assert.NoError(err)
	assert.Equal("5", s.GuildID)
	err = txn.replace(s, schedule{
		GuildID:     s.GuildID,
		Title:       "4321",
		Description: "1234",
		Recurring:   false,
		Timestamp:   0,
		ChannelID:   "",
	})
	assert.NoError(err)
	s, err = txn.get("5")
	assert.NoError(err)
	assert.Equal("5", s.GuildID)
	assert.Equal("4321", s.Title)
	assert.Equal("1234", s.Description)
}
