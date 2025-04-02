package weblio_search

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
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const PID = "http://127.0.0.1:10809"

func init() {
	engine := control.Register("weblio_search", &ctrl.Options[*zero.Ctx]{
		DisableOnDefault: false,
		// 插件的简介
		Brief: "weblio_search",
		Help:  "- /搜索 [标签]",
		//PrivateDataFolder: "weblio_search",
	})
	engine.OnPrefix("/日语搜索").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		wordName := strings.TrimSpace(ctx.State["args"].(string))
		err := searchFromString(ctx, wordName)
		if err != nil {
			ctx.SendChain(message.Text(err.Error()))
		}
	})
}

func removeRichTextFormat(text string) string {
	// 1. 去除HTML标签
	reHTML := regexp.MustCompile(`<[^>]*>`)
	text = reHTML.ReplaceAllString(text, "")

	// 2. 去除Markdown标记（标题、链接等）
	// 去除###标题

	// 去除Markdown链接 [text](url)
	reMDLink := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	text = reMDLink.ReplaceAllString(text, "$1")

	// 合并多个空白为单个空格
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")

	// 去除首尾空白
	text = strings.TrimSpace(text)

	return text
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
		line = strings.Replace(line, "\t", "", -1)
		if len(line) > 0 &&
			!strings.HasPrefix(line, "<img") &&
			!strings.Contains(line, "※ご利用のPCやブラウザにより") &&
			!strings.Contains(line, "Copyright © KANJIDIC2") {
			//fmt.Println(line)
			ret += removeRichTextFormat(line) + "\n"
		}
	}
	// 检查是否发生错误
	if err := scanner.Err(); err != nil {
		errMsg := errors.New("读取时发生错误:" + err.Error())
		return "", errMsg
	}

	return ret, nil
}

func searchFromString(ctx *zero.Ctx, text string) error {
	// 要搜索的字符串
	searchQuery := text

	// 构建Weblio搜索URL
	searchURL := fmt.Sprintf("https://www.weblio.jp/content/%s", url.QueryEscape(searchQuery))

	/*	proxyURL, err := url.Parse(PID) // 替换为你的代理地址
		if err != nil {
			errMsg := errors.New(err.Error())
			return errMsg
		}*/
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			//Proxy:                 http.ProxyURL(proxyURL),
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")

	// 重试机制
	var resp *http.Response
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		var errMsg error
		if resp.StatusCode == 404 {
			errMsg = errors.New("找不到指定条目。")
		} else {
			errMsg = errors.New("不正常的响应码: " + strconv.Itoa(resp.StatusCode))
		}
		return errMsg
	}

	// 解析HTML文档
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		errMsg := errors.New(err.Error())
		return errMsg
	}

	// 提取所需的信息
	var results []map[string]interface{}
	discNames := []string{}

	doc.Find("div .kiji").Each(func(i int, s *goquery.Selection) {
		discName, err := fmtText(s.Text())
		if err != nil {
			return
		}
		discNames = append(discNames, discName)
	})
	doc.Find(".pbarTL").Each(func(i int, s *goquery.Selection) {
		dicName, err := fmtText(s.Text())
		if err != nil {
			return
		}
		item := map[string]interface{}{
			"dict": dicName,
			"data": discNames[i],
		}
		results = append(results, item)

	})
	retStr := ""
	for v := range results {
		dictInV := results[v]["dict"].(string)
		if dictInV == "Weblio日本語例文用例辞書\n" || dictInV == "ウィキペディア\n" {
			continue
		}
		dataInV := results[v]["data"].(string)
		retStr += "===========================\n"
		retStr += "### " + dictInV + "\n"
		retStr += dataInV + "\n"

	}
	data, err := fctext.RenderToBase64(retStr, fctext.FontFile, 400, 20)
	if err != nil {
		return err
	}
	ctx.SendChain(message.Image("base64://" + binary.BytesToString(data)))
	return nil
}
