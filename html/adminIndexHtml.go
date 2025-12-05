package html

import (
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"time"

	"github.com/gin-gonic/gin"
)

func Index(c *gin.Context) {

	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}

	var pageData = dto.AdminIndexDto{
		LoginUser: username,
		Title:     "首页",
	}

	today := time.Now().Truncate(24 * time.Hour).Unix()

	cfg := dao.GetConfig()
	var query string = "enable = 1 and type not like 'auto%'"
	if dao.Lic.Type != 0 && cfg.Proxy.Status == 1 && cfg.Aggregation.Status == 1 {
		query = "enable = 1"
	}

	dao.DB.Model(&models.IptvUser{}).Count(&pageData.UserTotal)
	dao.DB.Model(&models.IptvUser{}).Where("lasttime > ?", today).Count(&pageData.UserToday)
	dao.DB.Model(&models.IptvCategory{}).Where(query).Count(&pageData.ChannelTypeCount)
	dao.DB.Model(&models.IptvChannel{}).Where("status = 1").Count(&pageData.ChannelCount)
	dao.DB.Model(&models.IptvEpg{}).Where("status = 1").Count(&pageData.EpgCount)
	dao.DB.Model(&models.IptvMeals{}).Where("status = 1").Count(&pageData.MealsCount)

	var categoryList []models.IptvCategory
	dao.DB.Model(&models.IptvCategory{}).Where("enable = 1").Find(&categoryList)
	for i, c := range categoryList {
		var channelType dto.ChannelType

		tmpCh := until.CaGetChannels(c, true)
		channelType.ChannelCount = int64(len(tmpCh))
		channelType.Num = int64(i + 1)
		channelType.Name = c.Name
		if c.Type == "add" {
			channelType.ShowRawCount = true
		}
		channelType.RawCount = c.Rawcount
		pageData.ChannelTypeList = append(pageData.ChannelTypeList, channelType)
	}

	c.HTML(200, "admin_index.html", pageData)
}
