package native_image

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	ctrl "github.com/FloatTech/zbpctrl"
	"github.com/FloatTech/zbputils/control"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	engine := control.Register("native_image", &ctrl.Options[*zero.Ctx]{
		DisableOnDefault: false,
		// 插件的简介
		Brief:             "native_image",
		Help:              "- /添加 [标签]\n- /随机 [标签]",
		PrivateDataFolder: "nativeImage",
	})
	engine.OnPrefix("/添加").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		var imageURL []string
		var imageName []string
		for _, elem := range ctx.Event.Message {
			if elem.Type == "image" {
				imageName = append(imageName, elem.Data["file"])
				imageURL = append(imageURL, elem.Data["url"])
			}
		}
		tagName := strings.TrimSpace(ctx.State["args"].(string))
		tagDirName := engine.DataFolder() + tagName + "/"
		_, err := os.Stat(tagDirName)
		if os.IsNotExist(err) {
			// 目录不存在，创建目录
			os.MkdirAll(tagDirName, 0755) // 0755是权限设置，可读写执行，用户组和其他用户也有相应权限
		}
		for i := range imageURL {
			//ctx.SendChain(message.Text(imageURL[i]))
			ext := filepath.Ext(imageName[i])
			sha256Hash := sha256.Sum256([]byte(imageURL[i] + imageName[i]))
			hashVlue := hex.EncodeToString(sha256Hash[:])
			//ctx.SendChain(message.Text(tagDirName + hashVlue + ext))
			err := downloadQQMedia(imageURL[i], tagDirName+hashVlue+ext)
			if err != nil {
				ctx.SendChain(message.Text(err.Error()))
				return
			}
		}
		msg := fmt.Sprintf("成功保存%d张照片到%s里.", len(imageURL), tagName)
		ctx.SendChain(message.Text(msg))
	})
	engine.OnPrefix("/随机").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		tagName := strings.TrimSpace(ctx.State["args"].(string))
		tagDirName := engine.DataFolder() + tagName + "/"
		entries, err := os.ReadDir(tagDirName)
		if err != nil {
			ctx.SendChain(message.Text("找不到标签。"))
			return
		}
		fileCount := 0
		for _, entry := range entries {
			if !entry.IsDir() { // 确保不是目录
				fileCount++
			}
		}
		rand.Seed(time.Now().UnixNano())
		if len(entries) == 0 { // 检查目录是否为空
			ctx.SendChain(message.Text("空标签。"))
			return
		}
		randomIndex := rand.Intn(len(entries))
		randomFile := entries[randomIndex]
		data, err := os.ReadFile(tagDirName + randomFile.Name())
		if err != nil {
			ctx.SendChain(message.Text("图片加载失败。"))
			return
		}

		base64Str := base64.StdEncoding.EncodeToString(data)

		ctx.SendChain(message.Image("base64://" + base64Str))
	})
}

func downloadQQMedia(url string, savePath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://im.qq.com/")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}
