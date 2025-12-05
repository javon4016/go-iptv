package until

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iptv/dao"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func IsRunning() bool {
	cmd := exec.Command("bash", "-c", "ps -ef | grep '/license' | grep -v grep")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return checkRun()
	}
	return strings.Contains(string(output), "license")
}

func checkRun() bool {
	defaultUA := "Go-http-client/1.1"
	useUA := defaultUA

	req, err := http.NewRequest("GET", "http://127.0.0.1:81/", nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", useUA)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return strings.Contains(string(body), "ok")
}

func RestartLic() bool {
	log.Println("♻️ 正在重启引擎...")

	r := GetUrlData("http://127.0.0.1:82/licRestart")
	if strings.TrimSpace(r) == "" {
		log.Println("升级服务未启动")
		return false
	}
	if strings.TrimSpace(r) != "OK" {
		return false
	}

	ws, err := dao.ConLicense("ws://127.0.0.1:81/ws")
	if err != nil {
		log.Println("引擎连接失败：", err)
		return false
	}
	dao.WS = ws
	res, err := dao.WS.SendWS(dao.Request{Action: "getlic"})
	if err == nil {
		if err := json.Unmarshal(res.Data, &dao.Lic); err == nil {
			log.Println("引擎初始化成功")
			log.Println("机器码:", dao.Lic.ID)
		} else {
			log.Println("授权信息解析错误:", err)
		}
	} else {
		log.Println("引擎初始化错误")
		return false
	}

	log.Println("✅  引擎已成功重启并重新连接")
	return true
}

func InitProxy() {
	var scheme, pAddr string
	var port int64

	cfg := dao.GetConfig()
	if cfg.Proxy.PAddr == "" {
		scheme, pAddr, port = ParseURL(cfg.ServerUrl)
	} else {
		scheme = cfg.Proxy.Scheme
		pAddr = cfg.Proxy.PAddr
		port = cfg.Proxy.Port
	}

	if scheme == "" || scheme == "http" {
		scheme = "http"
		if port == 0 {
			port = 80
		}
	} else {
		scheme = "https"
		if port == 0 {
			port = 443
		}
	}

	pAddr = strings.TrimPrefix(strings.TrimPrefix(pAddr, "https://"), "http://")

	if scheme != cfg.Proxy.Scheme || pAddr != cfg.Proxy.PAddr || port != cfg.Proxy.Port {
		cfg.Proxy.Scheme = scheme
		cfg.Proxy.PAddr = pAddr
		cfg.Proxy.Port = port
		dao.SetConfig(cfg)
	}
}

func CheckLicVer(latest string) (bool, error) {
	var oldVer string
	verJson, err := dao.WS.SendWS(dao.Request{Action: "getVersion"})
	if err != nil {
		oldVer = ReadFile("/config/bin/Version_lic")
		if oldVer == "" {
			return false, errors.New("引擎版本号获取失败，请检查引擎状态")
		}
	} else {
		if err := json.Unmarshal(verJson.Data, &oldVer); err != nil {
			log.Println("引擎版本信息解析错误:", err)
			return false, errors.New("引擎版本号获取失败")
		}
	}

	if latest == oldVer {
		return true, nil
	}
	vLen := 3
	latest = strings.TrimPrefix(latest, "v")
	oldVer = strings.TrimPrefix(oldVer, "v")

	np := strings.Split(latest, ".")
	op := strings.Split(oldVer, ".")
	for len(np) < vLen {
		np = append(np, "0")
	}
	for len(op) < vLen {
		op = append(op, "0")
	}

	for i := 0; i < vLen; i++ {
		var a, b int
		fmt.Sscanf(np[i], "%d", &a)
		fmt.Sscanf(op[i], "%d", &b)
		if a > b {
			return false, errors.New("该功能需要引擎最低版本为: " + latest + " ,当前版本为: " + oldVer + " ,请升级引擎")
		}
		if a == b {
			continue
		}
		if a < b {
			return true, nil
		}
	}
	return false, errors.New("版本号读取失败")
}
