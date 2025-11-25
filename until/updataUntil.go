package until

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
	target := "entrypoint.sh"

	cmd := exec.Command("ps", "-eo", "pid,args")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return errors.New("进程列表获取失败")
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		fs := strings.Fields(line)
		if len(fs) < 2 {
			continue
		}
		pidStr := fs[0]
		args := strings.Join(fs[1:], " ")

		if strings.Contains(args, target) {
			pid, _ := strconv.Atoi(pidStr)
			p, _ := os.FindProcess(pid)
			if err := p.Signal(syscall.SIGUSR1); err != nil {
				return errors.New("更新信号发送失败")
			}
			log.Println("已发送更新信号")
			return nil
		}
	}
	return errors.New("未找到更新监测进程")
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

func downloadFile(url, dst string) error {
	if url == "" {
		return fmt.Errorf("下载URL为空")
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
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
	sumFile := "SHA256SUMS.txt"

	urlMap := map[string]string{}
	for _, a := range rel.Assets {
		urlMap[a.Name] = a.BrowserDownloadURL
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

	return true, rel.TagName, nil
}

func verifySHA(p string, sums map[string]string) bool {
	h, _ := fileSHA256(p)
	return strings.EqualFold(h, sums[filepath.Base(p)])
}
