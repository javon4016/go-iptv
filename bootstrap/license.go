package bootstrap

import (
	"encoding/json"
	"go-iptv/dao"
	"log"
)

func InitLicense() {
	// dao.StartLicense()
	log.Println("license初始化中")
	ws, err := dao.ConLicense("ws://127.0.0.1:81/ws")
	if err != nil {
		log.Println("license初始化错误")
		return
	}
	dao.WS = ws
	res, err := dao.WS.SendWS(dao.Request{Action: "getlic"})
	if err == nil {
		if err := json.Unmarshal(res.Data, &dao.Lic); err == nil {
			log.Println("license初始化成功")
			log.Println("机器码:", dao.Lic.ID)
		} else {
			log.Println("license信息解析错误:", err)
		}
	} else {
		log.Println("license初始化错误")
		return
	}
}
