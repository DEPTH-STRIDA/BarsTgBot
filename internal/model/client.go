package model

import "gorm.io/gorm"

type Client struct {
	gorm.Model
	Name           string `json:"name" gorm:"type:varchar(255)"`
	Username       string `json:"username" gorm:"type:varchar(255)"`
	Phone          string `json:"phone" gorm:"type:varchar(32);uniqueIndex:client_phone_bar_unique"`
	Bar            string `json:"bar" gorm:"type:varchar(255);uniqueIndex:client_phone_bar_unique"`
	RegistrationAt string `json:"registration_at" gorm:"type:varchar(64)"`
	SheetIsSynced  bool   `json:"sheet_is_synced" gorm:"default:false"`
}
