package router

import (
	"go-iptv/assets"
	"go-iptv/bootstrap"
	"go-iptv/crontab"
	"go-iptv/dao"
	"go-iptv/dto"
	"go-iptv/models"
	"go-iptv/until"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

func InitRouter(debug bool) *gin.Engine {
	if os.Getenv("IPTVDEV") != "true" {
		gin.SetMode(gin.ReleaseMode)
	}

	var r *gin.Engine
	if debug {
		r = gin.Default()
	} else {
		r = gin.New()
	}

	r.SetTrustedProxies([]string{
		"10.0.0.0/8",
		"192.168.0.0/16",
		"172.0.0.0/8",      // docker私有网络地址
		"::1",              // IPv6 localhost
		"127.0.0.1",        // IPv4 localhost
		"::ffff:127.0.0.1", // IPv6 mapped IPv4 localhost
	})
	r.RemoteIPHeaders = []string{"X-Original-Forwarded-For", "X-Real-IP", "X-Forwarded-For"}

	r.SetFuncMap(template.FuncMap{
		"SiteName": func() string { return "清和IPTV管理系统" },
		"Version":  func() string { return "清和IPTV " + until.GetVersion() },
		"Author":   func() int64 { return dao.GetConfig().App.NeedAuthor },
		"Add":      func(a, b int64) int64 { return a + b },
		"Sub":      func(a, b int64) int64 { return a - b },
	})

	r.Static("/app", "./app")
	r.Static("/images", "/config/images/bj")
	r.Static("/icon", "/config/images/icon")
	r.Static("/logo", "/config/logo")

	r.Use(func(c *gin.Context) {
		if !bootstrap.Installed {
			path := c.Request.URL.Path
			if path != "/" && path != "/install" && path != "/ChangeLog.md" &&
				!strings.HasPrefix(path, "/images/") &&
				!strings.HasPrefix(path, "/favicon.ico") &&
				!strings.HasPrefix(path, "/static/") {
				c.Redirect(http.StatusFound, "/")
				c.Abort()
				return
			}
		}
		c.Next()
	})

	ApkRouter(r, "/apk")
	AdminRouter(r, "/admin")
	RssRouter(r, "/")

	loadTemplates(r)

	r.GET("/", func(c *gin.Context) {

		var pageData dto.IndexDto

		if !bootstrap.Installed {
			data, _ := os.ReadFile("/app/README.md")
			pageData.Content = string(data)
			c.HTML(http.StatusOK, "install_1.html", pageData)
			return
		}

		cfg := dao.GetConfig()

		timeStr, err := until.GetFileModTimeStr("./app/" + cfg.Build.Name + ".apk")
		if err != nil {
			pageData.ApkTime = "未知"
		} else {
			pageData.ApkTime = timeStr
			pageData.ApkName = cfg.Build.Name + ".apk"
			pageData.ApkVersion = cfg.Build.Version
			pageData.ApkSize = until.GetFileSize("./app/" + cfg.Build.Name + ".apk")
			pageData.ApkUrl = "/app/" + cfg.Build.Name + ".apk"
		}

		pageData.Status = bootstrap.GetBuildStatus()

		ua := c.GetHeader("User-Agent")
		templateName := "index.html" // 默认 PC 模板
		if isMobile(ua) {
			templateName = "mobile.html"
		}
		c.HTML(http.StatusOK, templateName, pageData)
	})

	r.GET("/ChangeLog.md", func(c *gin.Context) {
		var pageData dto.IndexDto
		data, _ := os.ReadFile("/app/ChangeLog.md")
		pageData.Content = string(data)
		c.HTML(http.StatusOK, "install_log.html", pageData)
	})

	r.GET("/install", func(c *gin.Context) {
		if bootstrap.Installed {
			c.HTML(http.StatusOK, "install_3.html", gin.H{})
			return
		}
		c.HTML(http.StatusOK, "install_2.html", gin.H{})
	})

	r.POST("/install", func(c *gin.Context) {

		username := c.PostForm("username")
		password := c.PostForm("password")
		password2 := c.PostForm("password2")
		apkApi := c.PostForm("apkapi")
		if apkApi == "" {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "APK接口错误",
				"type": "danger",
			})
			return
		}

		if username == "" || password == "" || password2 == "" {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "用户名或密码不能为空",
				"type": "danger",
			})
			return

		}

		if password != password2 {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "两次密码不一致",
				"type": "danger",
			})
			return
		}

		password = until.HashPassword(password)

		if !bootstrap.Installed {
			status, msg := bootstrap.Install()
			if status {
				dao.DB.Model(&models.IptvAdmin{}).Create(&models.IptvAdmin{
					UserName: username,
					PassWord: password,
				})
				cfg := dao.GetConfig()
				cfg.ServerUrl = strings.TrimSuffix(apkApi, "/")
				dao.SetConfig(cfg)
				crontab.StopChan = make(chan struct{})
				if os.Getenv("NOBUILD") != "true" {
					go bootstrap.BuildAPK()
				}
				go crontab.Crontab()
				go crontab.EpgCron()
				go until.InitCacheRebuild()
				bootstrap.Installed = true
				c.JSON(http.StatusOK, gin.H{
					"code": 1,
					"msg":  "安装成功,正在编译APK,请稍后访问" + cfg.ServerUrl + "查看...",
					"type": "success",
				})
				return
			} else {
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  msg,
					"type": "danger",
				})
			}

		} else {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "已安装，请勿重复安装",
				"type": "danger",
			})
			return
		}
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(200, "image/x-icon", assets.Favicon)
	})

	// r.Use(NoCache)
	r.Use(Cors)
	return r
}

func loadTemplates(r *gin.Engine) {
	dir, err := os.Getwd()
	if err != nil {
		log.Println("获取当前目录失败:", err)
		return
	}
	log.Println("当前工作目录:", dir)
	if gin.Mode() == gin.ReleaseMode {
		// 生产环境：用 embed.FS
		tmpl := template.New("").Funcs(template.FuncMap{
			"SiteName": func() string { return "清和IPTV管理系统" },
			"Version":  func() string { return "清和IPTV " + until.GetVersion() },
			"Author":   func() int64 { return dao.GetConfig().App.NeedAuthor },
			"Add":      func(a, b int64) int64 { return a + b },
			"Sub":      func(a, b int64) int64 { return a - b },
		})
		tmpl = template.Must(tmpl.ParseFS(assets.EmbeddedFS, "templates/*"))
		staticFiles, _ := fs.Sub(assets.StaticFS, "static")
		r.StaticFS("/static", http.FS(staticFiles))

		r.SetHTMLTemplate(tmpl)
	} else {
		log.Println("当前为开发模式，使用本地模板文件")
		r.Static("/static", "./assets/static")
		r.Static("/favicon.ico", "./assets/static/favicon.ico")

		// 开发环境：直接读取磁盘
		r.LoadHTMLGlob("./assets/templates/*")
	}
}

func isMobile(userAgent string) bool {
	ua := strings.ToLower(userAgent)

	// AppleWebKit.*mobile
	re1 := regexp.MustCompile(`(?i)applewebkit.*mobile`)
	if re1.MatchString(ua) {
		return true
	}

	// MIDP|SymbianOS|NOKIA|SAMSUNG|LG|NEC|TCL|Alcatel|BIRD|DBTEL|Dopod|PHILIPS|HAIER|LENOVO|MOT-|Nokia|SonyEricsson|SIE-|Amoi|ZTE
	re2 := regexp.MustCompile(`(?i)MIDP|SymbianOS|NOKIA|SAMSUNG|LG|NEC|TCL|Alcatel|BIRD|DBTEL|Dopod|PHILIPS|HAIER|LENOVO|MOT-|Nokia|SonyEricsson|SIE-|Amoi|ZTE`)
	return re2.MatchString(userAgent)
}

//	func NoCache(c *gin.Context) {
//		c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
//		c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
//		c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
//		c.Next()
//	}
func Cors(c *gin.Context) {
	if c.Request.Method != "OPTIONS" {
		c.Next()
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
		c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Content-Type", "application/json")
		c.AbortWithStatus(200)
	}
}
