package models

type IptvCategory struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"unique;column:name" json:"name"`
	Enable    int64  `gorm:"column:enable;default:1" json:"enable"`
	Type      string `gorm:"default:user;column:type" json:"type"`
	Proxy     int64  `gorm:"column:proxy" json:"proxy"`
	ReName    int64  `gorm:"column:rename" json:"rename"`
	Ku9       string `gorm:"column:ku9" json:"ku9"`
	UA        string `gorm:"column:ua" json:"ua"`
	Sort      int64  `gorm:"column:sort" json:"sort"`
	ListId    int64  `gorm:"column:list_id;default:0" json:"list_id"`
	Rules     string `gorm:"column:rules" json:"rules"` // 规则
	RulesShow string `gorm:"-" json:"rules_show"`       // 规则
	Rawcount  int64  `gorm:"column:rawcount;default:0" json:"rawcount"`
}

func (IptvCategory) TableName() string {
	return "iptv_category"
}

type IptvCategoryList struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"unique;column:name" json:"name"`
	Enable       int64  `gorm:"column:enable;default:1" json:"enable"`
	Url          string `gorm:"column:url" json:"url"`
	UA           string `gorm:"column:ua" json:"ua"`
	LatestTime   string `gorm:"column:latesttime" json:"latesttime"`
	AutoCategory int64  `gorm:"column:autocategory" json:"autocategory"`
	AutoGroup    int64  `gorm:"column:autogroup" json:"autogroup"`
	Ku9          int64  `gorm:"column:ku9" json:"ku9"`
	Repeat       int64  `gorm:"column:repeat" json:"repeat"` //是否去重
	ReName       int64  `gorm:"column:rename" json:"rename"`
}

func (IptvCategoryList) TableName() string {
	return "iptv_category_list"
}
