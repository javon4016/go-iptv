package crontab

import (
	"fmt"
	"go-iptv/until"
	"log"
	"math/rand"
	"time"

	"github.com/robfig/cron/v3"
)

func EpgCron() {
	c := cron.New(cron.WithSeconds()) // 支持秒级别 cron 表达式

	// 生成1:00:00到5:59:59之间的随机时间
	rand.Seed(time.Now().UnixNano())
	hour := rand.Intn(5) + 1 // 1-5点
	minute := rand.Intn(60)  // 0-59分
	second := rand.Intn(60)  // 0-59秒

	// 构建cron表达式（秒 分 时 * * *）
	cronExpr := fmt.Sprintf("%d %d %d * * *", second, minute, hour)
	log.Println("设置随机EPG自动更新时间为: %02d:%02d:%02d", hour, minute, second)

	// cron 表达式格式: 秒 分 时 日 月 星期
	// 下面表示每天 01:00:00
	c.AddFunc(cronExpr, func() {
		log.Println("自动更新EPG列表任务开始执行:", time.Now().Format("2006-01-02 15:04:05"))
		// 在这里写你的任务逻辑
		if until.UpdataEpgList() {
			log.Println("自动更新EPG列表任务执行成功:", time.Now().Format("2006-01-02 15:04:05"))
		} else {
			log.Println("自动更新EPG列表任务执行失败:", time.Now().Format("2006-01-02 15:04:05"))
		}
	})
	c.Start()
}
