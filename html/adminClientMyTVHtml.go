package html

import (
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/until"

	"github.com/gin-gonic/gin"
)

func ClientMyTV(c *gin.Context) {
	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	cfg := dao.GetConfig()

	if dao.Lic.Type == 0 {
		c.Redirect(302, "/admin/license")
	}

	var pageData = dto.AdminClientMyTVDto{
		LoginUser: username,
		Title:     "MyTV客户端设置",
		MyTV:      cfg.MyTV,
		ApkName:   "清和IPTV-mytv.apk",
		ApkUrl:    "/app/清和IPTV-mytv.apk", // APK下载地址
		UpSize:    until.GetFileSize("/config/app/清和IPTV-mytv.apk"),
		ServerUrl: cfg.ServerUrl,
	}

	if until.Exists("/config/images/icon/icon.png") {
		pageData.IconUrl = "/icon/icon.png"
	}

	c.HTML(200, "admin_client_mytv.html", pageData)
}
