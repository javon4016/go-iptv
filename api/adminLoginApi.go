package api

import (
	"go-iptv/dto"
	"go-iptv/service"
	"go-iptv/until"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	username := c.PostForm("username")
	password := until.HashPassword(c.PostForm("password"))
	remember := c.PostForm("rememberpass")

	reMe := false
	if remember == "on" || remember == "true" || remember == "1" {
		reMe = true
	}
	res := service.AdminLogin(username, password, reMe)

	token, ok := res.Data.(string)
	if !ok || token == "" {
		c.JSON(200, res)
		return
	}

	if reMe {
		c.SetCookie("token", token, 7*24*3600, "/", "", false, true)
	} else {
		c.SetCookie("token", token, 2*3600, "/", "", false, true)
	}
	res.Data = nil
	c.JSON(200, res)
}

func Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(200, dto.ReturnJsonDto{Code: 1, Msg: "退出登录成功", Type: "success"})
}
