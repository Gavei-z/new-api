package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEnterpriseInquiryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousDatabaseType := common.MainDatabaseType()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&model.EnterpriseInquiry{}))
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func runEnterpriseInquiryHandler(body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise/inquiries", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	CreateEnterpriseInquiry(context)
	return recorder
}

func TestCreateEnterpriseInquiryValidatesAndPersistsContact(t *testing.T) {
	db := setupEnterpriseInquiryTestDB(t)
	recorder := runEnterpriseInquiryHandler(`{
		"company_name":" Acme Labs ",
		"contact_name":" Alice ",
		"email":"ALICE@EXAMPLE.COM",
		"team_size":"11-50",
		"expected_monthly_usage":"2B tokens",
		"message":"Need team access",
		"privacy_accepted":true,
		"website":""
	}`)
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var inquiry model.EnterpriseInquiry
	require.NoError(t, db.First(&inquiry).Error)
	assert.Equal(t, "Acme Labs", inquiry.CompanyName)
	assert.Equal(t, "Alice", inquiry.ContactName)
	assert.Equal(t, "alice@example.com", inquiry.Email)
	assert.Equal(t, model.EnterpriseInquiryStatusPending, inquiry.Status)
}

func TestCreateEnterpriseInquiryRejectsMissingContactAndConsent(t *testing.T) {
	db := setupEnterpriseInquiryTestDB(t)
	recorder := runEnterpriseInquiryHandler(`{
		"company_name":"Acme Labs",
		"contact_name":"Alice",
		"privacy_accepted":false,
		"website":""
	}`)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	var count int64
	require.NoError(t, db.Model(&model.EnterpriseInquiry{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreateEnterpriseInquiryHoneypotDoesNotPersist(t *testing.T) {
	db := setupEnterpriseInquiryTestDB(t)
	recorder := runEnterpriseInquiryHandler(`{
		"company_name":"Bot Company",
		"contact_name":"Bot",
		"email":"bot@example.com",
		"privacy_accepted":true,
		"website":"https://spam.example"
	}`)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	var count int64
	require.NoError(t, db.Model(&model.EnterpriseInquiry{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreateEnterpriseInquiryCountsUnicodeCharacters(t *testing.T) {
	db := setupEnterpriseInquiryTestDB(t)
	validCompanyName := strings.Repeat("企", 120)
	recorder := runEnterpriseInquiryHandler(fmt.Sprintf(`{
		"company_name":%q,
		"contact_name":"张三",
		"wechat":"zhangsan",
		"privacy_accepted":true,
		"website":""
	}`, validCompanyName))
	assert.Equal(t, http.StatusCreated, recorder.Code)

	tooLongCompanyName := strings.Repeat("企", 121)
	recorder = runEnterpriseInquiryHandler(fmt.Sprintf(`{
		"company_name":%q,
		"contact_name":"张三",
		"wechat":"zhangsan",
		"privacy_accepted":true,
		"website":""
	}`, tooLongCompanyName))
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var count int64
	require.NoError(t, db.Model(&model.EnterpriseInquiry{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}
