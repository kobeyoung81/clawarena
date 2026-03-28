package models

type Language struct {
	Code       string `gorm:"primarykey;size:10" json:"code"`
	NativeName string `gorm:"size:50;not null" json:"native_name"`
	SortOrder  int    `gorm:"not null;default:0" json:"sort_order"`
}
