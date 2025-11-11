package service

import (
	"encoding/json"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"slices"
	"strings"
	"time"
)

func GetWeather() map[string]interface{} {
	res := make(map[string]interface{})
	res["code"] = 200
	res["msg"] = "请求成功!"
	res["content"] = map[string]interface{}{
		"city":        "北京",
		"date":        "2024-08-01",
		"weather":     "晴",
		"temperature": "30°C",
	}
	return res
}

func GetEpg(name string) dto.ApkResponse {
	var res dto.ApkResponse
	res.Code = 200
	res.Msg = "请求成功!"

	var epg models.IptvEpg
	name = strings.ToUpper(name)
	dao.DB.Model(&models.IptvEpg{}).Where("content like ? or remarks like ?", "%"+name+"%", "%"+name+"%").First(&epg)
	if epg.ID == 0 {
		res.Code = 500
		res.Msg = "未找到相关节目!"
		return res
	}

	fromList := strings.Split(epg.FromListStr, ",")

	if len(fromList) <= 0 {
		res.Code = 500
		res.Msg = "未找到相关节目!"
		return res
	}

	if slices.Contains(fromList, "0") {
		res = getEpgCntv(epg.Name)
		if len(res.Data) <= 0 {
			var epgFromList []models.IptvEpgList
			dao.DB.Where("epg_id in ?", fromList).Find(&epgFromList)
			if len(epgFromList) > 0 {
				return res
			}
			for _, epgFrom := range epgFromList {
				res = getEpgXml(epgFrom.ID, epg.Name)
			}
		}
	}
	return res
}

func GetSimpleEpg(name string) dto.SimpleResponse {
	var res dto.SimpleResponse

	res.Code = 200
	res.Msg = "请求成功!"

	var epg models.IptvEpg
	name = strings.ToUpper(name)
	dao.DB.Model(&models.IptvEpg{}).Where("content like ? or remarks like ?", "%"+name+"%", "%"+name+"%").First(&epg)
	if epg.ID == 0 {
		res.Code = 500
		res.Msg = "未找到相关节目!"
		return res
	}

	fromList := strings.Split(epg.FromListStr, ",")

	if len(fromList) <= 0 {
		res.Code = 500
		res.Msg = "未找到相关节目!"
		return res
	}

	if slices.Contains(fromList, "0") {
		res = getSimpleEpgCntv(epg.Name)
		if res.Data == (dto.Program{}) {
			var epgFromList []models.IptvEpgList
			dao.DB.Where("epg_id in ?", fromList).Find(&epgFromList)
			if len(epgFromList) > 0 {
				return res
			}
			for _, epgFrom := range epgFromList {
				res = getSimpleEpg(epgFrom.ID, epg.Name)
			}
		}
	}
	return res
}

func getEpgCntv(name string) dto.ApkResponse {

	var cacheKey = "cntv_" + name

	var res dto.ApkResponse
	res.Code = 200
	res.Msg = "请求成功!"

	if name == "" {
		res.Data = []dto.Program{}
		return res
	}

	name = strings.ToLower(name)
	epgUrl := "https://api.cntv.cn/epg/epginfo?c=" + name + "&serviceId=channel&d="

	var jsonMap map[string]map[string]interface{}

	readCacheOk := false
	if dao.Cache.Exists(cacheKey) {
		err := dao.Cache.GetJSON(cacheKey, jsonMap)
		if err == nil {
			readCacheOk = true
		}
	}

	if !readCacheOk {
		jsonStr := until.GetUrlData(epgUrl)
		err := json.Unmarshal([]byte(jsonStr), &jsonMap)
		if err != nil {
			res.Data = []dto.Program{}
			return res
		}
		if dao.Cache.SetJSON(cacheKey, jsonMap) != nil {
			dao.Cache.Delete(cacheKey)
		}
	}

	if _, ok := jsonMap["errcode"]; ok {
		res.Data = []dto.Program{}
		return res
	}

	if epgData, ok := jsonMap[name]; ok {
		dataList := []dto.Program{}
		pos := 0

		if len(epgData["program"].([]interface{})) <= 0 {
			res.Data = []dto.Program{}
			return res
		}
		currentTime := time.Now()
		zoneName, _ := currentTime.Zone()
		if zoneName == "UTC" {
			currentTime = currentTime.Add(8 * time.Hour)
		}
		nowTime := currentTime.Format("15:04")
		var a = 0
		for _, item := range epgData["program"].([]interface{}) {
			if dataMap, ok := item.(map[string]interface{}); ok {
				data := dto.Program{}
				data.Name = dataMap["t"].(string)
				data.StartTime = dataMap["showTime"].(string)

				data.Pos = a
				dataList = append(dataList, data)

				if nowTime > data.StartTime {
					pos += 1
				}
				a++

			}
		}
		if pos > 1 {
			pos = pos - 1
		}
		res.Pos = pos
		res.Data = dataList
	} else {
		res.Data = []dto.Program{}
	}

	return res
}

func getSimpleEpgCntv(name string) dto.SimpleResponse {

	cacheKey := "cntv_" + name
	var simpleRes dto.SimpleResponse
	simpleRes.Code = 200
	simpleRes.Msg = "请求成功!"

	if name == "" {
		simpleRes.Data = dto.Program{}
		return simpleRes
	}
	name = strings.ToLower(name)
	epgUrl := "https://api.cntv.cn/epg/epginfo?c=" + name + "&serviceId=channel&d="

	var jsonMap map[string]map[string]interface{}
	readCacheOk := false

	if dao.Cache.Exists(cacheKey) {
		err := dao.Cache.GetJSON(cacheKey, jsonMap)
		if err == nil {
			readCacheOk = true
		}
	}

	if !readCacheOk {
		jsonStr := until.GetUrlData(epgUrl)
		err := json.Unmarshal([]byte(jsonStr), &jsonMap)
		if err != nil {
			simpleRes.Data = dto.Program{}
			return simpleRes
		}
		if dao.Cache.SetJSON(cacheKey, jsonMap) != nil {
			dao.Cache.Delete(cacheKey)
		}
	}

	if _, ok := jsonMap["errcode"]; ok {
		simpleRes.Data = dto.Program{}
		dao.Cache.Delete(cacheKey)
		return simpleRes
	}

	if epgData, ok := jsonMap[name]; ok {
		var simpleRes dto.SimpleResponse
		data := dto.Program{}
		data.Name = epgData["isLive"].(string)
		data.StartTime = time.Unix(int64(epgData["liveSt"].(float64)), 0).Format("15:04")
		simpleRes.Data = data
		return simpleRes
	} else {
		simpleRes.Data = dto.Program{}
	}
	return simpleRes
}

func getEpgXml(epgFromId int64, epgName string) dto.ApkResponse {
	res := dto.ApkResponse{}
	res.Code = 200
	res.Msg = "请求成功!"

	var epgsList models.IptvEpgList
	if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ? and status = 1", epgFromId).First(&epgsList).Error; err != nil {
		return res
	}

	xmlTV := until.GetEpgListXml(epgsList.Name, epgsList.Url)
	if isXmlTVEmpty(xmlTV) {
		return res
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	currentTime := time.Now().In(loc)

	// nowTime := currentTime.Format("15:04")
	const layout = "20060102150405 -0700"
	dataList := make([]dto.Program, 0)
	pos := 0

	for _, channel := range xmlTV.Channels {
		if strings.EqualFold(channel.DisplayName[0].Value, epgName) {
			var a = 0
			for _, programme := range xmlTV.Programmes {
				if programme.Channel == channel.ID {
					tS, _ := time.ParseInLocation(layout, programme.Start, loc)
					// tE, _ := time.Parse(layout, programme.Stop)
					StartTime := tS.Format("15:04")
					// EndTime := tE.Format("15:04")

					data := dto.Program{}
					data.Name = programme.Title.Value
					data.StartTime = StartTime
					data.Pos = a
					dataList = append(dataList, data)
					if currentTime.After(tS) {

						pos += 1
					}
					a++
				}
			}

			if pos > 1 {
				pos = pos - 1
			}
			res.Pos = pos
			res.Data = dataList
			break
		}
	}

	return res
}

func getSimpleEpg(epgFromId int64, epgName string) dto.SimpleResponse {

	res := dto.SimpleResponse{}
	res.Code = 200
	res.Msg = "请求成功!"

	var epgsList models.IptvEpgList
	if err := dao.DB.Model(&models.IptvEpgList{}).Where("id = ? and status = 1", epgFromId).First(&epgsList).Error; err != nil {
		return res
	}

	xmlTV := until.GetEpgListXml(epgsList.Name, epgsList.Url)
	if isXmlTVEmpty(xmlTV) {
		return res
	}
	currentTime := time.Now()
	zoneName, _ := currentTime.Zone()
	if zoneName == "UTC" {
		currentTime = currentTime.Add(8 * time.Hour)
	}
	nowTime := currentTime.Format("15:04")
	const layout = "20060102150405 -0700"

	for _, channel := range xmlTV.Channels {
		if strings.EqualFold(channel.DisplayName[0].Value, epgName) {
			for _, programme := range xmlTV.Programmes {
				if programme.Channel == channel.ID {
					tS, _ := time.Parse(layout, programme.Start)
					tE, _ := time.Parse(layout, programme.Stop)
					StartTime := tS.Format("15:04")
					EndTime := tE.Format("15:04")

					data := dto.Program{}
					data.Name = programme.Title.Value
					data.StartTime = StartTime

					if nowTime < EndTime {
						res.Data = data
						return res
					} else {
						continue
					}
				}
			}
			break
		}
	}

	res.Data = dto.Program{}
	return res
}

func isXmlTVEmpty(tv dto.XmlTV) bool {
	return len(tv.Channels) == 0 || len(tv.Programmes) == 0
}
