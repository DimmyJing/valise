package utils_test

import (
	"testing"

	"github.com/DimmyJing/valise/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemver(t *testing.T) {
	t.Parallel()

	val, err := utils.CompareSemVer("1.0.0", "1.0.1")
	require.NoError(t, err)
	assert.Negative(t, val)

	val, err = utils.CompareSemVer("1.0.0", "1.0.0")
	require.NoError(t, err)
	assert.Zero(t, val)

	val, err = utils.CompareSemVer("1.0.1", "1.0.0")
	require.NoError(t, err)
	assert.Positive(t, val)

	val, err = utils.CompareSemVer("1.0.0", "1.1.0")
	require.NoError(t, err)
	assert.Negative(t, val)

	val, err = utils.CompareSemVer("1.1.0", "1.0.0")
	require.NoError(t, err)
	assert.Positive(t, val)

	val, err = utils.CompareSemVer("1.0.0", "2.0.0")
	require.NoError(t, err)
	assert.Negative(t, val)

	val, err = utils.CompareSemVer("2.0.0", "1.0.0")
	require.NoError(t, err)
	assert.Positive(t, val)

	_, err = utils.CompareSemVer("1.0.0", "1.0")
	require.Error(t, err)

	_, err = utils.CompareSemVer("1.0.a", "1.0.0")
	require.Error(t, err)

	_, err = utils.CompareSemVer("1.0.0", "1.0.a")
	require.Error(t, err)
}
