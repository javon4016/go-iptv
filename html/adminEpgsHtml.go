package html

import (
	"fmt"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Epgs(c *gin.Context) {
	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	var pageData = dto.AdminEpgsDto{
		LoginUser: username,
		Title:     "EPG管理",
	}

	recCountsStr := c.DefaultQuery("recCounts", "20")
	jumptoStr := c.DefaultQuery("jumpto", "")
	pageStr := c.DefaultQuery("page", "")

	if !until.IsSafe(recCountsStr) || !until.IsSafe(jumptoStr) || !until.IsSafe(pageStr) {
		recCountsStr = "20"
		jumptoStr = ""
		pageStr = ""
	}

	recCounts, err := strconv.ParseInt(recCountsStr, 10, 64)
	if err != nil {
		// 转换失败时设置默认值，比如 20
		recCounts = 20
	}
	pageData.RecCounts = recCounts

	if jumptoStr != "" {
		pageData.Page, err = strconv.ParseInt(jumptoStr, 10, 64)
		if err != nil {
			pageData.Page = 1
		}
	} else if pageStr != "" {
		pageData.Page, err = strconv.ParseInt(pageStr, 10, 64)
		if err != nil {
			pageData.Page = 1
		}
	} else {
		pageData.Page = 1
	}

	pageData.Keywords = c.DefaultQuery("keywords", "") // 关键词

	if !until.IsSafe(pageData.Keywords) {
		pageData.Keywords = ""
	}

	recStart := recCounts * (pageData.Page - 1)
	dbQuery := dao.DB.Model(&models.IptvEpg{})
	if pageData.Keywords != "" {
		keywords := "%" + pageData.Keywords + "%" // 模糊查询
		dbQuery = dbQuery.Where("name like ? or remarks like ? or content like ?", keywords, keywords, keywords)
	}

	var count int64
	err = dbQuery.Count(&count).Error
	if err != nil {
		pageData.PageCount = 1
	} else {
		if count == 0 {
			pageData.PageCount = 1
		} else {
			pageData.PageCount = int64(math.Ceil(float64(count) / float64(recCounts)))
		}
	}

	err = dbQuery.Offset(int(recStart)).Limit(int(recCounts)).Find(&pageData.Epgs).Error
	if err != nil {
		log.Println("查询epg失败:", err)
	}

	cfg := dao.GetConfig()
	var query string = "enable = 1 and type not like 'auto%'"
	if dao.Lic.Type != 0 && cfg.Proxy.Status == 1 && cfg.Aggregation.Status == 1 {
		query = "enable = 1"
	}

	dao.DB.Model(&models.IptvEpgList{}).Find(&pageData.EpgFromDb)
	dao.DB.Model(&models.IptvCategory{}).Where(query).Find(&pageData.CaList)

	logoList := until.GetLogos() // 获取logo列表

	for k, v := range pageData.Epgs {
		for _, logo := range logoList {
			logoName := strings.Split(logo, ".")[0]
			if strings.EqualFold(v.Name, logoName) {
				pageData.Epgs[k].Logo = "/logo/" + logo
			}
		}
		for _, a := range strings.Split(v.FromListStr, ",") {
			if a == "0" {
				pageData.Epgs[k].FromName += "CCTV官网,"
				continue
			}
			for _, b := range pageData.EpgFromDb {
				if a == fmt.Sprintf("%d", b.ID) {
					pageData.Epgs[k].FromName += b.Name + ","
				}
			}
		}
		pageData.Epgs[k].FromName = strings.TrimRight(pageData.Epgs[k].FromName, ",")
	}

	c.HTML(200, "admin_epgs.html", pageData)
}

func EpgsFrom(c *gin.Context) {
	username, ok := until.GetAuthName(c)
	if !ok {
		c.JSON(200, dto.NewAdminRedirectDto())
		return
	}
	var pageData = dto.AdminEpgsDto{
		LoginUser: username,
		Title:     "EPG源管理",
	}

	dao.DB.Model(&models.IptvEpgList{}).Find(&pageData.EpgFromDb)

	pageData.EpgFromList = make(map[string]string)
	for k, v := range pageData.EpgFromDb {
		pageData.EpgFromDb[k].LastTimeStr = time.Unix(v.LastTime, 0).Format("2006-01-02 15:04:05")
		pageData.EpgFromList[v.Name] = v.Remarks
	}

	c.HTML(200, "admin_epgs_list.html", pageData)
}
