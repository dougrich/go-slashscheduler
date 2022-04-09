package slashscheduler_test

import (
	"testing"

	"github.com/dougrich/go-slashscheduler"
)

func TestPlaceholder(t *testing.T) {
	if slashscheduler.Placeholder() != 2 {
		t.Fatal("Expected a placeholder value of 2")
	}
}
