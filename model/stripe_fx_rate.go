package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StripeFXSourceBOCSpotSelling       = "boc_spot_selling"
	StripeFXBaseCurrencyUSD            = "usd"
	StripeFXQuoteCurrencyCNY           = "cny"
	StripeFXBOCSourceURL               = "https://www.boc.cn/sourcedb/whpj/"
	StripeFXRateScale            int64 = 10_000

	stripeFXMinimumRateE4        int64 = 50_000
	stripeFXMaximumRateE4        int64 = 100_000
	stripeFXMaximumSourceAge           = 72 * time.Hour
	stripeFXFutureClockTolerance       = 5 * time.Minute
	stripeFXQuoteVersionPrefix         = "bocfx_"
)

var (
	ErrStripeFXRateUnavailable = errors.New("Stripe FX rate is unavailable")
	ErrStripeFXRateInvalid     = errors.New("Stripe FX rate is invalid")
	ErrStripeFXRateImmutable   = errors.New("Stripe FX daily rate is immutable")
	ErrStripeFXQuoteExpired    = errors.New("Stripe FX quote is expired")
)

// StripeFXDailyRate is the immutable USD/CNY rate selected for one Beijing
// pricing date. RateE4 stores CNY per USD at four decimal places, so the Bank
// of China quote 676.55 CNY per 100 USD is persisted as 67655 (6.7655).
type StripeFXDailyRate struct {
	Id                int64  `json:"id" gorm:"primaryKey"`
	PricingDate       string `json:"pricing_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_stripe_fx_daily_rate,priority:4"`
	Source            string `json:"source" gorm:"type:varchar(32);not null;uniqueIndex:idx_stripe_fx_daily_rate,priority:1"`
	BaseCurrency      string `json:"base_currency" gorm:"type:varchar(8);not null;uniqueIndex:idx_stripe_fx_daily_rate,priority:2"`
	QuoteCurrency     string `json:"quote_currency" gorm:"type:varchar(8);not null;uniqueIndex:idx_stripe_fx_daily_rate,priority:3"`
	RateE4            int64  `json:"rate_e4" gorm:"type:bigint;not null"`
	SourcePublishedAt int64  `json:"source_published_at" gorm:"type:bigint;not null;index"`
	FetchedAt         int64  `json:"fetched_at" gorm:"type:bigint;not null;index"`
	SourceURL         string `json:"source_url" gorm:"type:varchar(255);not null"`
}

// StripeFXRateHead is the stable synchronization row for rate rotation. Both
// daily-rate publication and CNY order creation lock this row before deciding
// which snapshot is current, so a new rate cannot become effective halfway
// through an order transaction.
type StripeFXRateHead struct {
	Id            int64  `json:"id" gorm:"primaryKey"`
	Source        string `json:"source" gorm:"type:varchar(32);not null;uniqueIndex:idx_stripe_fx_rate_head,priority:1"`
	BaseCurrency  string `json:"base_currency" gorm:"type:varchar(8);not null;uniqueIndex:idx_stripe_fx_rate_head,priority:2"`
	QuoteCurrency string `json:"quote_currency" gorm:"type:varchar(8);not null;uniqueIndex:idx_stripe_fx_rate_head,priority:3"`
	CurrentRateId int64  `json:"current_rate_id" gorm:"type:bigint;not null;index"`
}

func StripeFXBeijingLocation() *time.Location {
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func StripeFXPricingDate(now time.Time) string {
	return now.In(StripeFXBeijingLocation()).Format(time.DateOnly)
}

func validateStripeFXDailyRate(rate *StripeFXDailyRate) error {
	if rate == nil ||
		rate.Source != StripeFXSourceBOCSpotSelling ||
		rate.BaseCurrency != StripeFXBaseCurrencyUSD ||
		rate.QuoteCurrency != StripeFXQuoteCurrencyCNY ||
		rate.SourceURL != StripeFXBOCSourceURL ||
		rate.RateE4 < stripeFXMinimumRateE4 ||
		rate.RateE4 > stripeFXMaximumRateE4 ||
		rate.SourcePublishedAt <= 0 ||
		rate.FetchedAt <= 0 {
		return ErrStripeFXRateInvalid
	}
	if _, err := time.Parse(time.DateOnly, rate.PricingDate); err != nil {
		return ErrStripeFXRateInvalid
	}
	if rate.PricingDate != StripeFXPricingDate(time.Unix(rate.FetchedAt, 0)) {
		return ErrStripeFXRateInvalid
	}
	if rate.PricingDate != StripeFXPricingDate(time.Unix(rate.SourcePublishedAt, 0)) {
		return ErrStripeFXRateInvalid
	}
	if rate.SourcePublishedAt > rate.FetchedAt+int64(stripeFXFutureClockTolerance.Seconds()) ||
		rate.FetchedAt-rate.SourcePublishedAt > int64(stripeFXMaximumSourceAge.Seconds()) {
		return ErrStripeFXRateInvalid
	}
	return nil
}

func (rate *StripeFXDailyRate) BeforeCreate(_ *gorm.DB) error {
	return validateStripeFXDailyRate(rate)
}

func (rate *StripeFXDailyRate) BeforeUpdate(_ *gorm.DB) error {
	return ErrStripeFXRateImmutable
}

func (rate *StripeFXDailyRate) RateString() string {
	if rate == nil || rate.RateE4 <= 0 {
		return ""
	}
	return fmt.Sprintf("%d.%04d", rate.RateE4/StripeFXRateScale, rate.RateE4%StripeFXRateScale)
}

func (rate *StripeFXDailyRate) QuoteVersion() string {
	if rate == nil || rate.Id <= 0 || validateStripeFXDailyRate(rate) != nil {
		return ""
	}
	payload := fmt.Sprintf(
		"%d|%s|%s|%s|%s|%d|%d",
		rate.Id,
		rate.PricingDate,
		rate.Source,
		rate.BaseCurrency,
		rate.QuoteCurrency,
		rate.RateE4,
		rate.SourcePublishedAt,
	)
	return fmt.Sprintf("%s%d_%s", stripeFXQuoteVersionPrefix, rate.Id, common.Sha1([]byte(payload))[:16])
}

// CreateStripeFXDailyRate inserts at most one immutable BOC USD/CNY snapshot
// per Beijing pricing date. Concurrent identical inserts return the stored row;
// a conflicting rate for an already-selected date is rejected.
func findStripeFXDailyRateByPricingDateTx(tx *gorm.DB, pricingDate string) (*StripeFXDailyRate, error) {
	if tx == nil {
		return nil, ErrStripeFXRateUnavailable
	}
	var rate StripeFXDailyRate
	err := tx.Where(
		"source = ? AND base_currency = ? AND quote_currency = ? AND pricing_date = ?",
		StripeFXSourceBOCSpotSelling,
		StripeFXBaseCurrencyUSD,
		StripeFXQuoteCurrencyCNY,
		pricingDate,
	).First(&rate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStripeFXRateUnavailable
	}
	if err != nil {
		return nil, err
	}
	if err := validateStripeFXDailyRate(&rate); err != nil {
		return nil, err
	}
	return &rate, nil
}

func lockStripeFXRateHeadTx(tx *gorm.DB, create bool) (*StripeFXRateHead, error) {
	if tx == nil {
		return nil, ErrStripeFXRateUnavailable
	}
	if create {
		head := &StripeFXRateHead{
			Source:        StripeFXSourceBOCSpotSelling,
			BaseCurrency:  StripeFXBaseCurrencyUSD,
			QuoteCurrency: StripeFXQuoteCurrencyCNY,
		}
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "source"},
				{Name: "base_currency"},
				{Name: "quote_currency"},
			},
			DoNothing: true,
		}).Create(head)
		if result.Error != nil {
			return nil, result.Error
		}
	}

	var head StripeFXRateHead
	err := lockForUpdate(tx).Where(
		"source = ? AND base_currency = ? AND quote_currency = ?",
		StripeFXSourceBOCSpotSelling,
		StripeFXBaseCurrencyUSD,
		StripeFXQuoteCurrencyCNY,
	).First(&head).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStripeFXRateUnavailable
	}
	if err != nil {
		return nil, err
	}
	return &head, nil
}

func loadEffectiveStripeFXDailyRateTx(tx *gorm.DB, now time.Time, lockHead bool) (*StripeFXDailyRate, error) {
	if tx == nil || now.IsZero() {
		return nil, ErrStripeFXRateUnavailable
	}
	var head *StripeFXRateHead
	var err error
	if lockHead {
		head, err = lockStripeFXRateHeadTx(tx, false)
	} else {
		var currentHead StripeFXRateHead
		err = tx.Where(
			"source = ? AND base_currency = ? AND quote_currency = ?",
			StripeFXSourceBOCSpotSelling,
			StripeFXBaseCurrencyUSD,
			StripeFXQuoteCurrencyCNY,
		).First(&currentHead).Error
		if err == nil {
			head = &currentHead
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrStripeFXRateUnavailable) {
		return nil, ErrStripeFXRateUnavailable
	}
	if err != nil {
		return nil, err
	}
	if head == nil || head.CurrentRateId <= 0 {
		return nil, ErrStripeFXRateUnavailable
	}

	var rate StripeFXDailyRate
	if err := tx.First(&rate, head.CurrentRateId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStripeFXRateUnavailable
		}
		return nil, err
	}
	if err := validateStripeFXDailyRate(&rate); err != nil {
		return nil, err
	}
	if rate.PricingDate > StripeFXPricingDate(now) {
		return nil, ErrStripeFXRateUnavailable
	}
	nowUnix := now.Unix()
	if rate.SourcePublishedAt > nowUnix+int64(stripeFXFutureClockTolerance.Seconds()) ||
		nowUnix-rate.SourcePublishedAt > int64(stripeFXMaximumSourceAge.Seconds()) {
		return nil, ErrStripeFXRateUnavailable
	}
	return &rate, nil
}

func CreateStripeFXDailyRate(rate *StripeFXDailyRate) (*StripeFXDailyRate, error) {
	if err := validateStripeFXDailyRate(rate); err != nil {
		return nil, err
	}
	if DB == nil {
		return nil, ErrStripeFXRateUnavailable
	}
	var stored *StripeFXDailyRate
	err := DB.Transaction(func(tx *gorm.DB) error {
		head, err := lockStripeFXRateHeadTx(tx, true)
		if err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "source"},
				{Name: "base_currency"},
				{Name: "quote_currency"},
				{Name: "pricing_date"},
			},
			DoNothing: true,
		}).Create(rate)
		if result.Error != nil {
			return result.Error
		}
		// MySQL can report a no-op duplicate as one affected row when the DSN
		// enables clientFoundRows. Always reload by the immutable business key;
		// neither RowsAffected nor LastInsertId is authoritative for a replay.
		stored, err = findStripeFXDailyRateByPricingDateTx(tx, rate.PricingDate)
		if err != nil {
			return err
		}
		if stored.Source != rate.Source ||
			stored.BaseCurrency != rate.BaseCurrency ||
			stored.QuoteCurrency != rate.QuoteCurrency ||
			stored.RateE4 != rate.RateE4 ||
			stored.SourcePublishedAt != rate.SourcePublishedAt ||
			stored.SourceURL != rate.SourceURL {
			return ErrStripeFXRateImmutable
		}

		if head.CurrentRateId > 0 {
			var current StripeFXDailyRate
			if err := tx.First(&current, head.CurrentRateId).Error; err != nil {
				return err
			}
			if err := validateStripeFXDailyRate(&current); err != nil {
				return err
			}
			if current.PricingDate > stored.PricingDate {
				return nil
			}
		}
		if head.CurrentRateId == stored.Id {
			return nil
		}
		return tx.Model(&StripeFXRateHead{}).
			Where("id = ?", head.Id).
			Update("current_rate_id", stored.Id).Error
	})
	if err != nil {
		return nil, err
	}
	return stored, nil
}

func FindStripeFXDailyRateByPricingDate(pricingDate string) (*StripeFXDailyRate, error) {
	if _, err := time.Parse(time.DateOnly, pricingDate); err != nil {
		return nil, ErrStripeFXRateInvalid
	}
	if DB == nil {
		return nil, ErrStripeFXRateUnavailable
	}
	return findStripeFXDailyRateByPricingDateTx(DB, pricingDate)
}

func FindEffectiveStripeFXDailyRate(now time.Time) (*StripeFXDailyRate, error) {
	if now.IsZero() {
		return nil, ErrStripeFXRateInvalid
	}
	if DB == nil {
		return nil, ErrStripeFXRateUnavailable
	}
	return loadEffectiveStripeFXDailyRateTx(DB, now, false)
}

// FindStripeFXDailyRateByQuoteVersion resolves a quote only when it still
// identifies the current effective daily snapshot. This prevents a client from
// selecting an older, more favorable rate after a daily rotation.
func FindStripeFXDailyRateByQuoteVersion(version string, now time.Time) (*StripeFXDailyRate, error) {
	if now.IsZero() || !strings.HasPrefix(version, stripeFXQuoteVersionPrefix) {
		return nil, ErrStripeFXQuoteExpired
	}
	if DB == nil {
		return nil, ErrStripeFXRateUnavailable
	}
	identifierAndDigest := strings.TrimPrefix(version, stripeFXQuoteVersionPrefix)
	parts := strings.Split(identifierAndDigest, "_")
	if len(parts) != 2 || len(parts[1]) != 16 {
		return nil, ErrStripeFXQuoteExpired
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrStripeFXQuoteExpired
	}

	var quoted StripeFXDailyRate
	if err := DB.First(&quoted, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStripeFXQuoteExpired
		}
		return nil, err
	}
	if quoted.QuoteVersion() != version {
		return nil, ErrStripeFXQuoteExpired
	}
	effective, err := FindEffectiveStripeFXDailyRate(now)
	if err != nil {
		if errors.Is(err, ErrStripeFXRateUnavailable) {
			return nil, ErrStripeFXQuoteExpired
		}
		return nil, err
	}
	if effective.Id != quoted.Id || effective.QuoteVersion() != version {
		return nil, ErrStripeFXQuoteExpired
	}
	return &quoted, nil
}
