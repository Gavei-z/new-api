package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	PrepaidTargetUser = "user"
	PrepaidTargetTeam = "team"

	PrepaidTransactionCredit      = "credit"
	PrepaidTransactionReversal    = "reversal"
	PrepaidTransactionRestoration = "restoration"
	PrepaidTransactionRelease     = "release"
)

var (
	ErrPrepaidTargetInvalid       = errors.New("invalid prepaid balance target")
	ErrPrepaidQuotaInvalid        = errors.New("invalid prepaid quota")
	ErrPrepaidBalanceOutOfRange   = errors.New("prepaid balance is out of range")
	ErrPrepaidIdempotencyConflict = errors.New("prepaid idempotency key conflicts with an existing transaction")
)

// PrepaidReserve stores prepaid credit that cannot safely be placed in the
// legacy 32-bit active quota columns. The active quota remains the hot billing
// bucket; this bigint reserve is released into it only when a request needs
// more active quota.
type PrepaidReserve struct {
	Id         int64  `json:"id"`
	TargetType string `json:"target_type" gorm:"type:varchar(16);not null;uniqueIndex:idx_prepaid_reserve_target"`
	TargetId   int    `json:"target_id" gorm:"not null;uniqueIndex:idx_prepaid_reserve_target"`
	Quota      int64  `json:"quota" gorm:"type:bigint;not null;default:0"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

// PrepaidReserveTransaction is an immutable audit trail for credits,
// reversals, dispute restorations, and reserve-to-active releases.
type PrepaidReserveTransaction struct {
	Id                  int64  `json:"id"`
	TargetType          string `json:"target_type" gorm:"type:varchar(16);not null;index"`
	TargetId            int    `json:"target_id" gorm:"not null;index"`
	UserId              int    `json:"user_id" gorm:"index"`
	Type                string `json:"type" gorm:"type:varchar(24);not null;index"`
	RequestedQuota      int64  `json:"requested_quota" gorm:"type:bigint;not null"`
	QuotaDelta          int64  `json:"quota_delta" gorm:"type:bigint;not null"`
	ActiveQuotaDelta    int64  `json:"active_quota_delta" gorm:"type:bigint;not null"`
	ReserveQuotaDelta   int64  `json:"reserve_quota_delta" gorm:"type:bigint;not null"`
	ActiveBalanceAfter  int64  `json:"active_balance_after" gorm:"type:bigint;not null"`
	ReserveBalanceAfter int64  `json:"reserve_balance_after" gorm:"type:bigint;not null"`
	TotalBalanceAfter   int64  `json:"total_balance_after" gorm:"type:bigint;not null"`
	ShortfallQuota      int64  `json:"shortfall_quota" gorm:"type:bigint;not null;default:0"`
	IdempotencyKey      string `json:"idempotency_key" gorm:"type:varchar(191);not null;uniqueIndex"`
	Note                string `json:"note" gorm:"type:varchar(255)"`
	CreatedAt           int64  `json:"created_at" gorm:"autoCreateTime;index"`
}

// PrepaidBalance reports both the hot active bucket and the bigint reserve.
// TotalQuota is the user-visible spendable balance.
type PrepaidBalance struct {
	ActiveQuota  int64 `json:"active_quota"`
	ReserveQuota int64 `json:"reserve_quota"`
	TotalQuota   int64 `json:"total_quota"`
}

// TeamBalanceTransactionView normalizes the legacy active-quota ledger and
// the Stripe prepaid ledger for the team console.
type TeamBalanceTransactionView struct {
	Id                  string `json:"id"`
	TeamId              int    `json:"team_id"`
	UserId              int    `json:"user_id"`
	ActorUserId         int    `json:"actor_user_id"`
	Type                string `json:"type"`
	Source              string `json:"source"`
	QuotaDelta          int64  `json:"quota_delta"`
	UsedQuotaDelta      int64  `json:"used_quota_delta"`
	ActiveQuotaDelta    int64  `json:"active_quota_delta"`
	ReserveQuotaDelta   int64  `json:"reserve_quota_delta"`
	BalanceAfter        int64  `json:"balance_after"`
	ActiveBalanceAfter  int64  `json:"active_balance_after"`
	ReserveBalanceAfter int64  `json:"reserve_balance_after"`
	UsedQuotaAfter      int64  `json:"used_quota_after"`
	Note                string `json:"note"`
	CreatedAt           int64  `json:"created_at"`
}

// Keep the hot bucket at half of the int32 database limit. This leaves room
// for in-flight request refunds while allowing a single normal request to be
// pre-consumed without touching the reserve on every request.
const prepaidActiveRefillTarget = int64(common.MaxQuota / 2)

func validatePrepaidTarget(targetType string, targetId int) error {
	if targetId <= 0 {
		return ErrPrepaidTargetInvalid
	}
	switch targetType {
	case PrepaidTargetUser, PrepaidTargetTeam:
		return nil
	default:
		return ErrPrepaidTargetInvalid
	}
}

func validatePrepaidIdempotencyKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 191 {
		return errors.New("invalid prepaid idempotency key")
	}
	return nil
}

func prepaidNote(note string) string {
	runes := []rune(strings.TrimSpace(note))
	if len(runes) > 255 {
		runes = runes[:255]
	}
	return string(runes)
}

func loadPrepaidActiveQuotaTx(tx *gorm.DB, targetType string, targetId int, lock bool) (int64, error) {
	query := tx
	if lock {
		query = lockForUpdate(query)
	}
	switch targetType {
	case PrepaidTargetUser:
		var user User
		if err := query.Select("id", "quota").Where("id = ?", targetId).First(&user).Error; err != nil {
			return 0, err
		}
		if user.Quota < 0 || user.Quota > common.MaxQuota {
			return 0, ErrPrepaidBalanceOutOfRange
		}
		return int64(user.Quota), nil
	case PrepaidTargetTeam:
		var team Team
		if err := query.Select("id", "quota").Where("id = ?", targetId).First(&team).Error; err != nil {
			return 0, err
		}
		if team.Quota < 0 || team.Quota > common.MaxQuota {
			return 0, ErrPrepaidBalanceOutOfRange
		}
		return int64(team.Quota), nil
	default:
		return 0, ErrPrepaidTargetInvalid
	}
}

func updatePrepaidActiveQuotaTx(tx *gorm.DB, targetType string, targetId int, quota int64) error {
	if quota < 0 || quota > int64(common.MaxQuota) {
		return ErrPrepaidBalanceOutOfRange
	}
	switch targetType {
	case PrepaidTargetUser:
		return tx.Model(&User{}).Where("id = ?", targetId).Update("quota", int(quota)).Error
	case PrepaidTargetTeam:
		return tx.Model(&Team{}).Where("id = ?", targetId).Update("quota", int(quota)).Error
	default:
		return ErrPrepaidTargetInvalid
	}
}

func loadOrCreatePrepaidReserveTx(tx *gorm.DB, targetType string, targetId int) (*PrepaidReserve, error) {
	var reserve PrepaidReserve
	err := lockForUpdate(tx).
		Where("target_type = ? AND target_id = ?", targetType, targetId).
		First(&reserve).Error
	if err == nil {
		if reserve.Quota < 0 {
			return nil, ErrPrepaidBalanceOutOfRange
		}
		return &reserve, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	reserve = PrepaidReserve{
		TargetType: targetType,
		TargetId:   targetId,
		Quota:      0,
	}
	if err := tx.Create(&reserve).Error; err != nil {
		return nil, err
	}
	return &reserve, nil
}

func savePrepaidReserveTx(tx *gorm.DB, reserve *PrepaidReserve, quota int64) error {
	if reserve == nil || quota < 0 {
		return ErrPrepaidBalanceOutOfRange
	}
	reserve.Quota = quota
	return tx.Model(&PrepaidReserve{}).
		Where("id = ?", reserve.Id).
		Update("quota", quota).Error
}

func findPrepaidTransactionTx(tx *gorm.DB, key string) (*PrepaidReserveTransaction, error) {
	var transaction PrepaidReserveTransaction
	err := tx.Where("idempotency_key = ?", key).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func findPrepaidTransactionForUpdateTx(tx *gorm.DB, key string) (*PrepaidReserveTransaction, error) {
	var transaction PrepaidReserveTransaction
	err := lockForUpdate(tx).
		Where("idempotency_key = ?", key).
		First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func validateExistingPrepaidTransaction(
	transaction *PrepaidReserveTransaction,
	targetType string,
	targetId int,
	userId int,
	transactionType string,
	requestedQuota int64,
) error {
	if transaction == nil ||
		transaction.TargetType != targetType ||
		transaction.TargetId != targetId ||
		transaction.UserId != userId ||
		transaction.Type != transactionType ||
		transaction.RequestedQuota != requestedQuota {
		return ErrPrepaidIdempotencyConflict
	}
	return nil
}

func createPrepaidTransactionTx(
	tx *gorm.DB,
	targetType string,
	targetId int,
	userId int,
	transactionType string,
	requestedQuota int64,
	quotaDelta int64,
	activeDelta int64,
	reserveDelta int64,
	activeAfter int64,
	reserveAfter int64,
	shortfall int64,
	idempotencyKey string,
	note string,
) error {
	if activeAfter < 0 || reserveAfter < 0 || activeAfter > int64(common.MaxQuota) {
		return ErrPrepaidBalanceOutOfRange
	}
	if activeAfter > int64(^uint64(0)>>1)-reserveAfter {
		return ErrPrepaidBalanceOutOfRange
	}
	return tx.Create(&PrepaidReserveTransaction{
		TargetType:          targetType,
		TargetId:            targetId,
		UserId:              userId,
		Type:                transactionType,
		RequestedQuota:      requestedQuota,
		QuotaDelta:          quotaDelta,
		ActiveQuotaDelta:    activeDelta,
		ReserveQuotaDelta:   reserveDelta,
		ActiveBalanceAfter:  activeAfter,
		ReserveBalanceAfter: reserveAfter,
		TotalBalanceAfter:   activeAfter + reserveAfter,
		ShortfallQuota:      shortfall,
		IdempotencyKey:      strings.TrimSpace(idempotencyKey),
		Note:                prepaidNote(note),
	}).Error
}

func applyPrepaidCreditTx(
	tx *gorm.DB,
	targetType string,
	targetId int,
	userId int,
	quota int64,
	idempotencyKey string,
	note string,
	transactionType string,
) error {
	if tx == nil {
		return errors.New("database transaction is required")
	}
	if err := validatePrepaidTarget(targetType, targetId); err != nil {
		return err
	}
	if quota <= 0 {
		return ErrPrepaidQuotaInvalid
	}
	if err := validatePrepaidIdempotencyKey(idempotencyKey); err != nil {
		return err
	}
	if existing, err := findPrepaidTransactionTx(tx, strings.TrimSpace(idempotencyKey)); err == nil {
		return validateExistingPrepaidTransaction(
			existing, targetType, targetId, userId, transactionType, quota,
		)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	activeBefore, err := loadPrepaidActiveQuotaTx(tx, targetType, targetId, true)
	if err != nil {
		return err
	}
	// A concurrent delivery can insert the idempotency record while this
	// transaction waits for the target row lock. Recheck after acquiring it so
	// the retry succeeds cleanly instead of surfacing a unique-key error.
	if existing, findErr := findPrepaidTransactionForUpdateTx(tx, strings.TrimSpace(idempotencyKey)); findErr == nil {
		return validateExistingPrepaidTransaction(
			existing, targetType, targetId, userId, transactionType, quota,
		)
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return findErr
	}
	reserve, err := loadOrCreatePrepaidReserveTx(tx, targetType, targetId)
	if err != nil {
		return err
	}
	if quota > int64(^uint64(0)>>1)-reserve.Quota-activeBefore {
		return ErrPrepaidBalanceOutOfRange
	}

	activeDelta := int64(0)
	if activeBefore < prepaidActiveRefillTarget {
		activeDelta = prepaidActiveRefillTarget - activeBefore
		if activeDelta > quota {
			activeDelta = quota
		}
	}
	reserveDelta := quota - activeDelta
	activeAfter := activeBefore + activeDelta
	reserveAfter := reserve.Quota + reserveDelta

	if err := updatePrepaidActiveQuotaTx(tx, targetType, targetId, activeAfter); err != nil {
		return err
	}
	if err := savePrepaidReserveTx(tx, reserve, reserveAfter); err != nil {
		return err
	}
	return createPrepaidTransactionTx(
		tx,
		targetType,
		targetId,
		userId,
		transactionType,
		quota,
		quota,
		activeDelta,
		reserveDelta,
		activeAfter,
		reserveAfter,
		0,
		idempotencyKey,
		note,
	)
}

// ApplyPrepaidCreditTx atomically credits a Stripe payment to a user or team.
// Quota beyond the safe 32-bit active bucket is retained in the bigint reserve.
func ApplyPrepaidCreditTx(
	tx *gorm.DB,
	targetType string,
	targetId int,
	userId int,
	quota int64,
	idempotencyKey string,
	note string,
) error {
	return applyPrepaidCreditTx(
		tx, targetType, targetId, userId, quota, idempotencyKey, note, PrepaidTransactionCredit,
	)
}

// ApplyPrepaidRestorationTx restores credit after a dispute is won or
// otherwise reversed by Stripe.
func ApplyPrepaidRestorationTx(
	tx *gorm.DB,
	targetType string,
	targetId int,
	userId int,
	quota int64,
	idempotencyKey string,
	note string,
) error {
	return applyPrepaidCreditTx(
		tx, targetType, targetId, userId, quota, idempotencyKey, note, PrepaidTransactionRestoration,
	)
}

// ApplyPrepaidReversalTx removes unused balance after a refund or dispute.
// It drains the reserve first and then the active bucket. A non-zero shortfall
// means some credit was already spent and requires reconciliation.
func ApplyPrepaidReversalTx(
	tx *gorm.DB,
	targetType string,
	targetId int,
	userId int,
	quota int64,
	idempotencyKey string,
	note string,
) (applied int64, shortfall int64, err error) {
	if tx == nil {
		return 0, 0, errors.New("database transaction is required")
	}
	if err = validatePrepaidTarget(targetType, targetId); err != nil {
		return 0, 0, err
	}
	if quota <= 0 {
		return 0, 0, ErrPrepaidQuotaInvalid
	}
	if err = validatePrepaidIdempotencyKey(idempotencyKey); err != nil {
		return 0, 0, err
	}
	key := strings.TrimSpace(idempotencyKey)
	if existing, findErr := findPrepaidTransactionTx(tx, key); findErr == nil {
		if err = validateExistingPrepaidTransaction(
			existing, targetType, targetId, userId, PrepaidTransactionReversal, quota,
		); err != nil {
			return 0, 0, err
		}
		return -existing.QuotaDelta, existing.ShortfallQuota, nil
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return 0, 0, findErr
	}

	activeBefore, err := loadPrepaidActiveQuotaTx(tx, targetType, targetId, true)
	if err != nil {
		return 0, 0, err
	}
	if existing, findErr := findPrepaidTransactionForUpdateTx(tx, key); findErr == nil {
		if err = validateExistingPrepaidTransaction(
			existing, targetType, targetId, userId, PrepaidTransactionReversal, quota,
		); err != nil {
			return 0, 0, err
		}
		return -existing.QuotaDelta, existing.ShortfallQuota, nil
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return 0, 0, findErr
	}
	reserve, err := loadOrCreatePrepaidReserveTx(tx, targetType, targetId)
	if err != nil {
		return 0, 0, err
	}

	reserveDeduction := quota
	if reserveDeduction > reserve.Quota {
		reserveDeduction = reserve.Quota
	}
	remaining := quota - reserveDeduction
	activeDeduction := remaining
	if activeDeduction > activeBefore {
		activeDeduction = activeBefore
	}
	applied = reserveDeduction + activeDeduction
	shortfall = quota - applied
	activeAfter := activeBefore - activeDeduction
	reserveAfter := reserve.Quota - reserveDeduction

	if err = updatePrepaidActiveQuotaTx(tx, targetType, targetId, activeAfter); err != nil {
		return 0, 0, err
	}
	if err = savePrepaidReserveTx(tx, reserve, reserveAfter); err != nil {
		return 0, 0, err
	}
	err = createPrepaidTransactionTx(
		tx,
		targetType,
		targetId,
		userId,
		PrepaidTransactionReversal,
		quota,
		-applied,
		-activeDeduction,
		-reserveDeduction,
		activeAfter,
		reserveAfter,
		shortfall,
		key,
		note,
	)
	if err != nil {
		return 0, 0, err
	}
	return applied, shortfall, nil
}

// GetPrepaidBalance returns the total spendable balance without moving funds
// between buckets.
func GetPrepaidBalance(targetType string, targetId int) (PrepaidBalance, error) {
	var balance PrepaidBalance
	if err := validatePrepaidTarget(targetType, targetId); err != nil {
		return balance, err
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		active, err := loadPrepaidActiveQuotaTx(tx, targetType, targetId, true)
		if err != nil {
			return err
		}
		balance.ActiveQuota = active
		var reserve PrepaidReserve
		err = tx.Where("target_type = ? AND target_id = ?", targetType, targetId).First(&reserve).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if reserve.Quota < 0 || active > int64(^uint64(0)>>1)-reserve.Quota {
				return ErrPrepaidBalanceOutOfRange
			}
			balance.ReserveQuota = reserve.Quota
		}
		balance.TotalQuota = balance.ActiveQuota + balance.ReserveQuota
		return nil
	})
	return balance, err
}

// PopulateTeamPrepaidBalance adds user-visible reserve and total fields to a
// team response while leaving the legacy active quota unchanged.
func PopulateTeamPrepaidBalance(team *Team) error {
	if team == nil || team.Id <= 0 {
		return ErrPrepaidTargetInvalid
	}
	balance, err := GetPrepaidBalance(PrepaidTargetTeam, team.Id)
	if err != nil {
		return err
	}
	team.ReserveQuota = balance.ReserveQuota
	team.TotalQuota = balance.TotalQuota
	return nil
}

// ListTeamBalanceTransactions merges team usage adjustments with externally
// funded Stripe balance events. Internal reserve releases are intentionally
// omitted because they do not change total spendable balance.
func ListTeamBalanceTransactions(
	teamId int,
	offset int,
	limit int,
) ([]TeamBalanceTransactionView, int64, error) {
	if teamId <= 0 {
		return nil, 0, ErrPrepaidTargetInvalid
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var teamTotal int64
	if err := DB.Model(&TeamQuotaTransaction{}).
		Where("team_id = ?", teamId).
		Count(&teamTotal).Error; err != nil {
		return nil, 0, err
	}
	var prepaidTotal int64
	if err := DB.Model(&PrepaidReserveTransaction{}).
		Where(
			"target_type = ? AND target_id = ? AND type <> ?",
			PrepaidTargetTeam,
			teamId,
			PrepaidTransactionRelease,
		).
		Count(&prepaidTotal).Error; err != nil {
		return nil, 0, err
	}

	fetchLimit := offset + limit
	var teamRows []TeamQuotaTransaction
	if err := DB.Where("team_id = ?", teamId).
		Order("created_at desc, id desc").
		Limit(fetchLimit).
		Find(&teamRows).Error; err != nil {
		return nil, 0, err
	}
	var prepaidRows []PrepaidReserveTransaction
	if err := DB.Where(
		"target_type = ? AND target_id = ? AND type <> ?",
		PrepaidTargetTeam,
		teamId,
		PrepaidTransactionRelease,
	).
		Order("created_at desc, id desc").
		Limit(fetchLimit).
		Find(&prepaidRows).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]TeamBalanceTransactionView, 0, len(teamRows)+len(prepaidRows))
	for _, transaction := range teamRows {
		totalBalanceAfter := transaction.TotalBalanceAfter
		if totalBalanceAfter == 0 &&
			(transaction.BalanceAfter != 0 || transaction.ReserveBalanceAfter != 0) {
			// Rows created before the bigint snapshot columns were introduced
			// have zero-valued migrated fields. Preserve their legacy active
			// balance while preferring the exact total for every new row.
			totalBalanceAfter = int64(transaction.BalanceAfter) + transaction.ReserveBalanceAfter
		}
		rows = append(rows, TeamBalanceTransactionView{
			Id:                  fmt.Sprintf("team:%d", transaction.Id),
			TeamId:              transaction.TeamId,
			UserId:              transaction.UserId,
			ActorUserId:         transaction.ActorUserId,
			Type:                transaction.Type,
			Source:              "team",
			QuotaDelta:          int64(transaction.QuotaDelta),
			UsedQuotaDelta:      transaction.UsedQuotaDelta,
			ActiveQuotaDelta:    int64(transaction.QuotaDelta),
			BalanceAfter:        totalBalanceAfter,
			ActiveBalanceAfter:  int64(transaction.BalanceAfter),
			ReserveBalanceAfter: transaction.ReserveBalanceAfter,
			UsedQuotaAfter:      transaction.UsedQuotaAfter,
			Note:                transaction.Note,
			CreatedAt:           transaction.CreatedAt,
		})
	}
	for _, transaction := range prepaidRows {
		transactionType := "stripe_" + transaction.Type
		if transaction.Type == PrepaidTransactionCredit {
			transactionType = "stripe_recharge"
		}
		rows = append(rows, TeamBalanceTransactionView{
			Id:                  fmt.Sprintf("prepaid:%d", transaction.Id),
			TeamId:              transaction.TargetId,
			UserId:              transaction.UserId,
			ActorUserId:         transaction.UserId,
			Type:                transactionType,
			Source:              "stripe",
			QuotaDelta:          transaction.QuotaDelta,
			ActiveQuotaDelta:    transaction.ActiveQuotaDelta,
			ReserveQuotaDelta:   transaction.ReserveQuotaDelta,
			BalanceAfter:        transaction.TotalBalanceAfter,
			ActiveBalanceAfter:  transaction.ActiveBalanceAfter,
			ReserveBalanceAfter: transaction.ReserveBalanceAfter,
			Note:                transaction.Note,
			CreatedAt:           transaction.CreatedAt,
		})
	}
	sort.SliceStable(rows, func(i int, j int) bool {
		if rows[i].CreatedAt == rows[j].CreatedAt {
			return rows[i].Id > rows[j].Id
		}
		return rows[i].CreatedAt > rows[j].CreatedAt
	})
	if offset >= len(rows) {
		return []TeamBalanceTransactionView{}, teamTotal + prepaidTotal, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], teamTotal + prepaidTotal, nil
}

func preparePrepaidSpendWithActiveTx(
	tx *gorm.DB,
	targetType string,
	targetId int,
	userId int,
	activeBefore int64,
	required int,
) (PrepaidBalance, error) {
	result := PrepaidBalance{
		ActiveQuota: activeBefore,
		TotalQuota:  activeBefore,
	}
	if activeBefore >= int64(required) {
		return result, nil
	}
	reserve, err := loadOrCreatePrepaidReserveTx(tx, targetType, targetId)
	if err != nil {
		return PrepaidBalance{}, err
	}
	if reserve.Quota < 0 || activeBefore > int64(^uint64(0)>>1)-reserve.Quota {
		return PrepaidBalance{}, ErrPrepaidBalanceOutOfRange
	}
	result.ReserveQuota = reserve.Quota
	result.TotalQuota = activeBefore + reserve.Quota
	if reserve.Quota == 0 {
		return result, nil
	}

	target := prepaidActiveRefillTarget
	if int64(required) > target {
		target = int64(required)
	}
	if target > int64(common.MaxQuota) {
		target = int64(common.MaxQuota)
	}
	release := target - activeBefore
	if release > reserve.Quota {
		release = reserve.Quota
	}
	if release <= 0 {
		return result, nil
	}
	activeAfter := activeBefore + release
	reserveAfter := reserve.Quota - release
	if err := updatePrepaidActiveQuotaTx(tx, targetType, targetId, activeAfter); err != nil {
		return PrepaidBalance{}, err
	}
	if err := savePrepaidReserveTx(tx, reserve, reserveAfter); err != nil {
		return PrepaidBalance{}, err
	}
	key := fmt.Sprintf(
		"prepaid:release:%s:%d:%s",
		targetType,
		targetId,
		common.GetRandomString(24),
	)
	if err := createPrepaidTransactionTx(
		tx,
		targetType,
		targetId,
		userId,
		PrepaidTransactionRelease,
		release,
		0,
		release,
		-release,
		activeAfter,
		reserveAfter,
		0,
		key,
		"Released prepaid reserve into active quota",
	); err != nil {
		return PrepaidBalance{}, err
	}
	result.ActiveQuota = activeAfter
	result.ReserveQuota = reserveAfter
	result.TotalQuota = activeAfter + reserveAfter
	return result, nil
}

// PreparePrepaidSpend releases enough reserve to satisfy a pending active
// deduction. It does no work for accounts without a reserve and therefore is
// intended for low-active-balance paths, not every request.
func PreparePrepaidSpend(targetType string, targetId int, required int) (PrepaidBalance, error) {
	var result PrepaidBalance
	if err := validatePrepaidTarget(targetType, targetId); err != nil {
		return result, err
	}
	if required < 0 || required > common.MaxQuota {
		return result, ErrPrepaidQuotaInvalid
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		activeBefore, err := loadPrepaidActiveQuotaTx(tx, targetType, targetId, true)
		if err != nil {
			return err
		}
		result, err = preparePrepaidSpendWithActiveTx(
			tx,
			targetType,
			targetId,
			0,
			activeBefore,
			required,
		)
		return err
	})
	if err != nil {
		return PrepaidBalance{}, err
	}
	if targetType == PrepaidTargetUser {
		if err := RefreshPrepaidTargetCache(targetType, targetId); err != nil {
			common.SysLog("failed to refresh user quota cache after prepaid reserve release: " + err.Error())
		}
	}
	return result, nil
}

// RefreshPrepaidTargetCache synchronizes the existing active-quota cache after
// a committed Stripe credit, reversal, restoration, or reserve release.
func RefreshPrepaidTargetCache(targetType string, targetId int) error {
	if targetType != PrepaidTargetUser || !common.RedisEnabled {
		return nil
	}
	lock := userQuotaMutationLock(targetId)
	lock.Lock()
	defer lock.Unlock()
	active, err := loadPrepaidActiveQuotaTx(DB, targetType, targetId, false)
	if err != nil {
		return err
	}
	return updateUserQuotaCache(targetId, int(active))
}
