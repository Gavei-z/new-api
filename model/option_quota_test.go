package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeQuotaGrantOptionValueRequiresBoundedInteger(t *testing.T) {
	for _, key := range []string{
		"QuotaForNewUser",
		"QuotaForInviter",
		"QuotaForInvitee",
	} {
		value, err := NormalizeQuotaGrantOptionValue(key, " 500000 ")
		require.NoError(t, err)
		assert.Equal(t, "500000", value)

		for _, invalid := range []string{
			"-1",
			"500000.5",
			"not-a-number",
			strconv.FormatInt(int64(common.MaxQuota)+1, 10),
		} {
			_, err := NormalizeQuotaGrantOptionValue(key, invalid)
			require.Error(t, err)
		}
	}

	value, err := NormalizeQuotaGrantOptionValue("UnrelatedOption", "raw")
	require.NoError(t, err)
	assert.Equal(t, "raw", value)
}

func TestUpdateOptionsBulkRejectsInvalidQuotaBeforeWritingAnything(t *testing.T) {
	db := useFrontendOptionMigrationDB(t)

	err := UpdateOptionsBulk(map[string]string{
		"QuotaForNewUser": "500000.5",
		"UnrelatedOption": "must-not-be-written",
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, db.Model(&Option{}).Count(&count).Error)
	assert.Zero(t, count)
}
