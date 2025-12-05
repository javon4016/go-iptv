package html

import (
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"strings"

	"github.com/gin-gonic/gin"
)

func Meals(c *gin.Context) {
	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	var pageData = dto.AdminMealsDto{
		LoginUser: username,
		Title:     "套餐管理",
	}

	dao.DB.Model(&models.IptvMeals{}).Find(&pageData.Meals)

	cfg := dao.GetConfig()
	var query string = "enable = 1 and type not like 'auto%'"
	if dao.Lic.Type != 0 && cfg.Proxy.Status == 1 && cfg.Aggregation.Status == 1 {
		query = "enable = 1"
	}

	var tmpCas []models.IptvCategory
	dao.DB.Model(&models.IptvCategory{}).Where(query).Find(&tmpCas)
	pageData.ChannelNum = int64(len(tmpCas))

	for i, meal := range pageData.Meals {
		caIds := strings.Split(meal.Content, ",")
		for _, v := range tmpCas {
			if until.Int64InStringSlice(v.ID, caIds) {
				pageData.Meals[i].CaName += v.Name + ","
			}
		}
		if len(pageData.Meals[i].CaName) > 0 {
			pageData.Meals[i].CaName = pageData.Meals[i].CaName[:len(pageData.Meals[i].CaName)-1]
		}
	}

	c.HTML(200, "admin_meals.html", pageData)
}
