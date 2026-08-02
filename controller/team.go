package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var teamSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

type createTeamRequest struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Owner struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	} `json:"owner"`
}

type createTeamMemberRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type teamMemberStatusRequest struct {
	Status int `json:"status"`
}

type resetTeamMemberPasswordRequest struct {
	Password string `json:"password"`
}

type teamFundRequest struct {
	Amount         int    `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

type teamQuotaRequest struct {
	Operation      string `json:"operation"`
	Amount         int    `json:"amount"`
	Note           string `json:"note"`
	IdempotencyKey string `json:"idempotency_key"`
}

type teamStatusRequest struct {
	Status int `json:"status"`
}

func writeTeamError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, model.ErrTeamAccessDenied), errors.Is(err, model.ErrTeamMemberCannotManage):
		status = http.StatusForbidden
	case errors.Is(err, model.ErrTeamNotFound), errors.Is(err, model.ErrTeamMemberNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrTeamDisabled):
		status = http.StatusForbidden
	case errors.Is(err, model.ErrTeamQuotaInsufficient):
		status = http.StatusPaymentRequired
	}
	c.JSON(status, gin.H{"success": false, "message": err.Error()})
}

func teamIdFromContext(c *gin.Context) int {
	return c.GetInt(middleware.TeamContextIdKey)
}

func getTeamIdParam(c *gin.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, errors.New("invalid team id")
	}
	return id, nil
}

func runeLengthBetween(value string, minimum int, maximum int) bool {
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}

func GetCurrentTeam(c *gin.Context) {
	team, err := model.GetTeamById(teamIdFromContext(c))
	if err != nil {
		writeTeamError(c, err)
		return
	}
	members, err := model.ListTeamMembers(team.Id)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"team":         team,
			"member_count": len(members),
			"role":         c.GetInt(middleware.TeamContextRoleKey),
		},
	})
}

func GetCurrentTeamMembers(c *gin.Context) {
	members, err := model.ListTeamMembers(teamIdFromContext(c))
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": members})
}

func CreateCurrentTeamMember(c *gin.Context) {
	var request createTeamMemberRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	user := &model.User{
		Username:    strings.TrimSpace(request.Username),
		Password:    request.Password,
		DisplayName: strings.TrimSpace(request.DisplayName),
		Email:       model.NormalizeEmail(request.Email),
	}
	if err := common.Validate.Struct(user); err != nil {
		writeTeamError(c, err)
		return
	}
	if err := model.CreateTeamMemberUser(teamIdFromContext(c), c.GetInt("id"), user); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":  user.Id,
			"username": user.Username,
		},
	})
}

func UpdateCurrentTeamMemberStatus(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		writeTeamError(c, errors.New("invalid user id"))
		return
	}
	var request teamMemberStatusRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	if err := model.SetTeamMemberStatus(teamIdFromContext(c), c.GetInt("id"), userId, request.Status); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ResetCurrentTeamMemberPassword(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		writeTeamError(c, errors.New("invalid user id"))
		return
	}
	var request resetTeamMemberPasswordRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	if len(request.Password) < 8 || len(request.Password) > 20 {
		writeTeamError(c, errors.New("password must be between 8 and 20 characters"))
		return
	}
	if err := model.ResetTeamMemberPassword(teamIdFromContext(c), c.GetInt("id"), userId, request.Password); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func GetCurrentTeamUsage(c *gin.Context) {
	startTime, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	rows, err := model.GetTeamUsage(teamIdFromContext(c), startTime, endTime)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func GetCurrentTeamTransactions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	transactions, total, err := model.ListTeamQuotaTransactions(
		teamIdFromContext(c),
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(transactions)
	common.ApiSuccess(c, pageInfo)
}

func FundCurrentTeam(c *gin.Context) {
	var request teamFundRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if len(request.IdempotencyKey) < 8 || len(request.IdempotencyKey) > 128 {
		writeTeamError(c, errors.New("invalid idempotency key"))
		return
	}
	if err := model.TransferUserQuotaToTeam(
		teamIdFromContext(c),
		c.GetInt("id"),
		request.Amount,
		"team-fund:"+request.IdempotencyKey,
	); err != nil {
		writeTeamError(c, err)
		return
	}
	team, err := model.GetTeamById(teamIdFromContext(c))
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": team})
}

func AdminListTeams(c *gin.Context) {
	teams, err := model.ListTeams()
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": teams})
}

func AdminCreateTeam(c *gin.Context) {
	var request createTeamRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Slug = strings.ToLower(strings.TrimSpace(request.Slug))
	if !runeLengthBetween(request.Name, 2, 120) || !teamSlugPattern.MatchString(request.Slug) {
		writeTeamError(c, errors.New("invalid team name or slug"))
		return
	}
	owner := model.TeamCreateOwner{
		Username:    request.Owner.Username,
		Password:    request.Owner.Password,
		DisplayName: request.Owner.DisplayName,
		Email:       request.Owner.Email,
	}
	validationUser := &model.User{
		Username:    owner.Username,
		Password:    owner.Password,
		DisplayName: owner.DisplayName,
		Email:       model.NormalizeEmail(owner.Email),
	}
	if err := common.Validate.Struct(validationUser); err != nil {
		writeTeamError(c, err)
		return
	}
	team, user, err := model.CreateTeamWithOwner(request.Name, request.Slug, owner, c.GetInt("id"))
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"team": team,
			"owner": gin.H{
				"user_id":  user.Id,
				"username": user.Username,
			},
		},
	})
}

func AdminGetTeam(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	team, err := model.GetTeamById(teamId)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": team})
}

func AdminGetTeamMembers(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	members, err := model.ListTeamMembers(teamId)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": members})
}

func AdminCreateTeamMember(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	var request createTeamMemberRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	user := &model.User{
		Username:    strings.TrimSpace(request.Username),
		Password:    request.Password,
		DisplayName: strings.TrimSpace(request.DisplayName),
		Email:       model.NormalizeEmail(request.Email),
	}
	if err := common.Validate.Struct(user); err != nil {
		writeTeamError(c, err)
		return
	}
	if err := model.CreateTeamMemberUserByRoot(teamId, c.GetInt("id"), user); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":  user.Id,
			"username": user.Username,
		},
	})
}

func AdminUpdateTeamMemberStatus(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		writeTeamError(c, errors.New("invalid user id"))
		return
	}
	var request teamMemberStatusRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	if err := model.SetTeamMemberStatusByRoot(teamId, c.GetInt("id"), userId, request.Status); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminResetTeamMemberPassword(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		writeTeamError(c, errors.New("invalid user id"))
		return
	}
	var request resetTeamMemberPasswordRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	if len(request.Password) < 8 || len(request.Password) > 20 {
		writeTeamError(c, errors.New("password must be between 8 and 20 characters"))
		return
	}
	if err := model.ResetTeamMemberPasswordByRoot(teamId, c.GetInt("id"), userId, request.Password); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminGetTeamUsage(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	startTime, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	rows, err := model.GetTeamUsage(teamId, startTime, endTime)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func AdminGetTeamTransactions(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	transactions, total, err := model.ListTeamQuotaTransactions(teamId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		writeTeamError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(transactions)
	common.ApiSuccess(c, pageInfo)
}

func AdminAdjustTeamQuota(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	var request teamQuotaRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.Amount <= 0 {
		writeTeamError(c, errors.New("invalid quota adjustment"))
		return
	}
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if len(request.IdempotencyKey) < 8 || len(request.IdempotencyKey) > 128 {
		writeTeamError(c, errors.New("invalid idempotency key"))
		return
	}
	changeType := model.TeamQuotaTypeAdminGrant
	delta := request.Amount
	if request.Operation == "subtract" {
		changeType = model.TeamQuotaTypeAdminDeduction
		delta = -request.Amount
	} else if request.Operation != "add" {
		writeTeamError(c, errors.New("invalid quota operation"))
		return
	}
	if err := model.ApplyTeamQuotaChange(model.TeamQuotaChange{
		TeamId:         teamId,
		ActorUserId:    c.GetInt("id"),
		Type:           changeType,
		QuotaDelta:     delta,
		IdempotencyKey: "team-admin:" + strconv.Itoa(teamId) + ":" + request.IdempotencyKey,
		Note:           request.Note,
	}); err != nil {
		writeTeamError(c, err)
		return
	}
	team, err := model.GetTeamById(teamId)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": team})
}

func AdminUpdateTeamStatus(c *gin.Context) {
	teamId, err := getTeamIdParam(c)
	if err != nil {
		writeTeamError(c, err)
		return
	}
	var request teamStatusRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	if err := model.UpdateTeamStatus(teamId, request.Status); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type enterpriseInquiryRequest struct {
	CompanyName          string `json:"company_name"`
	ContactName          string `json:"contact_name"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	WeChat               string `json:"wechat"`
	TeamSize             string `json:"team_size"`
	ExpectedMonthlyUsage string `json:"expected_monthly_usage"`
	Message              string `json:"message"`
	PrivacyAccepted      bool   `json:"privacy_accepted"`
	Website              string `json:"website"`
}

func CreateEnterpriseInquiry(c *gin.Context) {
	var request enterpriseInquiryRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		writeTeamError(c, errors.New("invalid request"))
		return
	}
	// A hidden honeypot quietly absorbs basic form bots.
	if strings.TrimSpace(request.Website) != "" {
		c.JSON(http.StatusCreated, gin.H{"success": true})
		return
	}
	request.CompanyName = strings.TrimSpace(request.CompanyName)
	request.ContactName = strings.TrimSpace(request.ContactName)
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Phone = strings.TrimSpace(request.Phone)
	request.WeChat = strings.TrimSpace(request.WeChat)
	request.TeamSize = strings.TrimSpace(request.TeamSize)
	request.ExpectedMonthlyUsage = strings.TrimSpace(request.ExpectedMonthlyUsage)
	request.Message = strings.TrimSpace(request.Message)
	if !request.PrivacyAccepted ||
		!runeLengthBetween(request.CompanyName, 2, 120) ||
		!runeLengthBetween(request.ContactName, 2, 80) ||
		utf8.RuneCountInString(request.Email) > 120 ||
		utf8.RuneCountInString(request.Phone) > 40 ||
		utf8.RuneCountInString(request.WeChat) > 80 ||
		utf8.RuneCountInString(request.TeamSize) > 40 ||
		utf8.RuneCountInString(request.ExpectedMonthlyUsage) > 80 ||
		utf8.RuneCountInString(request.Message) > 2000 {
		writeTeamError(c, errors.New("invalid inquiry"))
		return
	}
	if request.Email == "" && request.Phone == "" && request.WeChat == "" {
		writeTeamError(c, errors.New("at least one contact method is required"))
		return
	}
	if request.Email != "" {
		if _, err := mail.ParseAddress(request.Email); err != nil {
			writeTeamError(c, errors.New("invalid email address"))
			return
		}
	}
	inquiry := &model.EnterpriseInquiry{
		CompanyName:          request.CompanyName,
		ContactName:          request.ContactName,
		Email:                request.Email,
		Phone:                request.Phone,
		WeChat:               request.WeChat,
		TeamSize:             request.TeamSize,
		ExpectedMonthlyUsage: request.ExpectedMonthlyUsage,
		Message:              request.Message,
	}
	if err := model.CreateEnterpriseInquiry(inquiry); err != nil {
		writeTeamError(c, err)
		return
	}
	notificationReceiver := service.EnterpriseInquiryNotificationReceiver()
	if notificationReceiver != "" {
		inquiryForNotification := *inquiry
		go func() {
			if err := service.SendEnterpriseInquiryNotification(inquiryForNotification, notificationReceiver); err != nil {
				common.SysError(fmt.Sprintf(
					"failed to send enterprise inquiry notification (inquiry_id=%d): %v",
					inquiryForNotification.Id,
					err,
				))
				return
			}
			common.SysLog(fmt.Sprintf(
				"enterprise inquiry notification sent (inquiry_id=%d)",
				inquiryForNotification.Id,
			))
		}()
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    gin.H{"inquiry_id": inquiry.Id},
	})
}

func AdminListEnterpriseInquiries(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	inquiries, total, err := model.ListEnterpriseInquiries(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		writeTeamError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(inquiries)
	common.ApiSuccess(c, pageInfo)
}

func AdminUpdateEnterpriseInquiryStatus(c *gin.Context) {
	inquiryId, err := strconv.Atoi(c.Param("id"))
	if err != nil || inquiryId <= 0 {
		writeTeamError(c, errors.New("invalid inquiry id"))
		return
	}
	var request struct {
		Status int `json:"status"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil ||
		(request.Status != model.EnterpriseInquiryStatusPending &&
			request.Status != model.EnterpriseInquiryStatusContacted &&
			request.Status != model.EnterpriseInquiryStatusClosed) {
		writeTeamError(c, errors.New("invalid inquiry status"))
		return
	}
	if err := model.UpdateEnterpriseInquiryStatus(inquiryId, request.Status); err != nil {
		writeTeamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
