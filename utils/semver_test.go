package utils_test

import (
	"testing"

	"github.com/DimmyJing/valise/utils"
	"github.com/stretchr/testify/assert"
)

func TestSemver(t *testing.T) {
	t.Parallel()

	val, err := utils.CompareSemVer("1.0.0", "1.0.1")
	assert.NoError(t, err)
	assert.Negative(t, val)

	val, err = utils.CompareSemVer("1.0.0", "1.0.0")
	assert.NoError(t, err)
	assert.Zero(t, val)

	val, err = utils.CompareSemVer("1.0.1", "1.0.0")
	assert.NoError(t, err)
	assert.Positive(t, val)

	val, err = utils.CompareSemVer("1.0.0", "1.1.0")
	assert.NoError(t, err)
	assert.Negative(t, val)

	val, err = utils.CompareSemVer("1.1.0", "1.0.0")
	assert.NoError(t, err)
	assert.Positive(t, val)

	val, err = utils.CompareSemVer("1.0.0", "2.0.0")
	assert.NoError(t, err)
	assert.Negative(t, val)

	val, err = utils.CompareSemVer("2.0.0", "1.0.0")
	assert.NoError(t, err)
	assert.Positive(t, val)

	_, err = utils.CompareSemVer("1.0.0", "1.0")
	assert.Error(t, err)

	_, err = utils.CompareSemVer("1.0.a", "1.0.0")
	assert.Error(t, err)

	_, err = utils.CompareSemVer("1.0.0", "1.0.a")
	assert.Error(t, err)
}
