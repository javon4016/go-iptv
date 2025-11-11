package api

import (
	"go-iptv/dto"
	"go-iptv/service"
	"go-iptv/until"

	"github.com/gin-gonic/gin"
)

func Epgs(c *gin.Context) {
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
		case "bdingepg":
			res = service.GetChName(params)
		case "epgGetCa":
			res = service.GetCa(params)
		case "save_epg":
			res = service.SaveEpg(params)
		case "bding_save_epg":
			res = service.BdingEpg(params)
		case "change_status":
			res = service.ChangeStatus(params)
		case "delepg":
			res = service.DeleteEpg(params)
		case "bindchannel":
			res = service.BindChannel()
		case "clearbind":
			res = service.ClearBind()
		case "clearcache":
			res = service.ClearCache()
		case "deleteLogo":
			res = service.DeleteLogo(params)

		}

	}
	c.JSON(200, res)
}

func EpgsFrom(c *gin.Context) {
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
		case "change_status":
			res = service.ChangeListStatus(params)
		case "updatelist":
			res = service.UpdateEpgList(params)
		case "dellist":
			res = service.DelEpgList(params)
		case "epgImport":
			res = service.EpgImport(params)
		case "updatelistall":
			res = service.UpdateEpgListAll()
		}
	}
	c.JSON(200, res)
}

func UploadLogo(c *gin.Context) {
	_, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	c.JSON(200, service.UploadLogo(c))
}
