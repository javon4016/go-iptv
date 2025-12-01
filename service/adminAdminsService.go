package service

import (
	"encoding/json"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"log"
	"net/url"
	"runtime"
)

func Admins(params url.Values) dto.ReturnJsonDto {
	username := params.Get("username")
	oldPassword := params.Get("oldpassword")
	newpassword := params.Get("newpassword")
	newpassword2 := params.Get("newpassword_2")

	if username == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "用户名不能为空", Type: "danger"}
	}
	if oldPassword == "" && (newpassword != "" || newpassword2 != "") {
		return dto.ReturnJsonDto{Code: 0, Msg: "旧密码不能为空"}
	}

	if newpassword != newpassword2 && newpassword != "" && newpassword2 != "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "两次新密码不一致", Type: "danger"}
	}

	if oldPassword == newpassword && newpassword != "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "新密码不能与旧密码相同", Type: "danger"}
	}

	if !until.IsSafe(username) {
		return dto.ReturnJsonDto{Code: 0, Msg: "用户名不合法", Type: "danger"}
	}

	var adminData models.IptvAdmin
	dao.DB.Model(&models.IptvAdmin{}).Where("id = ?", 1).First(&adminData)
	if adminData.PassWord != until.HashPassword(oldPassword) {
		return dto.ReturnJsonDto{Code: 0, Msg: "旧密码错误", Type: "danger"}
	}

	dao.DB.Model(&models.IptvAdmin{}).Where("id = ?", 1).Updates(map[string]interface{}{
		"password": until.HashPassword(newpassword),
		"username": username,
	})

	// TODO
	return dto.ReturnJsonDto{Code: 1, Msg: "修改成功", Type: "success"}

	// TODO
}

func UpdataCheckWeb() dto.ReturnJsonDto {
	oldWeb := until.GetVersion()

	up, newWeb, err := until.CheckNewVerWeb(oldWeb)

	if err != nil {
		if up {
			return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "success"}
		}
		return dto.ReturnJsonDto{Code: 0, Msg: "检查更新失败: " + err.Error(), Type: "danger"}
	}
	if up {
		return dto.ReturnJsonDto{Code: 1, Msg: "管理系统有新版本: " + newWeb, Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 2, Msg: "当前已是最新版本", Type: "success"}
}

func UpdataCheckLic() dto.ReturnJsonDto {
	var oldLic string
	verJson, err := dao.WS.SendWS(dao.Request{Action: "getVersion"})

	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "检查更新失败: " + err.Error(), Type: "danger"}
	}
	if err := json.Unmarshal(verJson.Data, &oldLic); err != nil {
		log.Println("版本信息解析错误:", err)
		return dto.ReturnJsonDto{Code: 0, Msg: "引擎版本信息解析错误，请检查引擎是否正常", Type: "danger"}
	}
	up, newLic, err := until.CheckNewVerLic(oldLic)

	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "检查更新失败: " + err.Error(), Type: "danger"}
	}
	if up {
		return dto.ReturnJsonDto{Code: 1, Msg: "引擎有新版本: " + newLic, Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 2, Msg: "当前已是最新版本", Type: "success"}
}

func UpdataDownWeb() dto.ReturnJsonDto {
	up, newWeb, err := until.DownloadAndVerifyWeb(runtime.GOARCH)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "下载失败: " + err.Error(), Type: "danger"}
	}
	if up {
		return dto.ReturnJsonDto{Code: 1, Msg: "管理系统新版本: " + newWeb, Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 0, Msg: "下载失败", Type: "danger"}
}

func UpdataDownLic() dto.ReturnJsonDto {
	up, newLic, err := until.DownloadAndVerifyLic(runtime.GOARCH)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "下载失败: " + err.Error(), Type: "danger"}
	}
	if up {
		return dto.ReturnJsonDto{Code: 1, Msg: "引擎新版本: " + newLic, Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 0, Msg: "下载失败", Type: "danger"}
}

func Updata() dto.ReturnJsonDto {
	go until.UpdateSignal()
	return dto.ReturnJsonDto{Code: 1, Msg: "已触发更新，请稍后刷新...", Type: "success"}
}
