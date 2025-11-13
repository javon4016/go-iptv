package until

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

func ConvertCntvToXml(cntv dto.CntvJsonChannel, eName string) dto.XmlTV {
	tv := dto.XmlTV{
		GeneratorName: "清和IPTV管理系统",
		GeneratorURL:  "https://www.qingh.xyz",
	}

	// 添加频道
	tv.Channels = append(tv.Channels, dto.XmlChannel{
		ID: eName,
		DisplayName: []dto.DisplayName{
			{Lang: "zh",
				Value: eName,
			},
		},
	})

	// 添加节目表
	for _, p := range cntv.Program {
		start := time.Unix(p.StartTime, 0).UTC().Format("20060102150405 -0700")
		stop := time.Unix(p.EndTime, 0).UTC().Format("20060102150405 -0700")

		tv.Programmes = append(tv.Programmes, dto.Programme{
			Start:   start,
			Stop:    stop,
			Channel: eName,
			Title: dto.Title{
				Lang:  "zh",
				Value: p.Title,
			},
			Desc: dto.Desc{
				Lang:  "zh",
				Value: p.Title,
			},
		})
	}

	return tv
}

func GetEpgListXml(name, url string) dto.XmlTV {
	epgUrl := url
	cacheKey := "epgXmlFrom_" + name
	var xmlTV dto.XmlTV
	var xmlByte []byte
	readCacheOk := false
	if dao.Cache.Exists(cacheKey) {
		tmpByte, err := dao.Cache.Get(cacheKey)
		if err == nil {
			xmlByte = tmpByte
			readCacheOk = true
		}
	}

	if !readCacheOk {
		xmlByte = []byte(GetUrlData(epgUrl))
		if dao.Cache.Set(cacheKey, xmlByte) != nil {
			dao.Cache.Delete(cacheKey)
		}
	}
	xml.Unmarshal(xmlByte, &xmlTV)
	return xmlTV
}

func GetEpgCntv(name string) (dto.CntvJsonChannel, error) {

	var cacheKey = "cntv_" + strings.ToUpper(name)

	var cntvJson dto.CntvData

	if name == "" {
		return dto.CntvJsonChannel{}, errors.New("id is empty")
	}
	name = strings.ToLower(name)

	epgUrl := "https://api.cntv.cn/epg/epginfo?c=" + name + "&serviceId=channel&d="

	readCacheOk := false
	if dao.Cache.Exists(cacheKey) {
		err := dao.Cache.GetJSON(cacheKey, cntvJson)
		if err == nil {
			readCacheOk = true
		}
	}

	if !readCacheOk {
		jsonStr := GetUrlData(epgUrl)
		err := json.Unmarshal([]byte(jsonStr), &cntvJson)
		if err != nil {
			return dto.CntvJsonChannel{}, err
		}
		if dao.Cache.SetJSON(cacheKey, cntvJson) != nil {
			dao.Cache.Delete(cacheKey)
		}
	}
	return cntvJson[name], nil
}

func UpdataEpgList() bool {
	var epgLists []models.IptvEpgList
	dao.DB.Model(&models.IptvEpgList{}).Find(&epgLists)
	for _, list := range epgLists {
		cacheKey := "epgXmlFrom_" + list.Name
		dao.Cache.Delete(cacheKey)
		xmlStr := GetUrlData(strings.TrimSpace(list.Url), list.UA)
		if xmlStr != "" {
			xmlByte := []byte(xmlStr)
			if dao.Cache.Set(cacheKey, xmlByte) != nil {
				dao.Cache.Delete(cacheKey)
			}
			var xmlTV dto.XmlTV
			if xml.Unmarshal(xmlByte, &xmlTV) != nil {
				continue
			}
			var epgs []models.IptvEpg
			// 1️⃣ 匹配数字台，如 CCTV1、CCTV-5+、CCTV13 等
			reNum := regexp.MustCompile(`(?i)CCTV-?(\d+\+?)$`)

			// 2️⃣ 匹配字母台，如 CCTV4EUO、CCTV4AME、CCTVF、CCTVE 等
			reAlpha := regexp.MustCompile(`(?i)CCTV(\d*[A-Z]+)`)
			for _, channel := range xmlTV.Channels {
				remarks := channel.DisplayName[0].Value
				upper := strings.ToUpper(remarks)
				if strings.Contains(upper, "CCTV") {
					switch {
					case reNum.MatchString(upper):
						match := reNum.FindStringSubmatch(upper)
						num := match[1]
						remarks = fmt.Sprintf("CCTV%s|CCTV-%s|CCTV%s 4K|CCTV-%s 4K|CCTV%s HD|CCTV-%s HD", num, num, num, num, num, num)

					case reAlpha.MatchString(upper):
						match := reAlpha.FindStringSubmatch(upper)
						suffix := match[1]
						remarks = fmt.Sprintf("CCTV%s|CCTV-%s", suffix, suffix)
					}
				} else {
					remarks = fmt.Sprintf("%s|%s 4K|%s HD", remarks, remarks, remarks)
				}
				epgs = append(epgs, models.IptvEpg{
					Name:    channel.DisplayName[0].Value,
					Status:  1,
					Remarks: remarks,
				})
			}
			if len(epgs) > 0 {
				dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", list.ID).Updates(&models.IptvEpgList{Status: 1, LastTime: time.Now().Unix()})
				// dao.DB.Model(&models.IptvEpg{}).Where("name like ?", list.Remarks+"-%").Delete(&models.IptvEpg{})
				// dao.DB.Model(&models.IptvEpg{}).Create(&epgs)
				reload, _ := SyncEpgs(list.ID, epgs, false) // 同步
				if reload {
					go BindChannel() // 绑定频道
				}
				// CleanMealsXmlCacheAll() // 清除缓存
			}
		}
	}
	return true
}

func UpdataEpgListOne(list models.IptvEpgList, newAdd bool) (bool, error) {
	cacheKey := "epgXmlFrom_" + list.Name
	dao.Cache.Delete(cacheKey)
	xmlStr := GetUrlData(strings.TrimSpace(list.Url), list.UA)
	if xmlStr != "" {
		xmlByte := []byte(xmlStr)
		if dao.Cache.Set(cacheKey, xmlByte) != nil {
			dao.Cache.Delete(cacheKey)
		}
		var xmlTV dto.XmlTV
		if xml.Unmarshal(xmlByte, &xmlTV) != nil {
			return false, errors.New("xml解析失败")
		}
		var epgs []models.IptvEpg
		// 1️⃣ 匹配数字台，如 CCTV1、CCTV-5+、CCTV13 等
		reNum := regexp.MustCompile(`(?i)CCTV-?(\d+\+?)$`)

		// 2️⃣ 匹配字母台，如 CCTV4EUO、CCTV4AME、CCTVF、CCTVE 等
		reAlpha := regexp.MustCompile(`(?i)CCTV(\d*[A-Z]+)`)
		for _, channel := range xmlTV.Channels {
			remarks := channel.DisplayName[0].Value
			if remarks == "" {
				continue
			}
			upper := strings.ToUpper(remarks)
			if strings.Contains(upper, "CCTV") {
				switch {
				case reNum.MatchString(upper):
					match := reNum.FindStringSubmatch(upper)
					num := match[1]
					remarks = fmt.Sprintf("CCTV%s|CCTV-%s|CCTV%s 4K|CCTV-%s 4K|CCTV%s HD|CCTV-%s HD", num, num, num, num, num, num)

				case reAlpha.MatchString(upper):
					match := reAlpha.FindStringSubmatch(upper)
					suffix := match[1]
					remarks = fmt.Sprintf("CCTV%s|CCTV-%s", suffix, suffix)
				}
			} else {
				remarks = fmt.Sprintf("%s|%s 4K|%s HD", remarks, remarks, remarks)
			}

			epgs = append(epgs, models.IptvEpg{
				Name:    channel.DisplayName[0].Value,
				Status:  1,
				Remarks: remarks,
			})
		}
		if len(epgs) > 0 {
			dao.DB.Model(&models.IptvEpgList{}).Where("id = ?", list.ID).Updates(&models.IptvEpgList{Status: 1, LastTime: time.Now().Unix()})
			// dao.DB.Model(&models.IptvEpg{}).Where("name like ?", list.Remarks+"-%").Delete(&models.IptvEpg{})
			// dao.DB.Model(&models.IptvEpg{}).Create(&epgs)

			reload, _ := SyncEpgs(list.ID, epgs, newAdd) // 同步
			if reload {
				go BindChannel() // 绑定频道
			}

			return true, nil
		}
		return false, errors.New("未找到epg数据")
	}
	return false, errors.New("URL错误:" + list.Url)
}

func BindChannel() bool {
	// ClearBind() // 清空绑定

	var epgList []models.IptvEpg
	if err := dao.DB.Model(&models.IptvEpg{}).Where("status = 1").Find(&epgList).Error; err != nil {
		return false
	}

	for _, epgData := range epgList {
		caList := strings.Split(epgData.CasStr, ",")
		var tmpList []string
		nameList := strings.Split(epgData.Remarks, "|")
		var channelList []models.IptvChannel
		dao.DB.Model(&models.IptvChannel{}).Select("distinct name").Where("status = 1 and c_id in (?)", caList).Find(&channelList)

		for _, channelData := range channelList {
			if strings.EqualFold(channelData.Name, epgData.Name) {
				tmpList = append(tmpList, channelData.Name)
				continue
			}

			for _, name := range nameList {
				if strings.EqualFold(channelData.Name, name) || channelData.Name == name {
					tmpList = append(tmpList, channelData.Name)
					break
				}
			}
		}
		chNameList := MergeAndUnique(strings.Split(epgData.Content, ","), tmpList)

		if len(tmpList) > 0 {
			dao.DB.Model(&models.IptvChannel{}).Where("name in (?) and c_id in (?) and status = 1", chNameList, caList).Update("e_id", epgData.ID)

			if !EqualStringSets(strings.Split(epgData.Content, ","), chNameList) {
				epgData.Content = strings.Join(chNameList, ",")
				if epgData.Content != "" {
					dao.DB.Save(&epgData)
				}
			}
		}
	}

	go CleanAutoCacheAll() // 清理缓存
	return true
}

// SyncEpgs 同步 IPTV EPG 数据：
// - 保留数据库中已存在的记录（不更新）
// - 新数据中有但数据库没有的 → 新增
// - 数据库中有但新数据中没有的 → 删除
func SyncEpgs(fromId int64, epgs []models.IptvEpg, newAdd bool) (bool, error) {
	// 1. 查询数据库中已有的记录
	var oldEpgs []models.IptvEpg
	if err := dao.DB.Model(&models.IptvEpg{}).Where("status = 1").Find(&oldEpgs).Error; err != nil {
		return false, err
	}

	// 2. 建立 name 映射方便比对
	oldMap := make(map[string]bool)
	newMap := make(map[string]bool)
	for _, o := range oldEpgs {
		oldMap[o.Name] = true
		for i, n := range epgs {
			if o.Name == n.Name {
				epgs[i].ID = o.ID
				epgs[i].FromListStr = o.FromListStr
				epgs[i].Content = o.Content
				epgs[i].Remarks = o.Remarks
				epgs[i].Status = o.Status
				epgs[i].CasStr = o.CasStr
			}
			newMap[n.Name] = true
		}
	}

	// 3. 计算需要新增与删除的数据
	var toAdd []models.IptvEpg

	for _, n := range epgs {
		if !oldMap[n.Name] || newAdd {
			toAdd = append(toAdd, n)
		}
	}

	for _, o := range oldEpgs {
		if !newMap[o.Name] {
			tmpList := strings.Split(o.FromListStr, ",")
			exist := false
			for i, v := range tmpList {
				if v == fmt.Sprintf("%d", fromId) {
					exist = true
					tmpList = append(tmpList[:i], tmpList[i+1:]...)
					break // 若只删除第一个匹配项
				}
			}

			if exist {
				tmpList = RemoveEmptyStrings(tmpList)
				if len(tmpList) > 0 {
					dao.DB.Model(&models.IptvEpg{}).Where("id = ?", o.ID).Update("fromlist", strings.Join(tmpList, ","))
				}
			}
		}
	}
	addCount := 0
	if len(toAdd) > 0 {
		var caIDs []int64
		dao.DB.Model(&models.IptvCategory{}).
			Where("enable = 1 AND type != ?", "auto").
			Pluck("id", &caIDs)

		for _, toAddOne := range toAdd {
			oldList := strings.Split(toAddOne.FromListStr, ",")
			tmpList := append(oldList, fmt.Sprintf("%d", fromId))
			tmpList = RemoveEmptyStrings(tmpList)
			toAddOne.FromListStr = strings.Join(tmpList, ",")

			if EqualStringSets(oldList, tmpList) {
				continue
			}
			if toAddOne.ID == 0 {
				toAddOne.CasStr = strings.Trim(strings.Join(strings.Fields(fmt.Sprint(caIDs)), ","), "[]") // 转换为字符串
			}
			addCount++
			dao.DB.Save(&toAddOne)
		}
		log.Printf("新增 %d 条 EPG 记录\n", addCount)
	}
	if addCount > 0 {
		return true, nil
	}
	return false, errors.New("无新增数据")
}

func GetTxt(id int64) string {
	var res string

	txtCaCheKey := "rssMealTxt_" + strconv.FormatInt(id, 10)
	if dao.Cache.Exists(txtCaCheKey) {
		cacheData, err := dao.Cache.GetNotExpired(txtCaCheKey)
		if err == nil {
			return string(cacheData)
		}
	}

	var meal models.IptvMeals
	if err := dao.DB.Model(&models.IptvMeals{}).Where("id = ? and status = 1", id).First(&meal).Error; err != nil {
		return res
	}
	categoryIdList := strings.Split(meal.Content, ",")
	var categoryList []models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id in (?) and enable = 1", categoryIdList).Order("sort asc").Find(&categoryList).Error; err != nil {
		return res
	}
	cfg := dao.GetConfig()

	for _, category := range categoryList {
		var channels []models.IptvChannelShow
		if category.Type != "auto" {
			dao.DB.Model(&models.IptvChannelShow{}).Where("c_id = ? and status = 1", category.ID).Order("sort asc").Find(&channels)
		} else {
			channels = GetAutoChannelList(category)
		}
		if len(channels) == 0 {
			continue
		}
		res += category.Name + ",#genre#\n"
		for _, channel := range channels {
			if channel.Status == 1 {
				if category.Proxy == 1 && cfg.Proxy.Status == 1 {
					urlMsg := fmt.Sprintf("{\"c\":%d,\"u\":\"%s\"}", category.ID, channel.Url)
					msg, err := UrlEncrypt(dao.Lic.ID, urlMsg)
					if err == nil {
						channel.PUrl = fmt.Sprintf("%s:%d/p/%s", cfg.Proxy.PAddr, cfg.Proxy.Port, msg)
						res += channel.Name + "," + channel.PUrl + "\n"
						continue
					}
				}
				res += channel.Name + "," + channel.Url + "\n"
			}

		}
	}

	if err := dao.Cache.Set(txtCaCheKey, []byte(res)); err != nil {
		log.Println("epg缓存设置失败:", err)
		dao.Cache.Delete(txtCaCheKey)
	}

	return res
}

func Txt2M3u8(txtData, host, token string) string {

	epgURL := host + "/epg/" + token + "/e.xml" // ✅ 可自行修改 EPG 地址
	logoBase := host + "/logo/"                 // ✅ 可自行修改 logo 前缀

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("#EXTM3U url-tvg=\"%s\"\n\n", epgURL))

	scanner := bufio.NewScanner(strings.NewReader(txtData))
	currentGroup := "未分组"
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 检查是否为分组行（如 “中央台,#genre#”）
		if strings.HasSuffix(line, "#genre#") {
			group := strings.TrimSuffix(line, ",#genre#")
			currentGroup = strings.TrimSpace(group)
			continue
		}

		// 普通频道行
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			fmt.Printf("Txt2M3u8: 第 %d 行格式错误: %s\n", lineNum, line)
			continue
		}

		name := strings.TrimSpace(parts[0])
		url := strings.TrimSpace(parts[1])
		epgName := GetEpgName(name)
		var logo string
		if epgName != "" {
			logo = fmt.Sprintf("%s%s.png", strings.TrimRight(logoBase, "/")+"/", epgName)
		}

		// ✅ 生成 #EXTINF 信息
		extinf := fmt.Sprintf(`#EXTINF:-1 tvg-id="%s" tvg-name="%s" tvg-logo="%s" group-title="%s",%s`,
			name, name, logo, currentGroup, name)
		builder.WriteString(extinf + "\n")
		builder.WriteString(url + "\n\n")
	}

	if err := scanner.Err(); err != nil {
		log.Println("Txt2M3u8: m3u8解析出错:", err)
	}

	return builder.String()
}

func GetEpg(id int64) dto.XmlTV {

	res := dto.XmlTV{
		GeneratorName: "清和IPTV管理系统",
		GeneratorURL:  "https://www.qingh.xyz",
	}

	epgCaCheKey := "rssEpgXml_" + strconv.FormatInt(id, 10)
	if dao.Cache.Exists(epgCaCheKey) {
		cacheData, err := dao.Cache.Get(epgCaCheKey)
		if err == nil {
			err := xml.Unmarshal(cacheData, &res)
			if err == nil {
				return res
			}
		}
	}

	var meal models.IptvMeals
	if err := dao.DB.Model(&models.IptvMeals{}).Where("id = ? and status = 1", id).First(&meal).Error; err != nil {
		return res
	}
	categoryIdList := strings.Split(meal.Content, ",")
	categoryIdList = slices.DeleteFunc(categoryIdList, func(s string) bool {
		return strings.TrimSpace(s) == ""
	})
	if len(categoryIdList) == 0 {
		return res
	}
	var categoryList []models.IptvCategory
	if err := dao.DB.Model(&models.IptvCategory{}).Where("id in (?) and enable = 1", categoryIdList).Order("sort asc").Find(&categoryList).Error; err != nil {
		return res
	}

	var channels []models.IptvChannelShow
	for _, category := range categoryList {
		if category.Type != "auto" {
			var tmpChannels []models.IptvChannelShow
			dao.DB.Model(&models.IptvChannelShow{}).Where("c_id = ? and status = 1", category.ID).Order("sort asc").Find(&tmpChannels)
			channels = append(channels, tmpChannels...)
		} else {
			channels = GetAutoChannelList(category)
		}
	}

	res = CleanTV(GetEpgXml(channels))

	data, err := xml.Marshal(res)
	if err == nil {
		err := dao.Cache.Set(epgCaCheKey, data)
		if err != nil {
			log.Println("epg缓存设置失败:", err)
			dao.Cache.Delete(epgCaCheKey)
		}
	} else {
		log.Println("epg缓存序列化失败:", err)
		dao.Cache.Delete(epgCaCheKey)
	}
	return res
}

func CleanTV(tv dto.XmlTV) dto.XmlTV {
	// 1️⃣ 去重 Channel（按 ID 保留第一个）
	uniqueChannels := make([]dto.XmlChannel, 0, len(tv.Channels))
	seen := make(map[string]bool)
	ids := make(map[string]int)
	i := 1
	for _, ch := range tv.Channels {
		if !seen[ch.ID] {
			seen[ch.ID] = true
			ids[ch.ID] = i
			ch.ID = strconv.Itoa(i)
			uniqueChannels = append(uniqueChannels, ch)
			i++
		}
	}
	tv.Channels = uniqueChannels

	// 2️⃣ 删除无效的 Programme（仅保留 channel 存在的）
	validProgrammes := make([]dto.Programme, 0, len(tv.Programmes))
	progSet := make(map[string]bool) // 记录唯一键

	for _, p := range tv.Programmes {
		if seen[p.Channel] {
			p.Channel = strconv.Itoa(ids[p.Channel])
			t, err := time.Parse("20060102150405 -0700", p.Start)
			if err != nil {
				log.Println("解析时间错误:", err)
				continue
			}
			key := p.Channel + "_" + fmt.Sprintf("%d", t.Unix()) + "_" + p.Title.Value // 唯一键

			if !progSet[key] {
				validProgrammes = append(validProgrammes, p)
				progSet[key] = true
			}
		}
	}
	tv.Programmes = validProgrammes

	return tv
}

func GetEpgXml(channelList []models.IptvChannelShow) dto.XmlTV {
	epgXml := dto.XmlTV{
		GeneratorName: "清和IPTV管理系统",
		GeneratorURL:  "https://www.qingh.xyz",
	}

	var epgXmlexit map[string]bool = make(map[string]bool)                               // 记录已经存在的epg xml
	var epgCache map[int64]models.IptvEpg = make(map[int64]models.IptvEpg)               // 记录已经存在的epg数据库
	var epgListCache map[string]models.IptvEpgList = make(map[string]models.IptvEpgList) // 记录已经存在的epg数据库
	for _, channel := range channelList {
		if channel.EId <= 0 {
			continue
		}
		if epgXmlexit[channel.Name] {
			continue
		}
		var epg models.IptvEpg
		if epgCache[channel.EId].ID <= 0 {
			if err := dao.DB.Model(&models.IptvEpg{}).Where("id = ? and status = 1", channel.EId).First(&epg).Error; err != nil {
				continue
			}
			epgCache[channel.EId] = epg
		} else {
			epg = epgCache[channel.EId]
		}

		fromList := strings.Split(epg.FromListStr, ",")

		if len(fromList) <= 0 || fromList[0] == "" {
			continue
		}

		dName := []dto.DisplayName{}
		exists := false

		if slices.Contains(fromList, "0") {
			if strings.EqualFold(epg.Name, "cctv5+") || strings.EqualFold(epg.Name, "cctv-5+") {
				epg.Name = "cctv5plus"
			}
			tmpData, err := GetEpgCntv(epg.Name)
			if err == nil {
				tmpXml := ConvertCntvToXml(tmpData, epg.Name)
				for k, c := range epgXml.Channels {
					if c.ID == epg.Name {
						exists = true
						var displayExists bool
						for _, v := range c.DisplayName {
							if v.Value == channel.Name {
								displayExists = true
								break
							}
						}
						if !displayExists {
							dName = append(c.DisplayName, dto.DisplayName{
								Lang:  "zh",
								Value: channel.Name,
							})
							epgXml.Channels[k].DisplayName = dName

						}
						break
					}
				}

				if !exists {
					dName = append(dName, dto.DisplayName{
						Lang:  "zh",
						Value: channel.Name,
					})
					epgXml.Channels = append(epgXml.Channels, dto.XmlChannel{
						ID:          epg.Name,
						DisplayName: dName,
					})
				}

				for _, p := range tmpXml.Programmes {
					p.Channel = epg.Name
					epgXml.Programmes = append(epgXml.Programmes, p)
				}
				if len(epgXml.Channels) > 0 && len(epgXml.Programmes) > 0 {
					epgXmlexit[channel.Name] = true
					continue
				}
				continue
			}
			if epgXmlexit[channel.Name] {
				continue
			}
		}

		for i, v := range fromList {
			if v == "0" || v == "" {
				fromList = append(fromList[:i], fromList[i+1:]...)
				break
			}
		}

		var epgFromList []models.IptvEpgList
		for _, from := range fromList {
			if epgListCache[from].ID > 0 {
				epgFromList = append(epgFromList, epgListCache[from])
			} else {
				var epgFrom models.IptvEpgList
				err := dao.DB.Where("id = ? and status = 1", from).First(&epgFrom).Error
				if err != nil {
					continue
				}
				epgListCache[from] = epgFrom
				epgFromList = append(epgFromList, epgFrom)
			}
		}

		for _, epgFrom := range epgFromList {
			if epgFrom.Url == "" || epgFrom.Name == "" {
				continue
			}
			tmpXml := GetEpgListXml(epgFrom.Name, epgFrom.Url)
			for k, c := range epgXml.Channels {
				if c.ID == epg.Name {
					exists = true
					dName = append(c.DisplayName, dto.DisplayName{
						Lang:  "zh",
						Value: channel.Name,
					})
					epgXml.Channels[k].DisplayName = dName
				}
			}

			if !exists {
				dName = append(dName, dto.DisplayName{
					Lang:  "zh",
					Value: channel.Name,
				})
				epgXml.Channels = append(epgXml.Channels, dto.XmlChannel{
					ID:          epg.Name,
					DisplayName: dName,
				})
			}

			var cId string
			for _, c := range tmpXml.Channels {
				if c.DisplayName[0].Value == epg.Name {
					cId = c.ID
					break
				}
			}

			for _, p := range tmpXml.Programmes {
				if p.Channel == cId {
					p.Channel = epg.Name
					epgXml.Programmes = append(epgXml.Programmes, p)
				}
			}
			if len(epgXml.Channels) > 0 && len(epgXml.Programmes) > 0 {
				epgXmlexit[channel.Name] = true
				break
			}
		}
	}

	return epgXml
}
