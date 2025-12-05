package router

import (
	"go-iptv/api"
	"go-iptv/html"
	"go-iptv/until"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AdminRouter(r *gin.Engine, path string) {
	router := r.Group(path)
	{
		router.POST("/login", api.Login)
		router.GET("/login", html.Login)
		router.GET("/logout", api.Logout)

		router.Use(JWTMiddleware(router))
		{
			router.GET("/", html.Index)
			router.GET("/index", html.Index)

			router.GET("/users", html.Users)
			router.POST("/users", api.EditUsers)

			router.GET("/authors", html.Authors)
			router.POST("/authors", api.Authors)

			router.GET("/meals", html.Meals)
			router.POST("/meals", api.Meals)

			router.GET("/channels", html.Channels)
			router.POST("/channels", api.Channels)
			router.POST("/channels/uploadPayList", api.UploadPayList)
			router.POST("/channels/uploadLogo", api.UploadLogo)

			router.GET("/epgsList", html.Epgs)
			router.POST("/epgsList", api.Epgs)

			router.GET("/epgFrom", html.EpgsFrom)
			router.POST("/epgFrom", api.EpgsFrom)

			router.GET("/notice", html.Notice)
			router.POST("/notice", api.Notice)

			router.GET("/client", html.Client)
			router.POST("/client", api.Client)
			router.GET("/client/buildStatus", api.BuildStatus)
			router.POST("/client/uploadIcon", api.ClientUploadIcon)
			router.POST("/client/uploadBj", api.ClientUploadBj)

			router.GET("/admins", html.Admins)
			router.POST("/admins", api.Admins)

			router.GET("/movie", html.Movie)
			router.POST("/movie", api.Movie)

			router.GET("/about", html.About)

			router.GET("/license", html.License)
			router.GET("/license/checkProxy", api.CheckProxy)
			router.POST("/license", api.License)

			router.GET("/updata", html.Updata)
			router.GET("/updata/checkWeb", api.UpdataCheckWeb)
			router.GET("/updata/checkLic", api.UpdataCheckLic)
			router.GET("/updata/downWeb", api.UpdataDownWeb)
			router.GET("/updata/downLic", api.UpdataDownLic)
			router.GET("/updata/updata", api.Updata)

			router.POST("/getRssUrl", api.GetRssUrl)
		}
	}
}

func JWTMiddleware(r *gin.RouterGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Cookie 获取 token
		tokenString, err := c.Cookie("token")
		if err != nil {
			// c.JSON(200, dto.NewAdminRedirectDto())
			c.Redirect(http.StatusFound, r.BasePath()+"/login") // 重定向到登录页面
			c.Abort()
			return
		}

		// 调用 VerifyJWT 验证 token
		claims, err, update := until.VerifyJWT(tokenString)
		if err != nil {
			// c.JSON(200, dto.NewAdminRedirectDto())
			c.Redirect(http.StatusFound, r.BasePath()+"/login") // 重定向到登录页面
			c.Abort()
			return
		}
		if update {
			// 更新 token
			tokenString, _ := until.GenerateJWT(claims["username"].(string), time.Hour)
			c.SetCookie("token", tokenString, 3600, "/", "", false, true)
		}

		// 保存 claims 到上下文
		c.Set("auth", claims)

		c.Next()
	}
}
