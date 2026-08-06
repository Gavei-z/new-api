package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	TeamStatusEnabled  = 1
	TeamStatusDisabled = 2

	TeamMemberStatusEnabled  = 1
	TeamMemberStatusDisabled = 2

	TeamMemberRoleMember = 1
	TeamMemberRoleAdmin  = 10
	TeamMemberRoleOwner  = 100

	TeamQuotaTypeAdminGrant       = "admin_grant"
	TeamQuotaTypeAdminDeduction   = "admin_deduction"
	TeamQuotaTypeRecharge         = "recharge"
	TeamQuotaTypeConsume          = "consume"
	TeamQuotaTypeSettlement       = "settlement"
	TeamQuotaTypeRefund           = "refund"
	TeamQuotaTypeTaskAdjustment   = "task_adjustment"
	TeamQuotaTypeManualCorrection = "manual_correction"
)

var (
	ErrTeamNotFound            = errors.New("team not found")
	ErrTeamDisabled            = errors.New("team is disabled")
	ErrTeamAccessDenied        = errors.New("team access denied")
	ErrTeamQuotaInsufficient   = errors.New("team quota is insufficient")
	ErrTeamQuotaOutOfRange     = errors.New("team quota is out of range")
	ErrTeamIdempotencyConflict = errors.New("team quota idempotency key conflicts with an existing change")
	ErrUserAlreadyInTeam       = errors.New("user already belongs to a team")
	ErrTeamMemberNotFound      = errors.New("team member not found")
	ErrTeamMemberCannotManage  = errors.New("team member cannot manage target")
	ErrTeamMemberCannotDelete  = errors.New("team members must be disabled instead of deleted")
	ErrTeamMemberCannotPromote = errors.New("team members cannot be promoted to global administrators")
	ErrTeamMemberPersonalQuota = errors.New("team member quota must be adjusted through the team balance")
)

// Team is an independent B2B tenant. User.Group remains a routing/billing
// group and is intentionally not reused as an ownership boundary.
type Team struct {
	Id        int    `json:"id"`
	Name      string `json:"name" gorm:"type:varchar(120);not null;index"`
	Slug      string `json:"slug" gorm:"type:varchar(64);not null;uniqueIndex"`
	Status    int    `json:"status" gorm:"type:int;not null;default:1;index"`
	Quota     int    `json:"quota" gorm:"type:int;not null;default:0"`
	UsedQuota int64  `json:"used_quota" gorm:"type:bigint;not null;default:0"`
	// ReserveQuota and TotalQuota are response-only values populated from the
	// bigint prepaid reserve. They are never persisted in the legacy team row.
	ReserveQuota int64 `json:"reserve_quota" gorm:"-"`
	TotalQuota   int64 `json:"total_quota" gorm:"-"`
	CreatedBy    int   `json:"created_by" gorm:"index"`
	CreatedAt    int64 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    int64 `json:"updated_at" gorm:"autoUpdateTime"`
}

// TeamMember keeps tenant membership out of the existing User model. A user
// may belong to one team for an unambiguous funding source.
type TeamMember struct {
	Id        int   `json:"id"`
	TeamId    int   `json:"team_id" gorm:"not null;uniqueIndex:idx_team_member;index"`
	UserId    int   `json:"user_id" gorm:"not null;uniqueIndex:idx_team_member;uniqueIndex"`
	Role      int   `json:"role" gorm:"type:int;not null;default:1;index"`
	Status    int   `json:"status" gorm:"type:int;not null;default:1;index"`
	CreatedBy int   `json:"created_by" gorm:"index"`
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime"`
}

// TeamQuotaTransaction is the immutable audit trail for every balance change.
// IdempotencyKey makes billing retries safe across request retries.
type TeamQuotaTransaction struct {
	Id                  int64  `json:"id"`
	TeamId              int    `json:"team_id" gorm:"not null;index"`
	UserId              int    `json:"user_id" gorm:"index"`
	ActorUserId         int    `json:"actor_user_id" gorm:"index"`
	Type                string `json:"type" gorm:"type:varchar(32);not null;index"`
	QuotaDelta          int    `json:"quota_delta" gorm:"type:int;not null"`
	UsedQuotaDelta      int64  `json:"used_quota_delta" gorm:"type:bigint;not null"`
	BalanceAfter        int    `json:"balance_after" gorm:"type:int;not null"`
	ReserveBalanceAfter int64  `json:"reserve_balance_after" gorm:"type:bigint;not null;default:0"`
	TotalBalanceAfter   int64  `json:"total_balance_after" gorm:"type:bigint;not null;default:0"`
	UsedQuotaAfter      int64  `json:"used_quota_after" gorm:"type:bigint;not null"`
	IdempotencyKey      string `json:"idempotency_key" gorm:"type:varchar(191);not null;uniqueIndex"`
	Note                string `json:"note" gorm:"type:varchar(255)"`
	CreatedAt           int64  `json:"created_at" gorm:"autoCreateTime;index"`
}

type TeamContext struct {
	TeamId       int    `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	TeamStatus   int    `json:"status"`
	MemberStatus int    `json:"member_status"`
	Role         int    `json:"role"`
	RoleName     string `json:"role_name"`
	IsManager    bool   `json:"is_manager"`
}

type TeamMemberView struct {
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	UserStatus   int    `json:"user_status"`
	Role         int    `json:"role"`
	MemberStatus int    `json:"member_status"`
	UsedQuota    int64  `json:"used_quota"`
	RequestCount int    `json:"request_count"`
	CreatedAt    int64  `json:"created_at"`
}

type TeamUsageRow struct {
	UserId           int    `json:"user_id"`
	Username         string `json:"username"`
	ModelName        string `json:"model_name"`
	RequestCount     int64  `json:"request_count"`
	Quota            int64  `json:"quota"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
}

type TeamCreateOwner struct {
	Username    string
	Password    string
	DisplayName string
	Email       string
}

type TeamQuotaChange struct {
	TeamId         int
	UserId         int
	ActorUserId    int
	Type           string
	QuotaDelta     int
	UsedQuotaDelta int
	IdempotencyKey string
	Note           string
	RequireEnabled bool
}

func teamRoleName(role int) string {
	switch role {
	case TeamMemberRoleOwner:
		return "owner"
	case TeamMemberRoleAdmin:
		return "admin"
	default:
		return "member"
	}
}

func GetTeamContextByUserId(userId int) (*TeamContext, error) {
	var member TeamMember
	if err := DB.Where("user_id = ?", userId).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var team Team
	if err := DB.Where("id = ?", member.TeamId).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return &TeamContext{
		TeamId:       team.Id,
		Name:         team.Name,
		Slug:         team.Slug,
		TeamStatus:   team.Status,
		MemberStatus: member.Status,
		Role:         member.Role,
		RoleName:     teamRoleName(member.Role),
		IsManager: member.Status == TeamMemberStatusEnabled &&
			team.Status == TeamStatusEnabled &&
			member.Role >= TeamMemberRoleAdmin,
	}, nil
}

func EnsureUserCanBeDeleted(userId int) error {
	isMember, err := IsTeamMember(userId)
	if err != nil {
		return err
	}
	if isMember {
		return ErrTeamMemberCannotDelete
	}
	return nil
}

func IsTeamMember(userId int) (bool, error) {
	if DB == nil || !DB.Migrator().HasTable(&TeamMember{}) {
		return false, nil
	}
	var count int64
	if err := DB.Model(&TeamMember{}).Where("user_id = ?", userId).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetActiveTeamFundingContext(userId int) (*TeamContext, *Team, error) {
	var member TeamMember
	if err := DB.Where("user_id = ?", userId).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if member.Status != TeamMemberStatusEnabled {
		return nil, nil, ErrTeamDisabled
	}
	var team Team
	if err := DB.Where("id = ?", member.TeamId).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrTeamNotFound
		}
		return nil, nil, err
	}
	if team.Status != TeamStatusEnabled {
		return nil, nil, ErrTeamDisabled
	}
	context := &TeamContext{
		TeamId:       team.Id,
		Name:         team.Name,
		Slug:         team.Slug,
		TeamStatus:   team.Status,
		MemberStatus: member.Status,
		Role:         member.Role,
		RoleName:     teamRoleName(member.Role),
		IsManager:    member.Role >= TeamMemberRoleAdmin,
	}
	return context, &team, nil
}

func GetTeamById(teamId int) (*Team, error) {
	var team Team
	if err := DB.Where("id = ?", teamId).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	if err := PopulateTeamPrepaidBalance(&team); err != nil {
		return nil, err
	}
	return &team, nil
}

func ListTeams() ([]Team, error) {
	var teams []Team
	if err := DB.Order("id desc").Find(&teams).Error; err != nil {
		return nil, err
	}
	for index := range teams {
		if err := PopulateTeamPrepaidBalance(&teams[index]); err != nil {
			return nil, err
		}
	}
	return teams, nil
}

func normalizeTeamSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func CreateTeamWithOwner(name string, slug string, owner TeamCreateOwner, createdBy int) (*Team, *User, error) {
	name = strings.TrimSpace(name)
	slug = normalizeTeamSlug(slug)
	owner.Username = strings.TrimSpace(owner.Username)
	owner.DisplayName = strings.TrimSpace(owner.DisplayName)
	owner.Email = NormalizeEmail(owner.Email)
	if name == "" || slug == "" || owner.Username == "" || owner.Password == "" {
		return nil, nil, errors.New("invalid team or owner")
	}
	if owner.DisplayName == "" {
		owner.DisplayName = owner.Username
	}

	team := &Team{
		Name:      name,
		Slug:      slug,
		Status:    TeamStatusEnabled,
		CreatedBy: createdBy,
	}
	user := &User{
		Username:    owner.Username,
		Password:    owner.Password,
		DisplayName: owner.DisplayName,
		Email:       owner.Email,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	if err := common.Validate.Struct(user); err != nil {
		return nil, nil, err
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := user.InsertWithTx(tx, 0); err != nil {
			return err
		}
		// Team members always spend the team wallet. Do not seed an unrelated
		// personal balance even when QuotaForNewUser is configured.
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", 0).Error; err != nil {
			return err
		}
		user.Quota = 0
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		member := TeamMember{
			TeamId:    team.Id,
			UserId:    user.Id,
			Role:      TeamMemberRoleOwner,
			Status:    TeamMemberStatusEnabled,
			CreatedBy: createdBy,
		}
		return tx.Create(&member).Error
	})
	if err != nil {
		return nil, nil, err
	}
	finishTeamUserInsert(user)
	return team, user, nil
}

func finishTeamUserInsert(user *User) {
	if user == nil || user.Id <= 0 {
		return
	}
	var createdUser User
	if err := DB.Where("id = ?", user.Id).First(&createdUser).Error; err != nil {
		common.SysLog("failed to initialize team user sidebar: " + err.Error())
		return
	}
	setting := createdUser.GetSetting()
	setting.SidebarModules = generateDefaultSidebarConfigForRole(createdUser.Role)
	createdUser.SetSetting(setting)
	if err := createdUser.Update(false); err != nil {
		common.SysLog("failed to initialize team user: " + err.Error())
	}
}

func requireTeamManagerTx(tx *gorm.DB, teamId int, actorUserId int) (*TeamMember, *Team, error) {
	var team Team
	if err := lockForUpdate(tx).Where("id = ?", teamId).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrTeamNotFound
		}
		return nil, nil, err
	}
	if team.Status != TeamStatusEnabled {
		return nil, nil, ErrTeamDisabled
	}
	var actor TeamMember
	if err := tx.Where("team_id = ? AND user_id = ?", teamId, actorUserId).First(&actor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrTeamAccessDenied
		}
		return nil, nil, err
	}
	if actor.Status != TeamMemberStatusEnabled || actor.Role < TeamMemberRoleAdmin {
		return nil, nil, ErrTeamAccessDenied
	}
	return &actor, &team, nil
}

func CreateTeamMemberUser(teamId int, actorUserId int, user *User) error {
	return createTeamMemberUser(teamId, actorUserId, user, false)
}

func CreateTeamMemberUserByRoot(teamId int, actorUserId int, user *User) error {
	return createTeamMemberUser(teamId, actorUserId, user, true)
}

func createTeamMemberUser(teamId int, actorUserId int, user *User, rootOverride bool) error {
	if user == nil {
		return errors.New("user is nil")
	}
	user.Username = strings.TrimSpace(user.Username)
	user.DisplayName = strings.TrimSpace(user.DisplayName)
	user.Email = NormalizeEmail(user.Email)
	if user.Username == "" || user.Password == "" {
		return errors.New("invalid user")
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	user.Role = common.RoleCommonUser
	user.Status = common.UserStatusEnabled
	if err := common.Validate.Struct(user); err != nil {
		return err
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if rootOverride {
			var team Team
			if err := lockForUpdate(tx).Where("id = ?", teamId).First(&team).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrTeamNotFound
				}
				return err
			}
			if team.Status != TeamStatusEnabled {
				return ErrTeamDisabled
			}
		} else {
			if _, _, err := requireTeamManagerTx(tx, teamId, actorUserId); err != nil {
				return err
			}
		}
		if err := user.InsertWithTx(tx, 0); err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", 0).Error; err != nil {
			return err
		}
		user.Quota = 0
		member := TeamMember{
			TeamId:    teamId,
			UserId:    user.Id,
			Role:      TeamMemberRoleMember,
			Status:    TeamMemberStatusEnabled,
			CreatedBy: actorUserId,
		}
		if err := tx.Create(&member).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return ErrUserAlreadyInTeam
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	finishTeamUserInsert(user)
	return nil
}

func ListTeamMembers(teamId int) ([]TeamMemberView, error) {
	var members []TeamMemberView
	err := DB.Table("team_members AS tm").
		Select("u.id AS user_id, u.username, u.display_name, u.email, u.status AS user_status, tm.role, tm.status AS member_status, u.used_quota, u.request_count, tm.created_at").
		Joins("JOIN users AS u ON u.id = tm.user_id").
		Where("tm.team_id = ?", teamId).
		Order("tm.role DESC, tm.id ASC").
		Scan(&members).Error
	return members, err
}

func SetTeamMemberStatus(teamId int, actorUserId int, targetUserId int, status int) error {
	return setTeamMemberStatus(teamId, actorUserId, targetUserId, status, false)
}

func SetTeamMemberStatusByRoot(teamId int, actorUserId int, targetUserId int, status int) error {
	return setTeamMemberStatus(teamId, actorUserId, targetUserId, status, true)
}

func setTeamMemberStatus(teamId int, actorUserId int, targetUserId int, status int, rootOverride bool) error {
	if status != TeamMemberStatusEnabled && status != TeamMemberStatusDisabled {
		return errors.New("invalid member status")
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var actor *TeamMember
		if !rootOverride {
			var err error
			actor, _, err = requireTeamManagerTx(tx, teamId, actorUserId)
			if err != nil {
				return err
			}
		}
		var target TeamMember
		if err := lockForUpdate(tx).Where("team_id = ? AND user_id = ?", teamId, targetUserId).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeamMemberNotFound
			}
			return err
		}
		if !rootOverride && (target.UserId == actorUserId || target.Role >= actor.Role) {
			return ErrTeamMemberCannotManage
		}
		if err := tx.Model(&TeamMember{}).Where("id = ?", target.Id).Update("status", status).Error; err != nil {
			return err
		}
		userStatus := common.UserStatusEnabled
		if status == TeamMemberStatusDisabled {
			userStatus = common.UserStatusDisabled
		}
		return tx.Model(&User{}).Where("id = ?", targetUserId).Updates(map[string]interface{}{
			"status":       userStatus,
			"auth_version": gorm.Expr("auth_version + 1"),
		}).Error
	})
	if err != nil {
		return err
	}
	if err := PublishUserAuthCache(targetUserId); err != nil {
		return err
	}
	if err := InvalidateUserTokensCache(targetUserId); err != nil {
		return err
	}
	_, err = RevokeAllUserSessions(targetUserId, "team_member_status_changed")
	return err
}

func ResetTeamMemberPassword(teamId int, actorUserId int, targetUserId int, password string) error {
	return resetTeamMemberPassword(teamId, actorUserId, targetUserId, password, false)
}

func ResetTeamMemberPasswordByRoot(teamId int, actorUserId int, targetUserId int, password string) error {
	return resetTeamMemberPassword(teamId, actorUserId, targetUserId, password, true)
}

func resetTeamMemberPassword(teamId int, actorUserId int, targetUserId int, password string, rootOverride bool) error {
	hashed, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var actor *TeamMember
		if !rootOverride {
			var err error
			actor, _, err = requireTeamManagerTx(tx, teamId, actorUserId)
			if err != nil {
				return err
			}
		}
		var target TeamMember
		if err := tx.Where("team_id = ? AND user_id = ?", teamId, targetUserId).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeamMemberNotFound
			}
			return err
		}
		if !rootOverride && (target.UserId == actorUserId || target.Role >= actor.Role) {
			return ErrTeamMemberCannotManage
		}
		return tx.Model(&User{}).Where("id = ?", targetUserId).Updates(map[string]interface{}{
			"password":     hashed,
			"auth_version": gorm.Expr("auth_version + 1"),
		}).Error
	})
	if err != nil {
		return err
	}
	if err := PublishUserAuthCache(targetUserId); err != nil {
		return err
	}
	if err := InvalidateUserTokensCache(targetUserId); err != nil {
		return err
	}
	_, err = RevokeAllUserSessions(targetUserId, "team_member_password_reset")
	return err
}

func validateTeamQuotaDelta(current int, delta int) (int, error) {
	if delta > 0 && current > common.MaxQuota-delta {
		return 0, ErrTeamQuotaOutOfRange
	}
	if delta < 0 && current < -delta {
		return 0, ErrTeamQuotaInsufficient
	}
	next := current + delta
	if next < 0 || next > common.MaxQuota {
		return 0, ErrTeamQuotaOutOfRange
	}
	return next, nil
}

func validateTeamUsedQuotaDelta(current int64, delta int) (int64, error) {
	if current < 0 || delta > common.MaxQuota || delta < -common.MaxQuota {
		return 0, ErrTeamQuotaOutOfRange
	}
	delta64 := int64(delta)
	if delta64 > 0 && current > math.MaxInt64-delta64 {
		return 0, ErrTeamQuotaOutOfRange
	}
	if delta64 < 0 && current < -delta64 {
		return 0, ErrTeamQuotaOutOfRange
	}
	next := current + delta64
	if next < 0 {
		return 0, ErrTeamQuotaOutOfRange
	}
	return next, nil
}

func applyTeamQuotaChangeTx(tx *gorm.DB, change TeamQuotaChange) error {
	if change.TeamId <= 0 || strings.TrimSpace(change.IdempotencyKey) == "" || len(change.IdempotencyKey) > 191 {
		return errors.New("invalid team quota change")
	}
	var team Team
	if err := lockForUpdate(tx).Where("id = ?", change.TeamId).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTeamNotFound
		}
		return err
	}
	if change.RequireEnabled && team.Status != TeamStatusEnabled {
		return ErrTeamDisabled
	}
	var existing TeamQuotaTransaction
	err := tx.Where("idempotency_key = ?", change.IdempotencyKey).First(&existing).Error
	if err == nil {
		if existing.TeamId != change.TeamId ||
			existing.UserId != change.UserId ||
			existing.ActorUserId != change.ActorUserId ||
			existing.Type != change.Type ||
			existing.QuotaDelta != change.QuotaDelta ||
			existing.UsedQuotaDelta != int64(change.UsedQuotaDelta) {
			return ErrTeamIdempotencyConflict
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if change.QuotaDelta < 0 && team.Quota < -change.QuotaDelta {
		balance, err := preparePrepaidSpendWithActiveTx(
			tx,
			PrepaidTargetTeam,
			team.Id,
			change.UserId,
			int64(team.Quota),
			-change.QuotaDelta,
		)
		if err != nil {
			return err
		}
		if balance.ActiveQuota > int64(common.MaxQuota) {
			return ErrTeamQuotaOutOfRange
		}
		team.Quota = int(balance.ActiveQuota)
	}
	nextQuota, err := validateTeamQuotaDelta(team.Quota, change.QuotaDelta)
	if err != nil {
		return err
	}
	nextUsed, err := validateTeamUsedQuotaDelta(team.UsedQuota, change.UsedQuotaDelta)
	if err != nil {
		return err
	}
	if err := tx.Model(&Team{}).Where("id = ?", team.Id).Updates(map[string]interface{}{
		"quota":      nextQuota,
		"used_quota": nextUsed,
	}).Error; err != nil {
		return err
	}
	var reserveBalanceAfter int64
	var reserve PrepaidReserve
	reserveErr := lockForUpdate(tx).
		Select("quota").
		Where("target_type = ? AND target_id = ?", PrepaidTargetTeam, team.Id).
		First(&reserve).Error
	if reserveErr != nil && !errors.Is(reserveErr, gorm.ErrRecordNotFound) {
		return reserveErr
	}
	if reserveErr == nil {
		if reserve.Quota < 0 || int64(nextQuota) > int64(^uint64(0)>>1)-reserve.Quota {
			return ErrPrepaidBalanceOutOfRange
		}
		reserveBalanceAfter = reserve.Quota
	}
	transaction := TeamQuotaTransaction{
		TeamId:              change.TeamId,
		UserId:              change.UserId,
		ActorUserId:         change.ActorUserId,
		Type:                change.Type,
		QuotaDelta:          change.QuotaDelta,
		UsedQuotaDelta:      int64(change.UsedQuotaDelta),
		BalanceAfter:        nextQuota,
		ReserveBalanceAfter: reserveBalanceAfter,
		TotalBalanceAfter:   int64(nextQuota) + reserveBalanceAfter,
		UsedQuotaAfter:      nextUsed,
		IdempotencyKey:      change.IdempotencyKey,
		Note:                strings.TrimSpace(change.Note),
	}
	noteRunes := []rune(transaction.Note)
	if len(noteRunes) > 255 {
		transaction.Note = string(noteRunes[:255])
	}
	return tx.Create(&transaction).Error
}

func ApplyTeamQuotaChange(change TeamQuotaChange) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return applyTeamQuotaChangeTx(tx, change)
	})
}

func TransferUserQuotaToTeam(teamId int, actorUserId int, amount int, idempotencyKey string) error {
	if amount <= 0 || amount > common.MaxQuota {
		return ErrTeamQuotaOutOfRange
	}
	applied := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if _, _, err := requireTeamManagerTx(tx, teamId, actorUserId); err != nil {
			return err
		}
		var existing TeamQuotaTransaction
		err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
		if err == nil {
			if existing.TeamId != teamId ||
				existing.UserId != actorUserId ||
				existing.ActorUserId != actorUserId ||
				existing.Type != TeamQuotaTypeRecharge ||
				existing.QuotaDelta != amount ||
				existing.UsedQuotaDelta != 0 {
				return ErrTeamIdempotencyConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var user User
		if err := lockForUpdate(tx).Select("id", "quota").Where("id = ?", actorUserId).First(&user).Error; err != nil {
			return err
		}
		if user.Quota < amount {
			balance, err := preparePrepaidSpendWithActiveTx(
				tx,
				PrepaidTargetUser,
				user.Id,
				actorUserId,
				int64(user.Quota),
				amount,
			)
			if err != nil {
				return err
			}
			if balance.ActiveQuota > int64(common.MaxQuota) {
				return ErrPrepaidBalanceOutOfRange
			}
			user.Quota = int(balance.ActiveQuota)
		}
		if user.Quota < amount {
			return fmt.Errorf("personal quota is insufficient")
		}
		if err := tx.Model(&User{}).Where("id = ?", actorUserId).Update("quota", user.Quota-amount).Error; err != nil {
			return err
		}
		if err := applyTeamQuotaChangeTx(tx, TeamQuotaChange{
			TeamId:         teamId,
			UserId:         actorUserId,
			ActorUserId:    actorUserId,
			Type:           TeamQuotaTypeRecharge,
			QuotaDelta:     amount,
			IdempotencyKey: idempotencyKey,
			Note:           "Transferred from personal wallet",
		}); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
		return err
	}
	if applied {
		if err := RefreshPrepaidTargetCache(PrepaidTargetUser, actorUserId); err != nil {
			common.SysLog("failed to refresh user quota cache after team recharge: " + err.Error())
		}
	}
	return nil
}

func ListTeamQuotaTransactions(teamId int, offset int, limit int) ([]TeamQuotaTransaction, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var total int64
	if err := DB.Model(&TeamQuotaTransaction{}).Where("team_id = ?", teamId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var transactions []TeamQuotaTransaction
	err := DB.Where("team_id = ?", teamId).
		Order("id desc").
		Offset(offset).
		Limit(limit).
		Find(&transactions).Error
	return transactions, total, err
}

func GetTeamUsage(teamId int, startTime int64, endTime int64) ([]TeamUsageRow, error) {
	var userIds []int
	if err := DB.Model(&TeamMember{}).Where("team_id = ?", teamId).Pluck("user_id", &userIds).Error; err != nil {
		return nil, err
	}
	if len(userIds) == 0 {
		return []TeamUsageRow{}, nil
	}
	query := LOG_DB.Table("logs").
		Select("user_id, username, model_name, count(*) AS request_count, COALESCE(sum(quota), 0) AS quota, COALESCE(sum(prompt_tokens), 0) AS prompt_tokens, COALESCE(sum(completion_tokens), 0) AS completion_tokens").
		Where("user_id IN ? AND type = ?", userIds, LogTypeConsume)
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	var rows []TeamUsageRow
	err := query.
		Group("user_id, username, model_name").
		Order("quota DESC").
		Scan(&rows).Error
	return rows, err
}

func UpdateTeamStatus(teamId int, status int) error {
	if status != TeamStatusEnabled && status != TeamStatusDisabled {
		return errors.New("invalid team status")
	}
	result := DB.Model(&Team{}).Where("id = ?", teamId).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTeamNotFound
	}
	return nil
}

// TeamContextForSelf is safe to attach to the dashboard user payload. Balance
// details remain behind team-manager endpoints.
func TeamContextForSelf(userId int) map[string]interface{} {
	if DB == nil {
		return nil
	}
	context, err := GetTeamContextByUserId(userId)
	if err != nil || context == nil {
		return nil
	}
	return map[string]interface{}{
		"id":            context.TeamId,
		"name":          context.Name,
		"slug":          context.Slug,
		"status":        context.TeamStatus,
		"member_status": context.MemberStatus,
		"role":          context.Role,
		"role_name":     context.RoleName,
		"is_manager":    context.IsManager,
	}
}
