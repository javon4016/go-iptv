package dto

type IndexDto struct {
	ApkTime      string `json:"apk_time"`
	ApkVersion   string `json:"apk_version"`
	ApkUrl       string `json:"apk_url"`
	ApkSize      string `json:"apk_size"`
	ApkName      string `json:"apk_name"`
	Content      string `json:"content"`
	Status       int64  `json:"status"`
	ShowDownMyTV bool   `json:"show_down_mytv"`
	MyTVName     string `json:"mytv_name"`
	MyTVUrl      string `json:"mytv_url"`
}
