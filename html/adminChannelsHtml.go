package html

import (
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"strings"

	"github.com/gin-gonic/gin"
)

func Channels(c *gin.Context) {
	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	var pageData = dto.AdminChannelsDto{
		LoginUser: username,
		Title:     "频道列表",
	}
	pageData.Lic = dao.Lic
	pageData.ShowAuto = false
	cfg := dao.GetConfig()

	var query string = "type not like 'auto%'"
	if pageData.Lic.Type != 0 && cfg.Proxy.Status == 1 && cfg.Aggregation.Status == 1 {
		pageData.ShowAuto = true
		query = "1=1"
	}

	autoUpdate := cfg.Channel.Auto
	if autoUpdate == 1 {
		pageData.AutoUpdate = true
	} else {
		pageData.AutoUpdate = false
	}

	pageData.UpdateInterval = cfg.Channel.Interval

	dao.DB.Model(&models.IptvCategoryList{}).Find(&pageData.CategoryList)
	dao.DB.Model(&models.IptvCategory{}).Where(query).Order("sort ASC").Find(&pageData.Categorys)
	dao.DB.Model(&models.IptvEpg{}).Where("status = 1").Find(&pageData.Epgs)

	for i, ch := range pageData.Categorys {
		if len(ch.Rules) > 10 {
			pageData.Categorys[i].RulesShow = ch.Rules[:10] + "..."
			continue
		}
		pageData.Categorys[i].RulesShow = ch.Rules
	}

	logoList := until.GetLogos()
	for i, v := range pageData.Epgs {
		for _, logo := range logoList {
			logoName := strings.Split(logo, ".")[0]
			if strings.EqualFold(v.Name, logoName) {
				pageData.Epgs[i].Logo = "/logo/" + logo
			}
		}
	}

	c.HTML(200, "admin_channels.html", pageData)
}
