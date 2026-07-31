package model

import "gorm.io/gorm"

const (
	EnterpriseInquiryStatusPending   = 1
	EnterpriseInquiryStatusContacted = 2
	EnterpriseInquiryStatusClosed    = 3
)

// EnterpriseInquiry stores a public enterprise quote request. It is separate
// from users and teams because an inquiry is not an authenticated tenant yet.
type EnterpriseInquiry struct {
	Id                   int    `json:"id"`
	CompanyName          string `json:"company_name" gorm:"type:varchar(120);not null;index"`
	ContactName          string `json:"contact_name" gorm:"type:varchar(80);not null"`
	Email                string `json:"email" gorm:"type:varchar(120);index"`
	Phone                string `json:"phone" gorm:"type:varchar(40)"`
	WeChat               string `json:"wechat" gorm:"type:varchar(80)"`
	TeamSize             string `json:"team_size" gorm:"type:varchar(40)"`
	ExpectedMonthlyUsage string `json:"expected_monthly_usage" gorm:"type:varchar(80)"`
	Message              string `json:"message" gorm:"type:text"`
	Status               int    `json:"status" gorm:"type:int;not null;default:1;index"`
	CreatedAt            int64  `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt            int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func CreateEnterpriseInquiry(inquiry *EnterpriseInquiry) error {
	inquiry.Status = EnterpriseInquiryStatusPending
	return DB.Create(inquiry).Error
}

func ListEnterpriseInquiries(offset int, limit int) ([]EnterpriseInquiry, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var total int64
	if err := DB.Model(&EnterpriseInquiry{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var inquiries []EnterpriseInquiry
	err := DB.Order("id desc").Offset(offset).Limit(limit).Find(&inquiries).Error
	return inquiries, total, err
}

func UpdateEnterpriseInquiryStatus(id int, status int) error {
	result := DB.Model(&EnterpriseInquiry{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
