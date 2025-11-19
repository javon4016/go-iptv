package main

import (
	"flag"
	"go-iptv/bootstrap"
	"go-iptv/crontab"
	"go-iptv/dao"
	"go-iptv/router"
	"go-iptv/until"
	"log"
	"os"
)

func main() {

	if !until.IsPrivileged() {
		log.Println("请使用privileged(特权模式、高权限执行容器)运行")
		return
	}

	if until.CheckRam() {
		log.Println("可用内存不足256MB，无法运行")
		return
	}

	build := true
	if os.Getenv("NOBUILD") == "true" {
		build = false
	}

	port := flag.String("port", "80", "启动端口 eg: 80")
	flag.Parse()
	if !until.CheckPort(*port) {
		return
	}

	if !until.CheckJava() {
		log.Println("请安装Java JDK 1.8环境")
		return
	}

	if !until.CheckApktool() {
		log.Println("请安装apktool环境")
		return
	}

	var debug bool = false
	if os.Getenv("DEBUG") == "true" {
		debug = true
	}

	log.Println("初始化EPG缓存...")
	cache, err := dao.NewFileCache("/config/cache/", true)
	if err != nil {
		log.Println("初始化缓存失败:", err)
		return
	}
	dao.Cache = cache
	if dao.Cache.Clear() != nil {
		log.Println("初始化清除缓存失败:", err)
		return
	}

	bootstrap.InitAlias() // 初始化epg别名

	if os.Getenv("IPTVDEV") != "true" {
		bootstrap.InitLicense() // 初始化授权信息
	}

	if !until.Exists("/config/iptv.db") || !until.Exists("/config/config.yml") || !until.Exists("/config/install.lock") {
		bootstrap.Installed = false
		log.Println("检测到未安装，开始安装...")
		log.Println("启动接口...")
		router := router.InitRouter(debug)
		router.Run(":" + *port)
	} else {
		bootstrap.Installed = true
	}

	dao.CONFIG_PATH = "/config/config.yml"
	dao.LoadConfigFile()

	if !dao.LoadConfig() {
		log.Println("conf加载错误")
		return
	}

	log.Println("加载数据库...")
	if debug {
		dao.InitDBDebug("/config/iptv.db")
	} else {
		dao.InitDB("/config/iptv.db")
	}

	if !bootstrap.InitDB() {
		log.Println("数据库初始化失败,请删除/config/iptv.db重新安装")
		return
	}
	until.PasswordReset() // 密码重置

	if !bootstrap.InitLogo() {
		log.Println("logo目录初始化错误")
		return
	}

	go crontab.Crontab()
	go crontab.EpgCron()
	go until.InitCacheRebuild()

	if !debug {
		bootstrap.InitJwtKey() // 初始化JWTkey
		if build {
			if !bootstrap.BuildAPK() {
				log.Println("APK编译错误")
				return
			}
		}
	}

	log.Println("启动接口...")
	router := router.InitRouter(debug)
	router.Run(":" + *port)
}
