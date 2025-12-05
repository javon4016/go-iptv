package dto

import (
	"go-iptv/models"
)

type AdminChannelsDto struct {
	LoginUser      string                    `json:"loginuser"`
	Title          string                    `json:"title"`
	AutoUpdate     bool                      `json:"autoupdate"`
	UpdateInterval int64                     `json:"updateinterval"`
	ShowAuto       bool                      `json:"showauto"`
	CategoryList   []models.IptvCategoryList `json:"categorylist"`
	Categorys      []models.IptvCategory     `json:"categorys"`
	Epgs           []models.IptvEpg          `json:"epgs"`
	Lic            Lic                       `json:"lic"`
}
