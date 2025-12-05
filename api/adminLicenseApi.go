package api

import (
	"fmt"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/service"
	"go-iptv/until"

	"github.com/gin-gonic/gin"
)

func License(c *gin.Context) {
	_, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	c.Request.ParseForm()
	params := c.Request.PostForm
	var res dto.ReturnJsonDto

	for k := range params {
		switch k {
		case "proxy":
			res = service.Proxy(params)
		case "resEng":
			res = service.ResEng()
		case "autoRes":
			res = service.AutoRes(params)
		case "disCh":
			res = service.DisCh(params)
		case "epgFuzz":
			res = service.EpgFuzz(params)
		case "aggStatus":
			res = service.AggStatus(params)
		case "register":
			res = service.Register(params)
		case "login":
			res = service.Login(params)
		case "logout":
			res = service.Logout()
		}

	}

	c.JSON(200, res)
}

func CheckProxy(c *gin.Context) {
	_, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	cfg := dao.GetConfig()

	url := fmt.Sprintf("%s://%s:%d/status", cfg.Proxy.Scheme, cfg.Proxy.PAddr, cfg.Proxy.Port)
	c.JSON(200, until.GetUrlData(url))
}
