package dlsite_crawler

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/FloatTech/floatbox/binary"
	ctrl "github.com/FloatTech/zbpctrl"
	"github.com/FloatTech/zbputils/control"
	fctext "github.com/FloatTech/zbputils/img/text"
	"github.com/PuerkitoBio/goquery"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	engine := control.Register("dlsite_crawler", &ctrl.Options[*zero.Ctx]{
		DisableOnDefault: false,
		// 插件的简介
		Brief: "dlsite_crawler",
		Help:  "",
		//PrivateDataFolder: "weblio_search",
	})
	engine.OnPrefix("").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		urlName := strings.TrimSpace(ctx.State["args"].(string))
		lowUrlName := strings.ToLower(urlName)
		if strings.Contains(lowUrlName, "dlsite.com") && strings.Contains(lowUrlName, "product_id") {
			proxyURL := "http://127.0.0.1:10809" // 替换为你的代理地址
			proxy, err := url.Parse(proxyURL)
			if err != nil {
				panic(err)
			}
			client := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxy),
				},
				Timeout: 10 * time.Second, // 设置超时时间
			}
			ret, err := getInfoByID(client, urlName)
			if err != nil {
				ctx.SendChain(message.Text(err.Error()))
				return
			}
			doJsonInfo(ctx, ret)
		}
	})
}

func doJsonInfo(ctx *zero.Ctx, info map[string]interface{}) {
	sig := info["success"].(bool)
	if sig != true {
		ctx.SendChain(message.Text("获取发生意外。"))
		return
	}
	data := info["data"].(map[string]interface{})
	PID := data["product_id"].(string)
	title := data["title"].(string)
	circle := data["circle"].(string)
	staff := data["staff"].(string)
	description := data["description"].(string)

	ret := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", PID, title, circle, staff, description)
	img := data["image"].(string)
	imgUrl := "https://" + img[2:]
	//fmt.Println(ret)
	//fmt.Println(imgUrl)

	result, err := fctext.RenderToBase64(ret, fctext.FontFile, 400, 20)
	if err != nil {
		ctx.SendChain(message.Text(err.Error()))
	}
	ctx.SendChain(message.Image("base64://" + binary.BytesToString(result)))
	ctx.SendChain(message.Image(imgUrl))
}

func getInfoByID(client *http.Client, urlStr string) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"success": false,
		"error":   "",
		"data":    nil,
	}
	doc, err := doRequest(client, urlStr)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	extractBasicInfo(doc, data, urlStr)
	extractDescription(doc, data)
	extractStaff(doc, data)

	result["success"] = true
	result["data"] = data
	return result, nil
}

func doRequest(client *http.Client, urlStr string) (*goquery.Document, error) {
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	//
	//req.Header.Set("Accept-Language", "ja,en-US;q=0.9"

	var resp *http.Response
	maxRetries := 3
	var err error
	for i := 0; i < maxRetries; i++ {
		resp, err = client.Do(req) // 注意这里使用=而不是:=
		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	// 错误处理优先
	if err != nil {
		errMsg := errors.New("请求失败: " + err.Error())
		return nil, errMsg
	}
	if resp == nil {
		errMsg := errors.New("所有重试均失败")
		return nil, errMsg
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != 200 {
		errMsg := errors.New(fmt.Sprintf("状态码异常: %d", resp.StatusCode))
		return nil, errMsg
	}

	// 安全读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf("读取响应失败: %v", err))
		return nil, errMsg
	}

	// 安全解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		errMsg := errors.New(fmt.Sprintf("HTML解析失败: %v", err))
		return nil, errMsg
	}
	return doc, nil

}

func extractBasicInfo(doc *goquery.Document, data map[string]interface{}, urlStr string) {
	// 产品ID
	if u, err := url.Parse(urlStr); err == nil {
		parts := strings.Split(u.Path, "/")
		if len(parts) > 0 {
			data["product_id"] = strings.TrimSuffix(parts[len(parts)-1], ".html")
		}
	}

	// 标题
	doc.Find("h1#work_name").Each(func(i int, s *goquery.Selection) {
		data["title"] = strings.TrimSpace(s.Text())
	})

	// 制作商
	doc.Find(".maker_name").Each(func(i int, s *goquery.Selection) {
		data["circle"] = strings.TrimSpace(s.Text())
	})
	// img
	doc.Find(".product-slider-data").Each(func(i int, s *goquery.Selection) {
		//data["image"] = strings.TrimSpace(s.Text())
		img, ex := s.Find("div").Attr("data-src")
		if ex {
			data["image"] = img
		}
	})
}

func extractDescription(doc *goquery.Document, data map[string]interface{}) {
	details := ""
	var err error
	doc.Find(".work_parts_container").Each(func(i int, s *goquery.Selection) {
		details, err = fmtText(s.Text())
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	})

	data["description"] = details
}

func extractStaff(doc *goquery.Document, data map[string]interface{}) {
	staff := ""
	var err error
	doc.Find("table#work_outline").Each(func(i int, s *goquery.Selection) {
		staff, err = fmtText(s.Text())
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	})
	data["staff"] = staff
}

func fmtText(text string) (string, error) {
	ret := ""
	scanner := bufio.NewScanner(strings.NewReader(text))

	// 设置分隔符为换行符，这样Scanner会在每行结束时停止
	scanner.Split(bufio.ScanLines)

	// 逐行读取并打印
	for scanner.Scan() {
		line := scanner.Text() // 获取当前行的内容
		//fmt.Println(line)      // 打印当前行的内容
		line = strings.Replace(line, "  ", "", -1)
		if len(line) > 0 {
			//fmt.Println(line)
			ret += line + "\n"
		}
	}

	// 检查是否发生错误
	if err := scanner.Err(); err != nil {
		errMsg := errors.New("读取时发生错误:" + err.Error())
		return "", errMsg
	}
	return ret, nil
}
