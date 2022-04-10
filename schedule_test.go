package slashscheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScheduleMessage(t *testing.T) {
	assert := assert.New(t)
	s := schedule{}
	assert.Equal("schedule is **disabled**", s.Message())
	s.Timestamp = time.Now().Unix() + 5000
	assert.Equal(fmt.Sprintf("schedule is **enabled**, next game is at <t:%d:T>", s.Timestamp), s.Message())
}
