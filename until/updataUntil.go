package until

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type githubRelease struct {
	TagName     string    `json:"tag_name"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// ------------------------------------------------------------
// 更新信号
// ------------------------------------------------------------

func UpdateSignal() error {
	res, err := http.Get("http://127.0.0.1:82/update")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("更新请求失败，状态码: %d", res.StatusCode)
	}

	return nil
}

// ------------------------------------------------------------
// 获取 release
// ------------------------------------------------------------

func fetchLatestStableRelease(owner, repo string) (*githubRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	var latest *githubRelease
	for _, r := range releases {
		if r.Prerelease {
			continue
		}
		if latest == nil || r.PublishedAt.After(latest.PublishedAt) {
			latest = &r
		}
	}
	if latest == nil {
		return nil, errors.New("无正式版")
	}
	return latest, nil
}

// ------------------------------------------------------------
// CheckNewVer
// ------------------------------------------------------------

func isNewer(newVer, oldVer string) bool {
	newVer = strings.TrimPrefix(newVer, "v")
	oldVer = strings.TrimPrefix(oldVer, "v")

	np := strings.Split(newVer, ".")
	op := strings.Split(oldVer, ".")
	for len(np) < 4 {
		np = append(np, "0")
	}
	for len(op) < 4 {
		op = append(op, "0")
	}

	for i := 0; i < 4; i++ {
		var a, b int
		fmt.Sscanf(np[i], "%d", &a)
		fmt.Sscanf(op[i], "%d", &b)
		if a > b {
			return true
		}
		if a < b {
			return false
		}
	}
	return false
}

func CheckNewVer(local string) (bool, string, error) {
	latest, err := fetchLatestStableRelease("wz1st", "go-iptv")
	if err != nil {
		return false, "", err
	}
	return isNewer(latest.TagName, local), latest.TagName, nil
}

// ------------------------------------------------------------
// 下载
// ------------------------------------------------------------

func downloadFile(urlStr, dst string) error {
	if urlStr == "" {
		return fmt.Errorf("下载URL为空")
	}

	// 设置 Transport，限制连接超时、TLS握手超时、响应头超时
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second, // 等待响应头超时
	}

	// 如果存在 PROXY 环境变量，设置代理
	if proxyEnv := os.Getenv("PROXY"); proxyEnv != "" {
		proxyURL, err := url.Parse(proxyEnv)
		if err != nil {
			return fmt.Errorf("代理URL无效: %v", err)
		}
		tr.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   0, // 不限制整个下载总时间
	}

	maxRetries := 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(urlStr)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return nil // 下载成功
	}

	return fmt.Errorf("下载失败: %v", lastErr)
}

// ------------------------------------------------------------
// SHA
// ------------------------------------------------------------

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func loadSums(file string) map[string]string {
	r := map[string]string{}

	f, err := os.Open(file)
	if err != nil {
		return r
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		p := strings.Fields(sc.Text())
		if len(p) == 2 {
			r[p[1]] = p[0]
		}
	}
	return r
}

// ------------------------------------------------------------
// cp
// ------------------------------------------------------------

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// ------------------------------------------------------------
// 主逻辑
// ------------------------------------------------------------

func DownloadAndVerify(arch string) (bool, string, error) {

	rel, err := fetchLatestStableRelease("wz1st", "go-iptv")
	if err != nil {
		return false, "", err
	}

	downDir := "/tmp/down"
	upDir := "/config/updata"

	os.MkdirAll(downDir, 0755)
	os.MkdirAll(upDir, 0755)

	iptv := "iptv_" + arch
	license := "license_" + arch

	required := []string{iptv, license}
	optional := []string{"updata.sh"}
	verFile := "Version"
	sumFile := "SHA256SUMS.txt"

	urlMap := map[string]string{}
	for _, a := range rel.Assets {
		urlMap[a.Name] = a.BrowserDownloadURL
	}

	if err := downloadFile(urlMap[verFile], filepath.Join(downDir, verFile)); err != nil {
		return false, "", err
	}

	// --------------------------------
	// 1) 总是先下载 SHA256SUMS.txt
	// --------------------------------
	if err := downloadFile(urlMap[sumFile], filepath.Join(downDir, sumFile)); err != nil {
		return false, "", err
	}

	sums := loadSums(filepath.Join(downDir, sumFile))

	// --------------------------------
	// 2) 有文件 → 校验
	// --------------------------------
	need := []string{}

	for _, f := range append(required, optional...) {
		local := filepath.Join(downDir, f)
		if _, err := os.Stat(local); err == nil {
			if verifySHA(local, sums) {
				continue
			}
		}
		need = append(need, f)
	}

	// --------------------------------
	// 3) 下载缺失/校验失败的
	// --------------------------------
	for _, f := range need {
		u := urlMap[f]
		if u == "" {
			continue
		}
		if err := downloadFile(u, filepath.Join(downDir, f)); err != nil {
			return false, "", err
		}
	}

	// --------------------------------
	// 4) 最终校验必需
	// --------------------------------
	for _, f := range required {
		p := filepath.Join(downDir, f)
		if !verifySHA(p, sums) {
			return false, "", fmt.Errorf("%s 校验失败", f)
		}
	}

	// --------------------------------
	// 5) 删除旧
	// --------------------------------
	os.Remove(filepath.Join(upDir, "iptv"))
	os.Remove(filepath.Join(upDir, "license"))
	os.Remove(filepath.Join(upDir, "Version"))
	// os.Remove(filepath.Join(upDir, "updata.sh"))

	// --------------------------------
	// 6) 覆盖 + 去掉_arch
	// --------------------------------
	cp := func(f string) {
		src := filepath.Join(downDir, f)
		if _, err := os.Stat(src); err == nil {
			dst := filepath.Join(upDir, strings.Replace(f, "_"+arch, "", 1))
			copyFile(src, dst)
		}
	}
	cp(iptv)
	cp(license)
	cp("updata.sh")
	cp(verFile)

	return true, rel.TagName, nil
}

func verifySHA(p string, sums map[string]string) bool {
	h, _ := fileSHA256(p)
	return strings.EqualFold(h, sums[filepath.Base(p)])
}
