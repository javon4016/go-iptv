package dto

type AdminLicenseDto struct {
	LoginUser   string `json:"loginuser"`
	Title       string `json:"title"`
	Scheme      string `json:"scheme"`
	Proxy       int64  `json:"proxy"`
	Port        int64  `json:"port"`
	ProxyAddr   string `json:"proxy_addr"`
	Lic         Lic    `json:"lic"`
	Status      int64  `json:"status"`
	Online      int64  `json:"online"`
	Version     string `json:"version"`
	AutoRes     int64  `json:"auto_res"`
	DisCh       int64  `json:"dis_ch"`
	EpgFuzz     int64  `json:"epg_fuzz"`
	Aggregation int64  `json:"aggregation"`
}

type Lic struct {
	ID     string `json:"id"`
	Type   int64  `json:"type"`
	Status int64  `json:"status"`
	Count  int64  `json:"count"`
	Exp    int64  `json:"exp"`
	Msg    string `json:"msg"`
	Name   string `json:"name"`
	ExpStr string `json:"exp_str"`
}

type LoginDto struct {
	Name string `json:"name"`
	Pwd  string `json:"pwd"`
	Pwd2 string `json:"pwd2"`
}
