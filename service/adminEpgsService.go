package service

import (
	"errors"
	"fmt"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetChName(params url.Values) dto.ReturnJsonDto {
	//编辑
	epgId := params.Get("bdingepg")
	if epgId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG id不能为空", Type: "danger"}
	}

	var epg models.IptvEpg
	if err := dao.DB.Where("id = ?", epgId).First(&epg).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询EPG 失败", Type: "danger"}
	}

	caList := strings.Split(epg.CasStr, ",")

	var channeList []models.IptvChannel
	if err := dao.DB.Model(&models.IptvChannel{}).Select("distinct name").Where("c_id in ? and status = 1", caList).Order("c_id,id").Find(&channeList).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询频道失败", Type: "danger"}
	}

	CheckList := until.MergeAndUnique(strings.Split(epg.Content, ","), strings.Split(epg.Remarks, "|"))

	var dataList []dto.EpgsReturnDto

	for _, v := range channeList {
		var data dto.EpgsReturnDto
		data.Name = v.Name
		data.Select = false
		for _, v1 := range CheckList {
			if strings.EqualFold(v1, v.Name) {
				data.Select = true
			}
		}
		if strings.EqualFold(epg.Name, v.Name) {
			data.Select = true
		}
		dataList = append(dataList, data)
	}

	return dto.ReturnJsonDto{Code: 1, Msg: "操作成功", Type: "success", Data: dataList}
}

func GetCa(params url.Values) dto.ReturnJsonDto {
	epgId := params.Get("epgGetCa")
	if epgId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG id不能为空", Type: "danger"}
	}

	var epg models.IptvEpg
	if err := dao.DB.Where("id = ?", epgId).First(&epg).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询EPG 失败", Type: "danger"}
	}

	var caList []models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("enable = 1 and type not like ?", "auto%").Find(&caList).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询分组失败", Type: "danger"}
	}

	caIdList := strings.Split(epg.CasStr, ",")
	var dataList []dto.EpgsReturnDto
	for _, v := range caList {
		var data dto.EpgsReturnDto
		data.Id = v.ID
		data.Name = v.Name
		data.Select = false
		for _, v1 := range caIdList {
			if strings.EqualFold(v1, fmt.Sprintf("%d", v.ID)) {
				data.Select = true
			}
		}
		dataList = append(dataList, data)
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "操作成功", Type: "success", Data: dataList}
}

func SaveEpg(params url.Values) dto.ReturnJsonDto {
	name := params.Get("name")
	if name == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG 名称不能为空", Type: "danger"}
	}

	var epgData models.IptvEpg
	id := params.Get("epgId")
	if id != "" {
		if err := dao.DB.First(&epgData, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.ReturnJsonDto{Code: 0, Msg: "EPG记录不存在", Type: "danger"}
			}
			return dto.ReturnJsonDto{Code: 0, Msg: "查询EPG失败", Type: "danger"}
		}
	}

	epgData.Name = name
	epgData.Remarks = params.Get("epgRemarks")

	epgData.CasStr = params.Get("caList")
	epgData.FromListStr = params.Get("fromList")

	if err := dao.DB.Save(&epgData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "保存EPG失败", Type: "danger"}
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "EPG " + epgData.Name + "保存成功", Type: "success"}
}

func BdingEpg(params url.Values) dto.ReturnJsonDto {
	id := params.Get("epgId")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG id不能为空", Type: "danger"}
	}

	namesList := params["names[]"]

	var epgData models.IptvEpg

	if err := dao.DB.First(&epgData, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ReturnJsonDto{Code: 0, Msg: "EPG记录不存在", Type: "danger"}
		}
		return dto.ReturnJsonDto{Code: 0, Msg: "查询EPG失败", Type: "danger"}
	}
	epgData.Content = strings.Join(namesList, ",")

	if err := dao.DB.Save(&epgData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "保存EPG失败", Type: "danger"}
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "EPG " + epgData.Name + "保存成功", Type: "success"}
}

func ChangeStatus(params url.Values) dto.ReturnJsonDto {
	id := params.Get("change_status")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG id不能为空", Type: "danger"}
	}

	var epgData models.IptvEpg
	if err := dao.DB.Model(&models.IptvEpg{}).Where("id = ?", id).First(&epgData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询EPG失败", Type: "danger"}
	}

	if epgData.Status == 1 {
		dao.DB.Model(&models.IptvEpg{}).Where("id = ?", id).Update("status", 0)
	} else {
		dao.DB.Model(&models.IptvEpg{}).Where("id = ?", id).Update("status", 1)
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "EPG " + epgData.Name + "状态修改成功", Type: "success"}
}

func ChangeListStatus(params url.Values) dto.ReturnJsonDto {
	id := params.Get("change_status")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG 列表不能为空", Type: "danger"}
	}

	var epgData models.IptvEpgList
	if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", id).First(&epgData).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "查询EPG失败", Type: "danger"}
	}

	if epgData.Status == 1 {
		dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", id).Update("status", 0)
		dao.DB.Model(&models.IptvEpg{}).Where("name like ?", epgData.Remarks+"-%").Update("status", 0)
	} else {
		dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", id).Update("status", 1)
		dao.DB.Model(&models.IptvEpg{}).Where("name like ?", epgData.Remarks+"-%").Update("status", 1)
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "EPG 列表 " + epgData.Name + "状态修改成功", Type: "success"}
}

func DeleteEpg(params url.Values) dto.ReturnJsonDto {
	id := params.Get("delepg")
	if id == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG id不能为空", Type: "danger"}
	}
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG id为数字", Type: "danger"}
	}
	if idInt64 <= 18 {
		return dto.ReturnJsonDto{Code: 0, Msg: "CNTV EPG不能删除", Type: "danger"}
	}
	dao.DB.Where("id = ?", id).Delete(&models.IptvEpg{})
	return dto.ReturnJsonDto{Code: 1, Msg: "EPG删除成功", Type: "success"}
}

func BindChannel() dto.ReturnJsonDto {
	// ClearBind() // 清空绑定
	until.BindChannel() // 绑定频道

	return dto.ReturnJsonDto{Code: 1, Msg: "绑定成功", Type: "success"}
}

func ClearBind() dto.ReturnJsonDto {
	dao.DB.Model(&models.IptvEpg{}).Where("content != ''").Update("content", "")
	until.BindChannel() // 绑定频道
	return dto.ReturnJsonDto{Code: 1, Msg: "清除绑定成功", Type: "success"}
}

func ClearCache() dto.ReturnJsonDto {
	dao.Cache.Clear()
	until.CleanMealsXmlCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "清除缓存成功", Type: "success"}
}

func EpgImport(params url.Values) dto.ReturnJsonDto {
	listName := params.Get("epgfromname")
	url := strings.TrimSpace(params.Get("epgfromurl"))
	ua := params.Get("epgfromua")
	eId := params.Get("eid")

	if listName == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表", Type: "danger"}
	}

	if !until.IsSafe(listName) || !until.IsSafe(eId) {
		return dto.ReturnJsonDto{Code: 0, Msg: "输入不合法", Type: "danger"}
	}

	remarks := until.GetMainDomain(url)
	if remarks == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入正确的频道列表地址", Type: "danger"}
	}
	var eOld models.IptvEpgList
	dao.DB.Model(&models.IptvEpgList{}).Where("url = ?", url).First(&eOld)
	if eOld.ID != 0 && eId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "该频道列表已存在", Type: "danger"}
	}

	iptvEpgList := models.IptvEpgList{Name: listName, Url: url, Status: 1, Remarks: remarks, UA: ua}
	if eId != "" {
		if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", eId).First(&eOld).Error; err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "频道列表不存在", Type: "danger"}
		}
		iptvEpgList.ID = eOld.ID
		if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", eId).Updates(&iptvEpgList).Error; err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "更新失败", Type: "danger"}
		}
		do, err := until.UpdataEpgListOne(iptvEpgList, true)
		if do {
			return dto.ReturnJsonDto{Code: 1, Msg: "更新成功", Type: "success"}
		}
		return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
	} else {
		if err := dao.DB.Model(&models.IptvEpgList{}).Create(&iptvEpgList).Error; err != nil {
			return dto.ReturnJsonDto{Code: 0, Msg: "添加失败", Type: "danger"}
		}
		do, err := until.UpdataEpgListOne(iptvEpgList, true)
		if do {
			return dto.ReturnJsonDto{Code: 1, Msg: "添加成功", Type: "success"}
		}
		return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
	}
}

func UploadLogo(c *gin.Context) dto.ReturnJsonDto {

	epgFromName := c.PostForm("epgname")
	if epgFromName == "" || !until.IsSafe(epgFromName) {
		return dto.ReturnJsonDto{Code: 0, Msg: "EPG名称不合法", Type: "danger"}
	}

	file, err := c.FormFile("uploadlogo")
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取文件失败:" + err.Error(), Type: "danger"}
	}

	f, err := file.Open()
	if err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "打开文件失败:" + err.Error(), Type: "danger"}
	}
	defer f.Close()

	// 读取前 512 字节判断 MIME 类型
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	if contentType != "image/png" {
		return dto.ReturnJsonDto{Code: 0, Msg: "只允许上传 PNG 文件", Type: "danger"}
	}

	dst := "/config/logo/" + epgFromName + ".png"
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "保存文件失败:" + err.Error(), Type: "danger"}
	}
	go until.CleanMealsXmlCacheAll()
	return dto.ReturnJsonDto{Code: 1, Msg: "上传成功", Type: "success"}
}

func UpdateEpgList(params url.Values) dto.ReturnJsonDto {
	listId := params.Get("updatelist")
	if listId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表", Type: "danger"}
	}
	var epgList models.IptvEpgList
	if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", listId).First(&epgList).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败:" + err.Error(), Type: "danger"}
	}
	do, err := until.UpdataEpgListOne(epgList, false)
	if do {
		return dto.ReturnJsonDto{Code: 1, Msg: "更新成功", Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 0, Msg: err.Error(), Type: "danger"}
}

func UpdateEpgListAll() dto.ReturnJsonDto {
	if until.UpdataEpgList() {
		return dto.ReturnJsonDto{Code: 1, Msg: "更新成功", Type: "success"}
	}
	return dto.ReturnJsonDto{Code: 0, Msg: "更新失败", Type: "danger"}
}

func DelEpgList(params url.Values) dto.ReturnJsonDto {
	listId := params.Get("dellist")
	if listId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入频道列表", Type: "danger"}
	}
	var epgList models.IptvEpgList
	if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", listId).First(&epgList).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败:" + err.Error(), Type: "danger"}
	}
	if err := dao.DB.Where("id = ?", listId).Delete(&models.IptvEpgList{}).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "删除列表失败:" + err.Error(), Type: "danger"}
	}

	var epgs []models.IptvEpg
	dao.DB.Model(&models.IptvEpg{}).Where("fromlist like ?", "%"+fmt.Sprintf("%d", epgList.ID)+"%").Find(&epgs)
	for _, epg := range epgs {
		fromList := strings.Split(epg.FromListStr, ",")
		for i, v := range fromList {
			if v == fmt.Sprintf("%d", epgList.ID) {
				fromList = append(fromList[:i], fromList[i+1:]...)
				break // 若只删除第一个匹配项
			}
		}
		dao.DB.Model(&models.IptvEpg{}).Where("id = ?", epg.ID).Update("fromlist", strings.Join(fromList, ","))
	}
	// if err := dao.DB.Where("name like ?", epgList.Remarks+"-%").Delete(&models.IptvEpg{}).Error; err != nil {
	// 	return dto.ReturnJsonDto{Code: 0, Msg: "删除EPG失败:" + err.Error(), Type: "danger"}
	// }
	return dto.ReturnJsonDto{Code: 1, Msg: "删除成功", Type: "success"}
}

func DeleteLogo(params url.Values) dto.ReturnJsonDto {
	bjId := params.Get("deleteLogo")
	if bjId == "" {
		return dto.ReturnJsonDto{Code: 0, Msg: "请输入epg ID", Type: "danger"}
	}
	var epg models.IptvEpg
	if err := dao.DB.Model(&models.IptvEpg{}).Where("id = ?", bjId).First(&epg).Error; err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "获取频道列表失败:" + err.Error(), Type: "danger"}
	}
	logoFile := "/config/logo/" + epg.Name + ".png"
	if err := os.Remove(logoFile); err != nil {
		return dto.ReturnJsonDto{Code: 0, Msg: "删除失败", Type: "danger"}
	}
	return dto.ReturnJsonDto{Code: 1, Msg: "删除成功", Type: "success"}
}

func DelNotFrom() dto.ReturnJsonDto {
	dao.DB.Where("id > 18 and (fromlist is null or fromlist = '' or fromlist = ' ')").Delete(&models.IptvEpg{})
	return dto.ReturnJsonDto{Code: 1, Msg: "删除成功", Type: "success"}
}

// func SaveEpgApi(params url.Values) dto.ReturnJsonDto {
// 	err1000 := params.Get("tipepgerror_1000")
// 	err1001 := params.Get("tipepgerror_1001")
// 	err1002 := params.Get("tipepgerror_1002")
// 	err1003 := params.Get("tipepgerror_1003")
// 	err1004 := params.Get("tipepgerror_1004")
// 	err1005 := params.Get("tipepgerror_1005")
// 	epgapiChk := params.Get("epgapi_chk")

// 	if err1000 == "" && err1001 == "" && err1002 == "" && err1003 == "" && err1004 == "" && err1005 == "" {
// 		return dto.ReturnJsonDto{Code: 0, Msg: "错误提示存在空", Type: "error"}
// 	}

// 	cfg := dao.GetConfig()

// 	cfg.EPGErrors.Err1000 = err1000
// 	cfg.EPGErrors.Err1001 = err1001
// 	cfg.EPGErrors.Err1002 = err1002
// 	cfg.EPGErrors.Err1003 = err1003
// 	cfg.EPGErrors.Err1004 = err1004
// 	cfg.EPGErrors.Err1005 = err1005

// 	if epgapiChk == "on" {
// 		cfg.App.EPGApiChk = 1
// 	} else {
// 		cfg.App.EPGApiChk = 0
// 	}

// 	dao.SetConfig(cfg)

// 	return dto.ReturnJsonDto{Code: 1, Msg: "保存EPG成功", Type: "success"}
// }
