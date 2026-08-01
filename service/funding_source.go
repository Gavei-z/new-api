package service

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// ---------------------------------------------------------------------------
// FundingSource — 资金来源接口（钱包 or 订阅）
// ---------------------------------------------------------------------------

// FundingSource 抽象了预扣费的资金来源。
type FundingSource interface {
	// Source 返回资金来源标识："wallet" 或 "subscription"
	Source() string
	// PreConsume 从该资金来源预扣 amount 额度
	PreConsume(amount int) error
	// Settle 根据差额调整资金来源（正数补扣，负数退还）
	Settle(delta int) error
	// Refund 退还所有预扣费
	Refund() error
}

// ---------------------------------------------------------------------------
// WalletFunding — 钱包资金来源实现
// ---------------------------------------------------------------------------

type WalletFunding struct {
	userId   int
	consumed int // 实际预扣的用户额度
}

func (w *WalletFunding) Source() string { return BillingSourceWallet }

func (w *WalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	if err := model.DecreaseUserQuota(w.userId, amount, false); err != nil {
		return err
	}
	w.consumed = amount
	return nil
}

func (w *WalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return model.DecreaseUserQuota(w.userId, delta, false)
	}
	return model.IncreaseUserQuota(w.userId, -delta, false)
}

func (w *WalletFunding) Refund() error {
	if w.consumed <= 0 {
		return nil
	}
	// IncreaseUserQuota 是 quota += N 的非幂等操作，不能重试，否则会多退额度。
	// 订阅的 RefundSubscriptionPreConsume 有 requestId 幂等保护所以可以重试。
	return model.IncreaseUserQuota(w.userId, w.consumed, false)
}

// ---------------------------------------------------------------------------
// TeamFunding — tenant wallet funding source
// ---------------------------------------------------------------------------

type TeamFunding struct {
	teamId              int
	userId              int
	requestId           string
	consumed            int
	reserveSequence     int
	lastReserveSequence int
}

func (t *TeamFunding) Source() string { return BillingSourceTeam }

func (t *TeamFunding) key(phase string) string {
	return fmt.Sprintf("team:%d:request:%s:%s", t.teamId, t.requestId, phase)
}

func (t *TeamFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	err := model.ApplyTeamQuotaChange(model.TeamQuotaChange{
		TeamId:         t.teamId,
		UserId:         t.userId,
		ActorUserId:    t.userId,
		Type:           model.TeamQuotaTypeConsume,
		QuotaDelta:     -amount,
		UsedQuotaDelta: amount,
		IdempotencyKey: t.key("preconsume"),
		RequireEnabled: true,
	})
	if err != nil {
		return err
	}
	t.consumed = amount
	return nil
}

func (t *TeamFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	err := model.ApplyTeamQuotaChange(model.TeamQuotaChange{
		TeamId:         t.teamId,
		UserId:         t.userId,
		ActorUserId:    t.userId,
		Type:           model.TeamQuotaTypeSettlement,
		QuotaDelta:     -delta,
		UsedQuotaDelta: delta,
		IdempotencyKey: t.key("settle"),
	})
	if err != nil {
		return err
	}
	t.consumed += delta
	return nil
}

func (t *TeamFunding) Refund() error {
	if t.consumed <= 0 {
		return nil
	}
	return model.ApplyTeamQuotaChange(model.TeamQuotaChange{
		TeamId:         t.teamId,
		UserId:         t.userId,
		ActorUserId:    t.userId,
		Type:           model.TeamQuotaTypeRefund,
		QuotaDelta:     t.consumed,
		UsedQuotaDelta: -t.consumed,
		IdempotencyKey: t.key("refund"),
	})
}

func (t *TeamFunding) Reserve(delta int) error {
	if delta <= 0 {
		return nil
	}
	t.reserveSequence++
	sequence := t.reserveSequence
	err := model.ApplyTeamQuotaChange(model.TeamQuotaChange{
		TeamId:         t.teamId,
		UserId:         t.userId,
		ActorUserId:    t.userId,
		Type:           model.TeamQuotaTypeConsume,
		QuotaDelta:     -delta,
		UsedQuotaDelta: delta,
		IdempotencyKey: t.key(fmt.Sprintf("reserve:%d", sequence)),
		RequireEnabled: true,
	})
	if err != nil {
		return err
	}
	t.lastReserveSequence = sequence
	t.consumed += delta
	return nil
}

func (t *TeamFunding) RollbackReserve(delta int) error {
	if delta <= 0 || t.lastReserveSequence <= 0 {
		return nil
	}
	sequence := t.lastReserveSequence
	err := model.ApplyTeamQuotaChange(model.TeamQuotaChange{
		TeamId:         t.teamId,
		UserId:         t.userId,
		ActorUserId:    t.userId,
		Type:           model.TeamQuotaTypeRefund,
		QuotaDelta:     delta,
		UsedQuotaDelta: -delta,
		IdempotencyKey: t.key(fmt.Sprintf("reserve:%d:rollback", sequence)),
	})
	if err == nil {
		t.consumed -= delta
		t.lastReserveSequence = 0
	}
	return err
}

// ---------------------------------------------------------------------------
// SubscriptionFunding — 订阅资金来源实现
// ---------------------------------------------------------------------------

type SubscriptionFunding struct {
	requestId      string
	userId         int
	modelName      string
	amount         int64 // 预扣的订阅额度（subConsume）
	subscriptionId int
	preConsumed    int64
	// 以下字段在 PreConsume 成功后填充，供 RelayInfo 同步使用
	AmountTotal     int64
	AmountUsedAfter int64
	PlanId          int
	PlanTitle       string
}

func (s *SubscriptionFunding) Source() string { return BillingSourceSubscription }

func (s *SubscriptionFunding) PreConsume(_ int) error {
	// amount 参数被忽略，使用内部 s.amount（已在构造时根据 preConsumedQuota 计算）
	res, err := model.PreConsumeUserSubscription(s.requestId, s.userId, s.modelName, 0, s.amount)
	if err != nil {
		return err
	}
	s.subscriptionId = res.UserSubscriptionId
	s.preConsumed = res.PreConsumed
	s.AmountTotal = res.AmountTotal
	s.AmountUsedAfter = res.AmountUsedAfter
	// 获取订阅计划信息
	if planInfo, err := model.GetSubscriptionPlanInfoByUserSubscriptionId(res.UserSubscriptionId); err == nil && planInfo != nil {
		s.PlanId = planInfo.PlanId
		s.PlanTitle = planInfo.PlanTitle
	}
	return nil
}

func (s *SubscriptionFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	return model.PostConsumeUserSubscriptionDelta(s.subscriptionId, int64(delta))
}

func (s *SubscriptionFunding) Refund() error {
	if s.preConsumed <= 0 {
		return nil
	}
	return refundWithRetry(func() error {
		return model.RefundSubscriptionPreConsume(s.requestId)
	})
}

// refundWithRetry 尝试多次执行退款操作以提高成功率，只能用于基于事务的退款函数！！！！！！
// try to refund with retries, only for refund functions based on transactions!!!
func refundWithRetry(fn func() error) error {
	if fn == nil {
		return nil
	}
	const maxAttempts = 3
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < maxAttempts-1 {
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
	}
	return lastErr
}
