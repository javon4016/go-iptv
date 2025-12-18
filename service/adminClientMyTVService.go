package service

import (
	"encoding/json"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/until"
	"log"
	"net/url"
	"strconv"
	"time"
)

var buildStatus int64 = 0

func SetMyTVAppInfo(params url.Values) dto.ReturnJsonDto {

	var status int64 = 0
	res, err := dao.WS.SendWS(dao.Request{Action: "getMyTVBuildStatus"})
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "连接引擎失败", Type: "danger"}
	} else if res.Code != 1 {
		return dto.ReturnJsonDto{Code: 0, Msg: res.Msg, Type: "danger"}
	} else {
		if err := json.Unmarshal(res.Data, &status); err != nil {
			log.Println("⚠️ 无法解析引擎返回的状态:", err)
			return dto.ReturnJsonDto{Code: 0, Msg: "连接引擎失败", Type: "danger"}
		}
	}

	if status == 1 {
		return dto.ReturnJsonDto{Code: 0, Msg: "正在打包中，请稍后再试", Type: "danger"}
	}
	appServerUrl := params.Get("serverUrl")
	appVersion := params.Get("app_version")
	upBody := params.Get("up_body")

	if appVersion == "" || appServerUrl == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}

	appVersionInt, err := strconv.ParseInt(appVersion, 10, 64)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "版本号为纯数字", Type: "danger"}
	}
	appVersion = strconv.FormatInt(appVersionInt, 10)

	if appVersionInt <= 0 || appVersionInt > 999 {
		return dto.ReturnJsonDto{Code: 0, Msg: "版本号范围为1-999的纯数字", Type: "danger"}
	}

	cfg := dao.GetConfig()

	if cfg.MyTV.Version == appVersion {
		return dto.ReturnJsonDto{Code: 0, Msg: "版本号不能相同", Type: "danger"}
	}

	cfg.MyTV.Version = appVersion

	if cfg.ServerUrl != appServerUrl {
		cfg.ServerUrl = appServerUrl
	}

	cfg.MyTV.Update = upBody

	// cfg.App.Update.Url = strings.TrimSuffix(cfg.ServerUrl, "/") + "/app/" + cfg.Build.Name + ".apk"

	res, err = dao.WS.SendWS(dao.Request{Action: "buildMyTV", Data: cfg})
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "连接引擎失败", Type: "danger"}
	}
	if res.Code == 1 {
		buildStatus = 1
		go waitMyTVBuildReady(cfg)
		return dto.ReturnJsonDto{Code: 1, Msg: "APK编译中...", Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 0, Msg: "APK编译出错，请查看引擎日志", Type: "danger"}

}

func waitMyTVBuildReady(cfg *dto.Config) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if getMyTVBuildStatus(cfg) {
			log.Println("MyTV 编译完成")
			return
		}
	}
}
func getMyTVBuildStatus(cfg *dto.Config) bool {

	res, err := dao.WS.SendWS(dao.Request{Action: "getMyTVBuildStatus"})
	if err != nil {
		return false
	} else if res.Code != 1 {
		return false
	} else {
		if err := json.Unmarshal(res.Data, &buildStatus); err != nil {
			log.Println("⚠️ 无法解析引擎返回的状态:", err)
			return false
		}
	}

	if buildStatus == 0 {
		dao.SetConfig(cfg)
		return true
	}
	return false
}

func GetMyTVBuildStatus() dto.ReturnJsonDto {
	if buildStatus == 1 {
		return dto.ReturnJsonDto{Code: 0, Msg: "APK编译中...", Type: "danger", Data: map[string]interface{}{"size": until.GetFileSize("/config/app/清和IPTV-mytv.apk")}}
	} else {
		cfg := dao.GetConfig()
		return dto.ReturnJsonDto{Code: 1, Msg: "APK编译完成", Type: "success", Data: map[string]interface{}{"size": until.GetFileSize("/config/app/清和IPTV-mytv.apk"), "version": cfg.MyTV.Version, "url": "/app/清和IPTV-mytv.apk", "name": "清和IPTV-mytv-1.2.0." + cfg.MyTV.Version + ".apk"}}
	}
}

func MytvReleases() dto.MyTvDto {
	cfg := dao.GetConfig()
	return dto.MyTvDto{
		Version:     "1.2.0." + cfg.MyTV.Version,
		DownloadUrl: cfg.ServerUrl + "/app/清和IPTV-mytv.apk",
		UpdateMsg:   cfg.MyTV.Update,
	}
}
