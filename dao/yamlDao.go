package dao

import (
	"fmt"
	"go-iptv/dto"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	// CONFIG       dto.Config
	// BUILD        dto.Build
	// APP          dto.App
	// TIPS         dto.Tips
	// AD           dto.Ad
	// SECURITY     dto.Security
	// EPG_ERROR    dto.EPGErrors
	// WEATHER      dto.Weather
	CONFIG_PATH  string
	GlobalConfig atomic.Value

	saveMutex sync.Mutex               // 保护定时器
	saveTimer *time.Timer              // 写文件定时器
	saveDelay = 500 * time.Millisecond // 去抖动延迟
	updating  atomic.Bool              // 是否正在更新配置
)

// 加载配置文件
func LoadConfigFile() bool {
	if CONFIG_PATH == "" {
		log.Println("配置文件路径为空")
		return false
	}

	viper.SetConfigFile(CONFIG_PATH)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("找不到配置文件:", CONFIG_PATH)
		} else {
			log.Println("配置文件解析出错:", err)
		}
		return false
	}

	return true
}

func LoadConfig() bool {
	var cfg dto.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Println("解析配置文件出错:", err)
		return false
	}

	// if cfg.Cache.Type == "" {
	// 	cfg.Cache.Type = "file"
	// }

	// if cfg.Cache.Type != "file" && cfg.Cache.Type != "redis" {
	// 	cfg.Cache.Type = "file"
	// }

	// if cfg.Cache.Type == "redis" && cfg.Cache.Redis.Host == "" {
	// 	cfg.Cache.Type = "file"
	// }

	log.Println("配置文件加载成功:", CONFIG_PATH)
	GlobalConfig.Store(&cfg)
	return true
}

func GetConfig() *dto.Config {
	v := GlobalConfig.Load()
	if v == nil {
		return nil
	}
	return v.(*dto.Config)
}

func SetConfig(cfg *dto.Config) {
	if cfg == nil {
		log.Println("配置数据为空，跳过")
		return
	}
	GlobalConfig.Store(cfg)
	scheduleSave()
	// SaveConfigToFile()

	// LoadConfig()
}

func WatchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		time.Sleep(saveDelay)
		if updating.Load() {
			// 内部修改，不处理
			return
		}
		log.Println("配置文件刚更新，正在重新加载...")
		LoadConfig()
	})
}

func SaveConfigToFile() error {
	v := GlobalConfig.Load()
	if v == nil {
		return fmt.Errorf("GlobalConfig为空，无法写入文件")
	}

	cfg := v.(*dto.Config)

	// 序列化为 YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("YAML序列化失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(CONFIG_PATH, data, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

func scheduleSave() {
	saveMutex.Lock()
	defer saveMutex.Unlock()

	if saveTimer != nil {
		saveTimer.Stop()
	}
	saveTimer = time.AfterFunc(saveDelay, func() {
		updating.Store(true)        // 标记正在写入
		defer updating.Store(false) // 写入完成，取消标记
		if err := SaveConfigToFile(); err != nil {
			log.Println("保存配置文件失败:", err)
		} else {
			go WS.SendWS(Request{Action: "reloadConfig"})
			log.Println("全局配置已保存到文件")
			// LoadConfig()
		}
	})
}
