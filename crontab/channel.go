package crontab

import (
	"errors"
	"fmt"
	"go-iptv/dao"
	"go-iptv/models"
	"go-iptv/until"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	CrontabStatus bool
	UpdateStatus  bool
	StopChan      = make(chan struct{})
	UpdateChan    = make(chan time.Duration) // 新增：用于动态调整间隔
	ticker        *time.Ticker
)

func Crontab() {
	if CrontabStatus {
		log.Println("定时任务已在运行，尝试更新定时间隔...")

		cfg := dao.GetConfig()
		newInterval := time.Duration(cfg.Channel.Interval) * time.Hour
		if newInterval <= 0 {
			log.Println("新间隔无效，忽略更新")
			return
		}

		// 发送新的时间间隔信号
		select {
		case UpdateChan <- newInterval:
			log.Println("已更新定时间隔为：", newInterval)
		default:
			log.Println("更新信号通道被占用，稍后再试")
		}
		return
	}

	cfg := dao.GetConfig()
	autoUpdate := cfg.Channel.Auto
	upInterval := cfg.Channel.Interval

	if autoUpdate == 1 && upInterval > 0 {
		log.Println("定时任务服务启动...")
		CrontabStatus = true
		defer func() { CrontabStatus = false }()

		interval := time.Duration(upInterval) * time.Hour
		ticker = time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case t := <-ticker.C:
				if UpdateStatus {
					log.Println("正在更新频道，请稍后...")
					continue
				}
				log.Println("开始执行更新频道任务：", t.Format("2006-01-02 15:04:05"))
				UpdateList()

			case newInterval := <-UpdateChan:
				log.Println("接收到新定时间隔，更新中...")
				ticker.Stop()
				ticker = time.NewTicker(newInterval)
				log.Println("频道更新时间间隔已更新为：", newInterval)

			case <-StopChan:
				log.Println("收到停止信号，停止更新频道任务")
				ticker.Stop()
				return
			}
		}
	} else {
		log.Println("定时任务服务未开启...")
	}
}

func UpdateList() {
	UpdateStatus = true
	defer func() { UpdateStatus = false }()
	// TODO: 定时任务
	var lists []models.IptvCategoryList
	res := dao.DB.Model(&models.IptvCategoryList{}).Where("1=1").Find(&lists)

	if res.RowsAffected == 0 {
		log.Println("没有可更新的频道列表")
		return
	}

	client := &http.Client{}
	for _, v := range lists {
		req, err := http.NewRequest("GET", strings.TrimSpace(v.Url), nil)
		if err != nil {
			log.Println("更新频道列表失败--->创建请求错误:: ", err.Error(), " URL: ", v.Url)
			continue
		}

		// 添加自定义 User-Agent
		req.Header.Set("User-Agent", v.UA)

		resp, err := client.Do(req)
		if err != nil {
			log.Println("更新频道列表失败--->无法访问url: ", err.Error(), " URL: ", v.Url)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Println("更新频道列表失败--->读取响应失败-状态码：", resp.StatusCode, " URL: ", v.Url)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("更新频道列表失败--->读取响应失败：", " URL: ", v.Url)
			continue
		}

		urlData := until.FilterEmoji(string(body)) // 过滤emoji表情

		if until.IsM3UContent(urlData) {
			urlData = until.M3UToGenreTXT(urlData)
		}

		var doRepeat = false
		if v.Repeat == 1 {
			doRepeat = true
		}

		updata := map[string]interface{}{
			"latesttime": time.Now().Format("2006-01-02 15:04:05"),
		}

		var oldC models.IptvCategory
		err = dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", v.ID).First(&oldC).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			log.Println("获取 EPG 列表失败:", err)
			continue
		}

		if v.AutoCategory == 1 {
			if !strings.Contains(urlData, "#genre#") {
				updata["autocategory"] = 0
				var oldC models.IptvCategory
				dao.DB.Model(&models.IptvCategory{}).Where("list_id = ?", v.ID).First(&oldC)
				until.AddChannelList(urlData, oldC.ID, v.ID, doRepeat)
			}
			GenreChannels(v.Name, urlData, v.UA, v.ID, doRepeat)
		} else {
			until.AddChannelList(urlData, oldC.ID, v.ID, doRepeat)
		}
		dao.DB.Model(&models.IptvCategoryList{}).Where("id = ?", v.ID).Updates(updata)
	}
	log.Println("定时执行更新频道任务结束")
}

func GenreChannels(listName, srclist, ua string, listId int64, doRepeat bool) {

	data := until.ConvertDataToMap(srclist)

	for genreName, genreList := range data {
		genreName = strings.TrimSpace(genreName)
		if genreName == "" {
			continue
		}

		categoryName := strings.ReplaceAll(fmt.Sprintf("%s(%s)", genreName, listName), " ", "")

		var category models.IptvCategory
		dao.DB.Model(&models.IptvCategory{}).Where("name = ?", categoryName).First(&category)

		if category.ID == 0 {
			var maxSort int64
			dao.DB.Model(&models.IptvCategory{}).Select("IFNULL(MAX(sort),0)").Scan(&maxSort)
			category := models.IptvCategory{
				Name:   categoryName,
				Sort:   maxSort + 1,
				Type:   "add",
				ListId: listId,
				UA:     ua,
			}

			if err := dao.DB.Create(&category).Error; err != nil {
				continue
			}
			go until.SyncCaToEpg(category.ID)
			until.AddChannelList(genreList, category.ID, listId, doRepeat)
		} else {
			until.AddChannelList(genreList, category.ID, listId, doRepeat)
			proxyCaCheck := "proxyCaCheck_" + strconv.FormatInt(category.ID, 10)
			dao.Cache.Delete(proxyCaCheck)
		}
	}
	log.Println("更新" + listName + "分类结束")
}
