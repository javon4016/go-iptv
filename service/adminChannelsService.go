package service

import (
	"errors"
	"fmt"
	"go-iptv/crontab"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CaGetChannels(params url.Values) dto.ReturnJsonDto {

	caId := params.Get("caId")
	if caId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表id", Type: "danger"}
	}

	var categoryDb models.IptvCategory

	if err := dao.DB.Where("id = ?", caId).First(&categoryDb).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "该分类不存在", Type: "danger"}
	}

	channels := until.CaGetChannels(categoryDb, true)

	return dto.ReturnJsonDto{Code: 1, Msg: "获取成功", Type: "success", Data: channels}
}

func UpdateInterval(params url.Values) dto.ReturnJsonDto {
	updateinterval := params.Get("updateinterval")
	autoupdate := params.Get("autoupdate")

	if updateinterval == "" || updateinterval == "0" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入更新时间", Type: "danger"}
	}

	if !until.IsSafe(updateinterval) || !until.IsSafe(autoupdate) {
		return dto.ReturnJsonDto{Code: 0, Msg: "输入不合法", Type: "danger"}
	}

	if autoupdate == "" || autoupdate == "0" {
		autoupdate = "0"
	} else {
		autoupdate = "1"
	}

	autoInt, err := strconv.ParseInt(autoupdate, 10, 64)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入数字", Type: "danger"}
	}

	interval, err := strconv.ParseInt(updateinterval, 10, 64)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入数字", Type: "danger"}
	}

	cfg := dao.GetConfig()

	cfg.Channel.Auto = autoInt
	cfg.Channel.Interval = interval
	dao.SetConfig(cfg)

	if autoInt == 1 && interval > 0 {
		crontab.StopChan = make(chan struct{})
		go crontab.Crontab()
	}
	if autoInt == 0 {
		close(crontab.StopChan)
		crontab.CrontabStatus = false
	}

	return dto.ReturnJsonDto{Code: 1, Msg: "更新成功", Type: "success"}
}

func AddList(params url.Values) dto.ReturnJsonDto {
	listName := params.Get("listname")
	url := strings.TrimSpace(params.Get("listurl"))
	ua := params.Get("listua")
	clId := params.Get("clId")
	autocategory := params.Get("autocategory")
	autogroup := params.Get("autogroup")
	ku9 := params.Get("ku9")
	repeat := params.Get("repeat")
	rename := params.Get("rename")

	if listName == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表", Type: "danger"}
	}

	if !until.IsSafe(listName) || !until.IsSafe(autocategory) || !until.IsSafe(autogroup) || !until.IsSafe(clId) {
		return dto.ReturnJsonDto{Code: 0, Msg: "输入不合法", Type: "danger"}
	}

	iptvCategoryList := models.IptvCategoryList{Name: listName, Url: url, UA: ua}

	if clId == "" {
		var category models.IptvCategoryList
		dao.DB.Model(&models.IptvCategoryList{}).Where("name = ?", listName).Find(&category)
		if category.Name != "" {
			return dto.ReturnJsonDto{Code: 0, Msg: "该列表名称存在", Type: "danger"}
		}
	} else {
		id, err := strconv.ParseInt(clId, 10, 64)
		if err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "cId请输入数字", Type: "danger"}
		}
		iptvCategoryList.ID = id
		if iptvCategoryList.ID != 0 {
			var cOld models.IptvCategoryList
			if err := dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", iptvCategoryList.ID).First(&cOld).Error; err != nil {
				return dto.ReturnJsonDto{Code: 0, Msg: "该频道列表不存在", Type: "danger"}
			}
			dao.DB.Model(&models.IptvChannel{}).Where("list_id = ?", cOld.ID).Delete(&models.IptvChannel{})
			dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", cOld.ID).Delete(&models.IptvCategory{})
		}
	}

	if autocategory == "on" || autocategory == "1" || autocategory == "true" {
		iptvCategoryList.AutoCategory = 1
		if autogroup == "on" || autogroup == "1" || autogroup == "true" {
			iptvCategoryList.AutoGroup = 1
		}
		if ku9 == "on" || ku9 == "1" || ku9 == "true" {
			iptvCategoryList.Ku9 = 1
		}
	}

	if rename == "on" || rename == "1" || rename == "true" {
		iptvCategoryList.ReName = 1
	}

	var doRepeat bool = false
	if repeat == "on" || repeat == "1" || repeat == "true" {
		iptvCategoryList.Repeat = 1
		doRepeat = true
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败-创建请求错误:" + err.Error(), Type: "danger"}
	}

	// 添加自定义 User-Agent
	req.Header.Set("User-Agent", ua)

	resp, err := client.Do(req)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败-无法访问url:" + err.Error(), Type: "danger"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败-状态码:" + strconv.Itoa(resp.StatusCode), Type: "danger"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败", Type: "danger"}
	}

	urlData := until.FilterEmoji(string(body))

	if until.IsM3UContent(urlData) {
		urlData = until.M3UToGenreTXT(urlData)
	}

	if !strings.Contains(urlData, "#genre#") && iptvCategoryList.AutoCategory == 1 {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到分组, 无法使用自动分组", Type: "danger"}
	}

	if iptvCategoryList.AutoCategory == 1 {
		iptvCategoryList.LatestTime = time.Now().Format("2006-01-02 15:04:05")
		if iptvCategoryList.ID != 0 {
			iptvCategoryList.Enable = 1
			dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", iptvCategoryList.ID).Save(&iptvCategoryList)
		} else {
			dao.DB.Model(&models.IptvCategoryList{}).Create(&iptvCategoryList)
		}

		if iptvCategoryList.AutoGroup == 1 {
			return GenreChannels(urlData, iptvCategoryList, doRepeat, true)
		}
		return GenreChannels(urlData, iptvCategoryList, doRepeat, false)
	} else {
		iptvCategoryList.LatestTime = time.Now().Format("2006-01-02 15:04:05")
		if iptvCategoryList.ID != 0 {
			iptvCategoryList.Enable = 1
			dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", iptvCategoryList.ID).Save(&iptvCategoryList)
		} else {
			dao.DB.Model(&models.IptvCategoryList{}).Create(&iptvCategoryList)
		}

		var maxSort int64
		dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)

		var iptvCategory = models.IptvCategory{
			Name:   iptvCategoryList.Name,
			Enable: 1,
			Type:   "add",
			Sort:   maxSort + 1,
			ListId: iptvCategoryList.ID,
			UA:     iptvCategoryList.UA,
			ReName: iptvCategoryList.ReName,
		}
		dao.DB.Model(&models.IptvCategory{}).Create(&iptvCategory)
		go until.SyncCaToEpg(iptvCategory.ID)
		repeat, err := until.AddChannelList(urlData, iptvCategory.ID, iptvCategoryList.ID, doRepeat)
		if err == nil {
			return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("更新列表 %s 成功，重复 %d 条\n", listName, repeat), Type: "success"}
		} else {
			return dto.ReturnJsonDto{Code: 0, Msg: fmt.Sprintf("更新列表 %s 失败\n", listName), Type: "danger"}
		}
	}
}

func UpdateList(params url.Values) dto.ReturnJsonDto {
	listId := params.Get("updatelist")
	if listId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表", Type: "danger"}
	}

	crontab.UpdateStatus = true
	defer func() { crontab.UpdateStatus = false }()

	var iptvCategoryList models.IptvCategoryList
	res := dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", listId).First(&iptvCategoryList)

	if res.RowsAffected == 0 {
		return dto.ReturnJsonDto{Code: 0, Msg: "频道列表不存在", Type: "danger"}
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", iptvCategoryList.Url, nil)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败-创建请求错误:" + err.Error(), Type: "danger"}
	}

	// 添加自定义 User-Agent
	req.Header.Set("User-Agent", iptvCategoryList.UA)

	resp, err := client.Do(req)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败-无法访问url:" + err.Error(), Type: "danger"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败-状态码:" + strconv.Itoa(resp.StatusCode), Type: "danger"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败", Type: "danger"}
	}

	urlData := until.FilterEmoji(string(body)) // 过滤emoji表情

	if until.IsM3UContent(urlData) {
		urlData = until.M3UToGenreTXT(urlData)
	}

	var doRepeat = false
	if iptvCategoryList.Repeat == 1 {
		doRepeat = true
	}

	updata := map[string]interface{}{
		"latesttime": time.Now().Format("2006-01-02 15:04:05"),
	}

	if iptvCategoryList.AutoCategory == 1 {
		if !strings.Contains(urlData, "#genre#") {
			updata["autocategory"] = 0
			dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", listId).Updates(updata)

			var oldC models.IptvCategory
			err := dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", iptvCategoryList.ID).First(&oldC).Error
			if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
				var maxSort int64
				dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)

				oldC = models.IptvCategory{
					Name:   iptvCategoryList.Name,
					Enable: 1,
					Type:   "add",
					Sort:   maxSort + 1,
					ListId: iptvCategoryList.ID,
					UA:     iptvCategoryList.UA,
					ReName: iptvCategoryList.ReName,
				}
				dao.DB.Model(&models.IptvCategory{}).Create(&oldC)
				go until.SyncCaToEpg(oldC.ID)
			}

			repeat, err := until.AddChannelList(urlData, oldC.ID, iptvCategoryList.ID, doRepeat)
			if err == nil {
				return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("更新列表 %s 成功，重复 %d 条\n", iptvCategoryList.Name, repeat), Type: "success"}
			} else {
				return dto.ReturnJsonDto{Code: 0, Msg: fmt.Sprintf("更新列表 %s 失败\n", iptvCategoryList.Name), Type: "danger"}
			}
		}
		if iptvCategoryList.AutoGroup == 1 {
			return GenreChannels(urlData, iptvCategoryList, doRepeat, true)
		}
		return GenreChannels(urlData, iptvCategoryList, doRepeat, false)
	} else {
		dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", listId).Updates(updata)
		var oldC models.IptvCategory
		err := dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", iptvCategoryList.ID).First(&oldC).Error

		if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
			var maxSort int64
			dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)

			oldC = models.IptvCategory{
				Name:   iptvCategoryList.Name,
				Enable: 1,
				Type:   "add",
				Sort:   maxSort + 1,
				ListId: iptvCategoryList.ID,
				ReName: iptvCategoryList.ReName,
			}
			dao.DB.Model(&models.IptvCategory{}).Create(&oldC)
			go until.SyncCaToEpg(oldC.ID)
		}

		repeat, err := until.AddChannelList(urlData, oldC.ID, iptvCategoryList.ID, doRepeat)
		if err == nil {
			return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("更新列表 %s 成功，重复 %d 条\n", iptvCategoryList.Name, repeat), Type: "success"}
		} else {
			return dto.ReturnJsonDto{Code: 0, Msg: fmt.Sprintf("更新列表 %s 失败\n", iptvCategoryList.Name), Type: "danger"}
		}
	}
}

func DelList(params url.Values) dto.ReturnJsonDto {
	listId := params.Get("dellist")
	if listId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表id", Type: "danger"}
	}
	var iptvCategoryList models.IptvCategoryList
	res := dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", listId).First(&iptvCategoryList)

	if res.RowsAffected == 0 {
		return dto.ReturnJsonDto{Code: 0, Msg: "频道列表不存在", Type: "danger"}
	}

	dao.DB.Where("id = ?", iptvCategoryList.ID).Delete(&models.IptvCategoryList{})
	var ids []int64
	dao.DB.Model(&models.IptvCategory{}).
		Where("list_id = ?", iptvCategoryList.ID).
		Pluck("id", &ids)
	for _, id := range ids {
		go until.RemoveCaFromEpg(id)
	}
	dao.DB.Where("list_id = ?", iptvCategoryList.ID).Delete(&models.IptvCategory{})
	dao.DB.Where("list_id = ?", iptvCategoryList.ID).Delete(&models.IptvChannel{})
	go until.CleanMealsTxtCacheAll() // 删除缓存
	return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("删除列表 %s 成功\n", iptvCategoryList.Name), Type: "success"}
}

func DelCa(params url.Values) dto.ReturnJsonDto {
	caId := params.Get("delca")
	if caId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}

	var category models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id = ?", caId).First(&category).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "该频道不存在", Type: "danger"}
	}

	dao.DB.Model(&models.IptvCategory{}).Where("id = ?", category.ID).Delete(&models.IptvCategory{})
	dao.DB.Model(&models.IptvChannel{}).Where("c_id = ?", category.ID).Delete(&models.IptvChannel{})
	go until.RemoveCaFromEpg(category.ID)
	go until.CleanAutoCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("删除频道 %s 成功\n", category.Name), Type: "success"}
}

func SubmitMoveUp(params url.Values) dto.ReturnJsonDto {
	id := params.Get("moveup")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}
	var current, prev models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id = ?", id).First(&current).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到当前记录", Type: "danger"}
	}
	if err := dao.DB.Model(&models.IptvCategory{}).
		Where("sort < ?", current.Sort).
		Order("sort DESC").
		First(&prev).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到可交换的记录", Type: "danger"}
	}

	if prev.Sort < 0 {
		return dto.ReturnJsonDto{Code: 0, Msg: "已在自定义分类最上", Type: "danger"}
	}

	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		// 交换 sort
		if err := tx.Model(&models.IptvCategory{}).
			Where("id = ?", current.ID).
			Update("sort", prev.Sort).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.IptvCategory{}).
			Where("id = ?", prev.ID).
			Update("sort", current.Sort).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "交换排序失败", Type: "danger"}
	} else {
		go until.CleanMealsTxtCacheAll()
		return dto.ReturnJsonDto{Code: 1, Msg: "交换排序成功", Type: "success"}
	}
}

func SubmitMoveDown(params url.Values) dto.ReturnJsonDto {
	id := params.Get("movedown")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}

	var current, next models.IptvCategory

	// 获取当前记录
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id = ?", id).First(&current).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到当前记录", Type: "danger"}
	}

	// 获取下一条记录（sort 大于当前记录）
	if err := dao.DB.Model(&models.IptvCategory{}).
		Where("sort > ?", current.Sort).
		Order("sort ASC").
		First(&next).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到可交换的记录", Type: "danger"}
	}

	// 交换 sort
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.IptvCategory{}).
			Where("id = ?", current.ID).
			Update("sort", next.Sort).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.IptvCategory{}).
			Where("id = ?", next.ID).
			Update("sort", current.Sort).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "交换排序失败", Type: "danger"}
	}

	go until.CleanMealsTxtCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "交换排序成功", Type: "success"}
}

func SubmitMoveTop(params url.Values) dto.ReturnJsonDto {
	id := params.Get("movetop")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}

	var current models.IptvCategory
	if err := dao.DB.Where("id = ?", id).First(&current).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到当前记录", Type: "danger"}
	}

	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		// 将所有记录的 sort 增加 1（为当前记录腾出最上位置）
		if err := tx.Model(&models.IptvCategory{}).
			Where("id != ?", current.ID).
			Update("sort", gorm.Expr("sort + 1")).Error; err != nil {
			return err
		}

		// 将当前记录的 sort 设置为 1（最上）
		if err := tx.Model(&models.IptvCategory{}).
			Where("id = ?", current.ID).
			Update("sort", 1).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "移动到最上失败", Type: "danger"}
	}
	go until.CleanMealsTxtCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "已移动到最上", Type: "success"}
}

func SubmitSave(params url.Values) dto.ReturnJsonDto {
	srclistStr := params.Get("srclist")
	categoryId := params.Get("caId")

	if categoryId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误", Type: "danger"}
	}

	// srcList := strings.Split(srclistStr, "\n")

	var category models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id = ?", categoryId).First(&category).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到当前记录", Type: "danger"}
	}
	if strings.Contains(category.Type, "auto") {
		return dto.ReturnJsonDto{Code: 0, Msg: "聚合分类不允许修改", Type: "danger"}
	}

	if category.Sort < 0 {
		return dto.ReturnJsonDto{Code: 0, Msg: "默认分类不允许修改", Type: "danger"}
	}

	dao.DB.Model(&models.IptvCategory{}).Where("id = ?", category.ID).Updates(map[string]interface{}{
		"type": "user",
	})
	until.AddChannelList(srclistStr, category.ID, category.ListId, false)

	return dto.ReturnJsonDto{Code: 1, Msg: "保存成功", Type: "success"}
}

func SaveChannelsOne(params url.Values) dto.ReturnJsonDto {
	chId := params.Get("chId")
	chname := params.Get("chname")
	chURL := params.Get("chURL")
	e_id := params.Get("e_id")

	if chId == "" || chname == "" || chURL == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误, 不得为空", Type: "danger"}
	}

	if !until.IsSafe(chId) || !until.IsSafe(e_id) {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误, 存在非法字符", Type: "danger"}
	}

	var channel models.IptvChannel
	if err := dao.DB.Model(&models.IptvChannel{}).Where("id = ?", chId).First(&channel).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "未找到对应的频道记录", Type: "danger"}
	}

	if e_id != "" {
		var epg models.IptvEpg
		if err := dao.DB.Model(&models.IptvEpg{}).Where("id = ?", e_id).First(&epg).Error; err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "未找到对应的 EPG 记录", Type: "danger"}
		}
		channel.EId = epg.ID
		var tmpList []string
		tmpList = append(tmpList, channel.Name)
		epg.Content = strings.Join(until.MergeAndUnique(strings.Split(epg.Content, ","), tmpList), ",")

		if err := dao.DB.Model(&models.IptvEpg{}).Where("id = ?", e_id).Save(&epg).Error; err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "保存EPG失败" + err.Error(), Type: "danger"}
		}
	} else {
		channel.EId = 0
	}

	channel.Name = chname
	channel.Url = chURL

	if err := dao.DB.Model(&models.IptvChannel{}).Where("id = ?", chId).Updates(map[string]interface{}{
		"name": channel.Name,
		"url":  channel.Url,
		"e_id": channel.EId,
	}).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "保存频道失败" + err.Error(), Type: "danger"}
	}

	go until.CleanAutoCacheAll() // 清理缓存
	return dto.ReturnJsonDto{Code: 1, Msg: "保存成功", Type: "success"}
}

func GenreChannels(srclist string, caList models.IptvCategoryList, doRepeat, group bool) dto.ReturnJsonDto {

	data := until.ConvertDataToMap(srclist, group)
	var repeatCount int
	for genreName, genreList := range data {
		genreName = strings.TrimSpace(genreName)
		if genreName == "" {
			continue
		}

		categoryName := strings.ReplaceAll(fmt.Sprintf("%s(%s)", genreName, caList.Name), " ", "")

		var category models.IptvCategory
		dao.DB.Model(&models.IptvCategory{}).Where("name = ?", categoryName).First(&category)

		if category.ID == 0 {
			var maxSort int64
			dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)
			category := models.IptvCategory{
				Name:   categoryName,
				Sort:   maxSort + 1,
				Type:   "add",
				ListId: caList.ID,
				UA:     caList.UA,
				ReName: caList.ReName,
			}
			if caList.Ku9 == 1 {
				category.Ku9 = genreList.Ku9
			}

			if err := dao.DB.Create(&category).Error; err != nil {
				return dto.ReturnJsonDto{Code: 0, Msg: fmt.Sprintf("新增分类 %s 失败\n", categoryName), Type: "danger"}
			}
			go until.SyncCaToEpg(category.ID)
			a, err := until.AddChannelList(genreList.SrcList, category.ID, caList.ID, doRepeat)
			if err != nil {
				log.Println(fmt.Sprintf("新增分类 %s 失败\n", categoryName), err)
				continue
			}
			repeatCount += a
			continue
		}
		a, err := until.AddChannelList(genreList.SrcList, category.ID, caList.ID, doRepeat)
		if err != nil {
			log.Println(fmt.Sprintf("新增分类 %s 失败\n", categoryName), err)
			continue
		}
		repeatCount += a
	}
	if repeatCount > 0 {
		if !doRepeat {
			return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("更新列表 %s 成功，重复 %d 条\n", caList.Name, repeatCount), Type: "success"}
		}
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "更新列表", Type: "success"}
}

func CategoryListChangeStatus(params url.Values) dto.ReturnJsonDto {
	listId := params.Get("categoryListStatus")
	if listId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "源 id 不能为空", Type: "danger"}
	}

	var cateData models.IptvCategoryList
	if err := dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", listId).First(&cateData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询Category源失败", Type: "danger"}
	}

	if cateData.Enable == 1 {
		dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", cateData.ID).Update("enable", 0)
		dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", cateData.ID).Update("enable", 0)
	} else {
		dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", cateData.ID).Update("enable", 1)
		dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", cateData.ID).Update("enable", 1)
	}
	go until.CleanAutoCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "源 " + cateData.Name + "状态修改成功", Type: "success"}
}

func CategoryChangeStatus(params url.Values) dto.ReturnJsonDto {
	caId := params.Get("categoryStatus")
	if caId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "源 id 不能为空", Type: "danger"}
	}

	var cateData models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id = ?", caId).First(&cateData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询Category失败", Type: "danger"}
	}

	if cateData.Enable == 1 {
		dao.DB.Model(&models.IptvCategory{}).Where("id = ?", cateData.ID).Update("enable", 0)
		dao.DB.Model(&models.IptvChannel{}).Where("c_id = ?", cateData.ID).Update("status", 0)
	} else {
		dao.DB.Model(&models.IptvCategory{}).Where("id = ?", cateData.ID).Update("enable", 1)
		dao.DB.Model(&models.IptvChannel{}).Where("c_id = ?", cateData.ID).Update("status", 1)
	}
	go until.CleanAutoCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "分类 " + cateData.Name + "状态修改成功", Type: "success"}
}

func ChannelsChangeStatus(params url.Values) dto.ReturnJsonDto {
	chId := params.Get("channelsStatus")
	if chId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "频道 id 不能为空", Type: "danger"}
	}

	var chData models.IptvChannel
	if err := dao.DB.Model(&models.IptvChannel{}).Where("id = ?", chId).First(&chData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询频道失败", Type: "danger"}
	}

	if chData.Status == 1 {
		dao.DB.Model(&models.IptvChannel{}).Where("id = ?", chData.ID).Update("status", 0)
	} else {
		dao.DB.Model(&models.IptvChannel{}).Where("id = ?", chData.ID).Update("status", 1)
	}
	go until.CleanAutoCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "频道 " + chData.Name + "状态修改成功", Type: "success"}
}

func UpdateListAll() dto.ReturnJsonDto {
	if crontab.UpdateStatus {
		return dto.ReturnJsonDto{Code: 0, Msg: "后台更新中", Type: "danger"}
	}

	crontab.UpdateStatus = true
	defer func() { crontab.UpdateStatus = false }()

	go crontab.UpdateList() // 更新所有频道列表
	return dto.ReturnJsonDto{Code: 1, Msg: "开始后台更新", Type: "success"}
}

func UploadPayList(c *gin.Context) dto.ReturnJsonDto {
	file, err := c.FormFile("paylistfile")
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取文件失败:" + err.Error(), Type: "danger"}
	}

	listName := "文件导入" + time.Now().Format("20060102")

	f, err := file.Open()
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "打开文件失败: " + err.Error(), Type: "danger"}
	}
	defer f.Close()

	// 读取内容
	data, err := io.ReadAll(f)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "读取文件失败: " + err.Error(), Type: "danger"}
	}

	// 转为字符串
	urlData := until.FilterEmoji(string(data)) // 过滤emoji表情

	if until.IsM3UContent(urlData) {
		urlData = until.M3UToGenreTXT(urlData)
	}

	if !strings.Contains(urlData, "#genre#") {
		var maxSort int64
		dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)
		var new = models.IptvCategory{Name: listName, Type: "file", Sort: maxSort + 1, ReName: 1}
		dao.DB.Model(&models.IptvCategory{}).Create(&new)
		go until.SyncCaToEpg(new.ID) // 异步同步到epg

		repeat, err := until.AddChannelList(urlData, new.ID, 0, false)
		if err == nil {
			return dto.ReturnJsonDto{Code: 1, Msg: fmt.Sprintf("更新列表 %s 成功，重复 %d 条\n", listName, repeat), Type: "success"}
		} else {
			return dto.ReturnJsonDto{Code: 0, Msg: fmt.Sprintf("更新列表 %s 失败\n", listName), Type: "danger"}
		}
	}
	caList := models.IptvCategoryList{ID: 0, Name: listName, UA: "", ReName: 1}
	return GenreChannels(urlData, caList, false, true)
}

func SaveCategory(params url.Values) dto.ReturnJsonDto {
	caId := params.Get("caId")
	caname := params.Get("caname")
	caua := params.Get("caua")
	autoType := params.Get("autoType")
	rulesRe := params.Get("rulesRe")
	ruleEpgs := params.Get("ruleEpgs")
	ku9 := params.Get("ku9")
	proxy := params.Get("caproxy")
	rename := params.Get("rename")

	if caname == "" || !until.IsSafe(caname) || !until.IsSafe(proxy) {
		return dto.ReturnJsonDto{Code: 0, Msg: "参数错误或非法参数", Type: "danger"}
	}

	if caId == "" {
		var tmpCa models.IptvCategory
		err := dao.DB.Model(&models.IptvCategory{}).Where("name = ?", caname).First(&tmpCa).Error
		if err == nil {
			// 找到记录 → 重复
			return dto.ReturnJsonDto{Code: 0, Msg: "分类名称重复", Type: "danger"}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// 其他查询错误
			return dto.ReturnJsonDto{Code: 0, Msg: "查询失败：" + err.Error(), Type: "danger"}
		}

		var maxSort int64
		dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)
		var new = models.IptvCategory{Name: caname, Type: "user", Sort: maxSort + 1, UA: caua, Ku9: ku9}

		if proxy == "1" || proxy == "true" || proxy == "on" {
			new.Proxy = 1
		}

		if autoType != "" {
			if dao.Lic.Type == 0 {
				return dto.ReturnJsonDto{Code: 0, Msg: "未授权不支持自动分类", Type: "danger"}
			}
			_, err := until.CheckLicVer("v1.5.10")
			if err != nil {
				return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
			}
			switch autoType {
			case "auto", "autoRe":
				new.Type = "autoRe"
				new.Rules = rulesRe
			case "autoEpgs":
				new.Type = "autoEpgs"
				new.Rules = ruleEpgs
			}
			new.Proxy = 1
		}

		if rename == "1" || rename == "true" || rename == "on" {
			new.ReName = 1
		}
		dao.DB.Model(&models.IptvCategory{}).Create(&new)
		if strings.Contains(new.Type, "auto") {
			go until.CleanAutoCacheAll()
		} else {
			go until.SyncCaToEpg(new.ID)
			go until.CleanMealsTxtCacheAll()
		}
	} else {
		caIdInt, err := strconv.ParseInt(caId, 10, 64)
		if err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: fmt.Sprintf("参数错误或非法参数:%s", err.Error()), Type: "danger"}
		}
		var ca models.IptvCategory
		if err := dao.DB.Where("id = ?", caIdInt).First(&ca).Error; err != nil || errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ReturnJsonDto{Code: 0, Msg: "分类不存在", Type: "danger"}
		}
		ca.Name = caname
		ca.UA = caua
		ca.Ku9 = ku9
		ca.Type = "user"
		ca.Rules = ""

		if proxy == "1" || proxy == "true" || proxy == "on" {
			ca.Proxy = 1
		} else {
			ca.Proxy = 0
		}

		if autoType != "" {
			if dao.Lic.Type == 0 {
				return dto.ReturnJsonDto{Code: 0, Msg: "未授权不支持自动分类", Type: "danger"}
			}
			_, err := until.CheckLicVer("v1.5.10")
			if err != nil {
				return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
			}
			switch autoType {
			case "auto", "autoRe":
				ca.Type = "autoRe"
				ca.Rules = rulesRe
				dao.DB.Model(&models.IptvChannel{}).Delete(&models.IptvChannel{}, "c_id = ?", ca.ID)
			case "autoEpgs":
				ca.Type = "autoEpgs"
				ca.Rules = ruleEpgs
				dao.DB.Model(&models.IptvChannel{}).Delete(&models.IptvChannel{}, "c_id = ?", ca.ID)
			}
			ca.Proxy = 1
		}

		if rename == "1" || rename == "true" || rename == "on" {
			ca.ReName = 1
		} else {
			ca.ReName = 0
		}
		dao.DB.Model(&models.IptvCategory{}).Where("id = ?", caIdInt).Updates(map[string]interface{}{
			"name":   ca.Name,
			"ua":     ca.UA,
			"type":   ca.Type,
			"rules":  ca.Rules,
			"proxy":  ca.Proxy,
			"rename": ca.ReName,
			"ku9":    ca.Ku9,
		})

		proxyCaCheck := "proxyCaCheck_" + strconv.FormatInt(caIdInt, 10)
		dao.Cache.Delete(proxyCaCheck)

		if strings.Contains(ca.Type, "auto") {
			go until.RemoveCaFromEpg(caIdInt)
			go until.CleanAutoCacheAll()
		} else {
			go until.CleanMealsTxtCacheAll()
		}
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "操作成功", Type: "success"}
}

func TestResolutionOne(params url.Values) dto.ReturnJsonDto {
	chId := params.Get("testResolutionOne")
	if dao.Lic.Type == 0 {
		return dto.ReturnJsonDto{Code: 0, Msg: "未授权", Type: "danger"}
	}
	if chId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "频道 id 不能为空", Type: "danger"}
	}

	var chData models.IptvChannel
	if err := dao.DB.Model(&models.IptvChannel{}).Where("id = ?", chId).First(&chData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询频道失败", Type: "danger"}
	}

	res, err := dao.WS.SendWS(dao.Request{Action: "testResolutionOne", Data: chData.ID})
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: res.Msg, Type: "danger"}
	}
	if res.Code == 1 {
		return dto.ReturnJsonDto{Code: 1, Msg: "操作成功", Type: "success"}
	} else {
		return dto.ReturnJsonDto{Code: 0, Msg: res.Msg, Type: "danger"}
	}
}
