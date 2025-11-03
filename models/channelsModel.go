package models

type IptvChannel struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name" json:"name"`
	Url        string `gorm:"column:url" json:"url"`
	Resolution string `gorm:"column:resolution" json:"resolution"`
	Status     int64  `gorm:"column:status" json:"status"`
	Sort       int64  `gorm:"column:sort" json:"sort"`
	EId        int64  `gorm:"column:e_id" json:"e_id"`
	CId        int64  `gorm:"column:c_id" json:"c_id"`
	ListId     int64  `gorm:"column:list_id" json:"list_id"`
}

func (IptvChannel) TableName() string {
	return "iptv_channels"
}

type IptvChannelShow struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name" json:"name"`
	Url        string `gorm:"column:url" json:"url"`
	Resolution string `gorm:"column:resolution" json:"resolution"`
	Status     int64  `gorm:"column:status" json:"status"`
	Sort       int64  `gorm:"column:sort" json:"sort"`
	EId        int64  `gorm:"column:e_id" json:"e_id"`
	CId        int64  `gorm:"column:c_id" json:"c_id"`
	ListId     int64  `gorm:"column:list_id" json:"list_id"`
	EpgName    string `gorm:"column:epg_name" json:"epg_name"`
	Logo       string `gorm:"-" json:"logo"`
	PUrl       string `gorm:"-" json:"purl"`
}

func (IptvChannelShow) TableName() string {
	return "iptv_channels"
}
