package model

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stripeFXTestTime(year int, month time.Month, day int, hour int, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, StripeFXBeijingLocation())
}

func stripeFXTestRate(now time.Time, rateE4 int64) *StripeFXDailyRate {
	return &StripeFXDailyRate{
		PricingDate:       StripeFXPricingDate(now),
		Source:            StripeFXSourceBOCSpotSelling,
		BaseCurrency:      StripeFXBaseCurrencyUSD,
		QuoteCurrency:     StripeFXQuoteCurrencyCNY,
		RateE4:            rateE4,
		SourcePublishedAt: now.Add(-90 * time.Minute).Unix(),
		FetchedAt:         now.Unix(),
		SourceURL:         StripeFXBOCSourceURL,
	}
}

func setupStripeFXTestDB(t *testing.T) {
	t.Helper()
	db := setupTeamTestDB(t)
	require.NoError(t, db.AutoMigrate(&StripeFXDailyRate{}, &StripeFXRateHead{}))
}

func TestStripeFXDailyRateAutoMigrateSQLite(t *testing.T) {
	setupStripeFXTestDB(t)

	for _, column := range []string{
		"pricing_date",
		"source",
		"base_currency",
		"quote_currency",
		"rate_e4",
		"source_published_at",
		"fetched_at",
		"source_url",
	} {
		assert.True(t, DB.Migrator().HasColumn(&StripeFXDailyRate{}, column), column)
	}
	assert.True(t, DB.Migrator().HasIndex(&StripeFXDailyRate{}, "idx_stripe_fx_daily_rate"))
	assert.True(t, DB.Migrator().HasColumn(&StripeFXRateHead{}, "current_rate_id"))
	assert.True(t, DB.Migrator().HasIndex(&StripeFXRateHead{}, "idx_stripe_fx_rate_head"))
}

func TestStripeFXRateHeadTracksNewestSnapshotWithoutRegressing(t *testing.T) {
	setupStripeFXTestDB(t)
	currentNow := stripeFXTestTime(2026, time.August, 9, 11, 5)
	current, err := CreateStripeFXDailyRate(stripeFXTestRate(currentNow, 67_655))
	require.NoError(t, err)

	previousNow := stripeFXTestTime(2026, time.August, 8, 11, 5)
	previous, err := CreateStripeFXDailyRate(stripeFXTestRate(previousNow, 67_600))
	require.NoError(t, err)
	assert.NotEqual(t, current.Id, previous.Id)

	var head StripeFXRateHead
	require.NoError(t, DB.First(&head).Error)
	assert.Equal(t, current.Id, head.CurrentRateId)
	effective, err := FindEffectiveStripeFXDailyRate(currentNow)
	require.NoError(t, err)
	assert.Equal(t, current.Id, effective.Id)
}

func TestStripeFXDailyRateStoresExactBOCRateAndIsImmutablePerDay(t *testing.T) {
	setupStripeFXTestDB(t)
	now := stripeFXTestTime(2026, time.August, 9, 13, 45)

	stored, err := CreateStripeFXDailyRate(stripeFXTestRate(now, 67_655))
	require.NoError(t, err)
	assert.EqualValues(t, 67_655, stored.RateE4)
	assert.Equal(t, "6.7655", stored.RateString())
	assert.NotEmpty(t, stored.QuoteVersion())

	replay := stripeFXTestRate(now.Add(10*time.Minute), 67_655)
	replay.SourcePublishedAt = stored.SourcePublishedAt
	replayed, err := CreateStripeFXDailyRate(replay)
	require.NoError(t, err)
	assert.Equal(t, stored.Id, replayed.Id)
	assert.Equal(t, stored.FetchedAt, replayed.FetchedAt)

	conflict := stripeFXTestRate(now.Add(20*time.Minute), 67_700)
	_, err = CreateStripeFXDailyRate(conflict)
	require.ErrorIs(t, err, ErrStripeFXRateImmutable)

	err = DB.Model(stored).Update("rate_e4", int64(67_700)).Error
	require.ErrorIs(t, err, ErrStripeFXRateImmutable)
}

func TestStripeFXDailyRateRejectsInvalidSourceAndStalePublication(t *testing.T) {
	setupStripeFXTestDB(t)
	now := stripeFXTestTime(2026, time.August, 9, 13, 45)

	invalidSource := stripeFXTestRate(now, 67_655)
	invalidSource.SourceURL = "https://example.com/rate"
	_, err := CreateStripeFXDailyRate(invalidSource)
	require.ErrorIs(t, err, ErrStripeFXRateInvalid)

	stale := stripeFXTestRate(now, 67_655)
	stale.SourcePublishedAt = now.Add(-73 * time.Hour).Unix()
	_, err = CreateStripeFXDailyRate(stale)
	require.ErrorIs(t, err, ErrStripeFXRateInvalid)

	wrongSourceDate := stripeFXTestRate(now, 67_655)
	wrongSourceDate.SourcePublishedAt = now.AddDate(0, 0, -1).Unix()
	_, err = CreateStripeFXDailyRate(wrongSourceDate)
	require.ErrorIs(t, err, ErrStripeFXRateInvalid)
}

func TestStripeFXQuoteVersionMustIdentifyCurrentEffectiveRate(t *testing.T) {
	setupStripeFXTestDB(t)
	previousNow := stripeFXTestTime(2026, time.August, 8, 12, 0)
	previous, err := CreateStripeFXDailyRate(stripeFXTestRate(previousNow, 67_600))
	require.NoError(t, err)

	beforeRotation := stripeFXTestTime(2026, time.August, 9, 10, 30)
	effective, err := FindEffectiveStripeFXDailyRate(beforeRotation)
	require.NoError(t, err)
	assert.Equal(t, previous.Id, effective.Id)
	resolved, err := FindStripeFXDailyRateByQuoteVersion(previous.QuoteVersion(), beforeRotation)
	require.NoError(t, err)
	assert.Equal(t, previous.Id, resolved.Id)

	currentNow := stripeFXTestTime(2026, time.August, 9, 11, 5)
	current, err := CreateStripeFXDailyRate(stripeFXTestRate(currentNow, 67_655))
	require.NoError(t, err)

	_, err = FindStripeFXDailyRateByQuoteVersion(previous.QuoteVersion(), currentNow)
	require.ErrorIs(t, err, ErrStripeFXQuoteExpired)
	resolved, err = FindStripeFXDailyRateByQuoteVersion(current.QuoteVersion(), currentNow)
	require.NoError(t, err)
	assert.Equal(t, current.Id, resolved.Id)

	_, err = FindStripeFXDailyRateByQuoteVersion("bocfx_999_forged000000000", currentNow)
	assert.True(t, errors.Is(err, ErrStripeFXQuoteExpired))
}

func TestStripeFXEffectiveRateFailsClosedWhenSourceBecomesStale(t *testing.T) {
	setupStripeFXTestDB(t)
	fetchedAt := stripeFXTestTime(2026, time.August, 6, 12, 0)
	stored, err := CreateStripeFXDailyRate(stripeFXTestRate(fetchedAt, 67_600))
	require.NoError(t, err)

	_, err = FindEffectiveStripeFXDailyRate(time.Unix(stored.SourcePublishedAt, 0).Add(72*time.Hour + time.Second))
	require.ErrorIs(t, err, ErrStripeFXRateUnavailable)
}
