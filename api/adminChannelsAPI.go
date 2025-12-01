package api

import (
	"go-iptv/dto"
	"go-iptv/service"
	"go-iptv/until"

	"github.com/gin-gonic/gin"
)

func Channels(c *gin.Context) {
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
		case "update_interval":
			res = service.UpdateInterval(params)
		case "updatelist":
			res = service.UpdateList(params)
		case "updatelistall":
			res = service.UpdateListAll()
		case "addlist":
			res = service.AddList(params)
		case "dellist":
			res = service.DelList(params)
		case "getchannels":
			res = service.CaGetChannels(params)
		case "delca":
			res = service.DelCa(params)
		case "moveup":
			res = service.SubmitMoveUp(params)
		case "movedown":
			res = service.SubmitMoveDown(params)
		case "movetop":
			res = service.SubmitMoveTop(params)
		case "saveChannels":
			res = service.SubmitSave(params)
		case "saveChannelsOne":
			res = service.SaveChannelsOne(params)
		case "categoryStatus":
			res = service.CategoryChangeStatus(params)
		case "categoryListStatus":
			res = service.CategoryListChangeStatus(params)
		// case "channelsStatus":
		// 	res = service.ChannelsChangeStatus(params)
		case "saveCa":
			res = service.SaveCategory(params)
		case "testResolutionOne":
			res = service.TestResolutionOne(params)
		}
	}

	c.JSON(200, res)
}

func UploadPayList(c *gin.Context) {
	_, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	c.JSON(200, service.UploadPayList(c))
}
