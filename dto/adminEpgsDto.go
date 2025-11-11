package dto

import (
	"go-iptv/models"
)

type AdminEpgsDto struct {
	LoginUser   string                `json:"loginuser"`
	Title       string                `json:"title"`
	Epgs        []models.IptvEpg      `json:"epgs"`
	PageCount   int64                 `json:"pagecount"`
	EpgFromDb   []models.IptvEpgList  `json:"epgfromdb"`
	CaList      []models.IptvCategory `json:"calist"`
	EpgFromList map[string]string     `json:"epgfromlist"`
	Page        int64                 `json:"page"`      // 当前页数
	Keywords    string                `json:"keywords"`  // 搜索关键字
	RecCounts   int64                 `json:"recCounts"` // 每页显示条数
	// EpgErr    EPGErrors        `json:"epgerr"` // epg错误信息
	// EPGApiChk int64            `json:"epgapichk"`
}

type EpgsReturnDto struct {
	Id     int64  `json:"id"`
	Name   string `json:"name"`
	Select bool   `json:"select"`
}
