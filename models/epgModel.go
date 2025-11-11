package models

type IptvEpg struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Content     string `gorm:"column:content" json:"content"`
	CasStr      string `gorm:"column:cas" json:"cas"`
	FromListStr string `gorm:"column:fromlist" json:"fromlist"`
	Status      int64  `gorm:"column:status" json:"status"`
	Remarks     string `gorm:"column:remarks" json:"remarks"`
	FromName    string `gorm:"-" json:"fromname"`
	Logo        string `gorm:"-" json:"logo"`
}

func (IptvEpg) TableName() string {
	return "iptv_epg"
}

type IptvEpgList struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Remarks     string `gorm:"column:remarks" json:"remarks"`
	Url         string `gorm:"column:url" json:"url"`
	UA          string `gorm:"column:ua" json:"ua"`
	LastTime    int64  `gorm:"column:lasttime" json:"lasttime"`
	LastTimeStr string `gorm:"-" json:"lasttimeStr"`
	Status      int64  `gorm:"column:status" json:"status"`
}

func (IptvEpgList) TableName() string {
	return "iptv_epg_list"
}
