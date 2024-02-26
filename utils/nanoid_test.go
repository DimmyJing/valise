package utils_test

import (
	"testing"

	"github.com/DimmyJing/valise/utils"
	"github.com/stretchr/testify/assert"
)

func TestNanoID(t *testing.T) {
	t.Parallel()

	assert.Len(t, utils.NanoID(), 21)
}

func TestNanoIDAlpha(t *testing.T) {
	t.Parallel()

	assert.Len(t, utils.NanoIDAlpha(), 21)
}
