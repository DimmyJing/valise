package utils_test

import (
	"testing"

	"github.com/DimmyJing/valise/utils"
	"github.com/stretchr/testify/assert"
)

func TestNanoID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 21, len(utils.NanoID()))
}

func TestNanoIDAlpha(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 21, len(utils.NanoIDAlpha()))
}
