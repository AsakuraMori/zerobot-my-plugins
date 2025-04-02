package animeProjectHelper

import (
	"encoding/json"
	"errors"
	"fmt"
	ctrl "github.com/FloatTech/zbpctrl"
	"github.com/FloatTech/zbputils/control"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
	"os"
	"path/filepath"
	"strings"
)

type animeProjectStruct struct {
	ProjectSeriesName string `json:"ProjectSeriesName"`
	ProjectName       string `json:"ProjectName"`

	//状态
	ProjectState string `json:"ProjectState"`

	//管理人员
	//统筹
	ProjectAdminName string `json:"ProjectAdminName"`
	//时轴
	ProjectTimeline string `json:"ProjectTimeline"`
	//初翻
	ProjectTranslatorName string `json:"ProjectTranslatorName"`
	//校对
	ProjectEditorName string `json:"ProjectEditorName"`
	//特效
	ProjectEffectorName string `json:"ProjectEffectorName"`
	//压制
	ProjectEncoderName string `json:"ProjectEncoderName"`
	//备注
	Remark string `json:"Remark"`
}

var animeProject animeProjectStruct

func init() {
	engine := control.Register("animeProjectHelper", &ctrl.Options[*zero.Ctx]{
		DisableOnDefault: false,
		// 插件的简介
		Brief:             "animeProjectHelper",
		Help:              "- #新建任务 [系列名|项目名]\n- #分配属性 [系列名|项目名]\n- #查询所有任务\n- #查看任务 [系列名|项目名]\n- #删除任务 [系列名|项目名]",
		PrivateDataFolder: "animeProject",
	})
	engine.OnPrefix("#新建任务").SetBlock(true).Handle(func(ctx *zero.Ctx) {

		projectName := strings.TrimSpace(ctx.State["args"].(string))
		if len(projectName) == 0 {
			ctx.SendChain(message.Text("animeInitProject: 项目名为空"))
			return
		}
		projectCmd := strings.Split(projectName, "|")
		if len(projectCmd) != 2 {
			ctx.SendChain(message.Text("animeInitProject: 获取系列名/项目名失败，可能是语法出错。"))
			return
		}
		dirName := engine.DataFolder()
		projectNewErr := animeProject.animeInitProject(ctx, dirName, projectCmd[0], projectCmd[1])
		if projectNewErr != nil {
			ctx.SendChain(message.Text(projectNewErr.Error()))
		}

	})
	engine.OnPrefix("#分配属性").SetBlock(true).Handle(func(ctx *zero.Ctx) {

		projectName := strings.TrimSpace(ctx.State["args"].(string))
		if len(projectName) == 0 {
			ctx.SendChain(message.Text("projectNameError: 项目名为空"))
			return
		}
		projectCmd := strings.Split(projectName, "|")
		if len(projectCmd) != 2 {
			ctx.SendChain(message.Text("projectNameError: 获取系列名/项目名失败，可能是语法出错。"))
			return
		}
		ctx.SendChain(message.Text("请修改项目属性。格式：【#属性配置 成员:人名】\n例如：#属性配置 时轴:咩啊|初翻:kaze"))
		ctx.SendChain(message.Text("成员只支持以下字符串：\n\t状态\n\t统筹\n\t时轴\n\t初翻\n\t校对\n\t特效\n\t压制\n\t备注"))

		engine.OnPrefix("#属性配置").SetBlock(true).Handle(func(ctx *zero.Ctx) {
			fieldCmd := strings.TrimSpace(ctx.State["args"].(string))
			if len(fieldCmd) == 0 {
				ctx.SendChain(message.Text("setFieldInProject: 成员为空"))
				return
			}
			dirName := engine.DataFolder()
			getErr := animeProject.setFieldInProject(ctx, dirName, projectCmd[0], projectCmd[1], fieldCmd)
			if getErr != nil {
				ctx.SendChain(message.Text("setFieldInProject: ", getErr))
				return
			}
		})
	})
	engine.OnPrefix("#查询所有任务").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		err := searchProjectName(ctx, engine.DataFolder())
		if err != nil {
			ctx.SendChain(message.Text(err.Error()))
		}
	})
	engine.OnPrefix("#查看任务").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		projectName := strings.TrimSpace(ctx.State["args"].(string))
		if len(projectName) == 0 {
			ctx.SendChain(message.Text("getProjectInfo: 项目名为空"))
			return
		}
		projectCmd := strings.Split(projectName, "|")
		if len(projectCmd) != 2 {
			ctx.SendChain(message.Text("getProjectInfo: 获取系列名/项目名失败，可能是语法出错。"))
			return
		}
		dirName := engine.DataFolder()
		projectNewErr := animeProject.getProjectInfo(ctx, dirName, projectCmd[0], projectCmd[1])
		if projectNewErr != nil {
			ctx.SendChain(message.Text(projectNewErr.Error()))
		}
	})
	engine.OnPrefix("#删除任务").SetBlock(true).Handle(func(ctx *zero.Ctx) {
		projectName := strings.TrimSpace(ctx.State["args"].(string))
		if len(projectName) == 0 {
			ctx.SendChain(message.Text("deleteProject: 项目名为空"))
			return
		}
		projectCmd := strings.Split(projectName, "|")
		if len(projectCmd) != 2 {
			ctx.SendChain(message.Text("deleteProject: 获取系列名/项目名失败，可能是语法出错。"))
			return
		}
		dirName := engine.DataFolder()
		projectNewErr := deleteProject(ctx, dirName, projectCmd[0], projectCmd[1])
		if projectNewErr != nil {
			ctx.SendChain(message.Text(projectNewErr.Error()))
		}
	})
}

func (aps *animeProjectStruct) animeInitProject(ctx *zero.Ctx, dirName, projectSeriesName, projectName string) error {
	aps.ProjectSeriesName = projectSeriesName
	aps.ProjectName = projectName
	aps.ProjectState = ""
	aps.ProjectAdminName = ""
	aps.ProjectTimeline = ""
	aps.ProjectTranslatorName = ""
	aps.ProjectEditorName = ""
	aps.ProjectEffectorName = ""
	aps.ProjectEncoderName = ""
	aps.Remark = ""

	jsonBuffer, jsonErr := json.MarshalIndent(aps, "", "\t")
	if jsonErr != nil {
		return jsonErr
	}
	jsonFileName := dirName + projectSeriesName + "-" + projectName + ".json"
	jsonFile, err := os.Create(jsonFileName)
	if err != nil {
		return err
	}

	_, jsonFileErr := jsonFile.Write(jsonBuffer)
	if jsonFileErr != nil {
		return jsonFileErr
	}
	jsonclsErr := jsonFile.Close()
	if jsonclsErr != nil {
		return jsonclsErr
	}

	info := fmt.Sprintf("系列名：%s\n项目名：%s\n状态：%s\n统筹：%s\n时轴：%s\n初翻：%s\n校对：%s\n特效：%s\n压制：%s\n备注：%s",
		aps.ProjectSeriesName, aps.ProjectName, aps.ProjectState, aps.ProjectAdminName,
		aps.ProjectTimeline, aps.ProjectTranslatorName, aps.ProjectEditorName,
		aps.ProjectEffectorName, aps.ProjectEncoderName, aps.Remark)

	ctx.SendChain(message.Text(info))

	return nil
}

func (aps *animeProjectStruct) setFieldInProject(ctx *zero.Ctx, dirName, pSeriesName, pProjectName string, cmd string) error {

	//Parse command
	scmd := strings.Split(cmd, "|")
	var cmdPool = make(map[string]string)
	for i := 0; i < len(scmd); i++ {
		ccmd := strings.Split(scmd[i], ":")
		cmdPool[ccmd[0]] = ccmd[1]
	}
	jsonFileName := dirName + pSeriesName + "-" + pProjectName + ".json"
	jsonfile, jsonErr := os.Open(jsonFileName)
	if jsonErr != nil {
		return jsonErr
	}
	jsonInfo, err1 := os.Stat(jsonFileName)
	if err1 != nil {
		return err1
	}
	jsonSize := jsonInfo.Size()

	jsonData := make([]byte, jsonSize)

	_, readErr := jsonfile.Read(jsonData)
	if readErr != nil {
		return readErr
	}
	jsonClsErr1 := jsonfile.Close()
	if jsonClsErr1 != nil {
		return jsonClsErr1
	}
	jsonNodeErr2 := json.Unmarshal(jsonData, &aps)
	if jsonNodeErr2 != nil {
		return jsonNodeErr2
	}
	for k, v := range cmdPool {
		switch k {
		case "状态":
			aps.ProjectState = v
		case "统筹":
			aps.ProjectAdminName = v
		case "时轴":
			aps.ProjectTimeline = v
		case "初翻":
			aps.ProjectTranslatorName = v
		case "校对":
			aps.ProjectEditorName = v
		case "特效":
			aps.ProjectEffectorName = v
		case "压制":
			aps.ProjectEncoderName = v
		case "备注":
			aps.Remark = v
		default:
			slErr := errors.New("不匹配的成员：" + k)
			return slErr
		}
	}
	jsonData2, err2 := json.MarshalIndent(aps, "", "\t")
	if err2 != nil {
		return err2
	}
	jsonFile, err := os.Create(jsonFileName)
	if err != nil {
		return err
	}

	_, jsonFileErr := jsonFile.Write(jsonData2)
	if jsonFileErr != nil {
		return jsonFileErr
	}
	jsonclsErr := jsonFile.Close()
	if jsonclsErr != nil {
		return jsonclsErr
	}

	info := fmt.Sprintf("系列名：%s\n项目名：%s\n状态：%s\n统筹：%s\n时轴：%s\n初翻：%s\n校对：%s\n特效：%s\n压制：%s\n备注：%s",
		aps.ProjectSeriesName, aps.ProjectName, aps.ProjectState, aps.ProjectAdminName,
		aps.ProjectTimeline, aps.ProjectTranslatorName, aps.ProjectEditorName,
		aps.ProjectEffectorName, aps.ProjectEncoderName, aps.Remark)

	ctx.SendChain(message.Text(info))
	return nil

}

func (aps *animeProjectStruct) getProjectInfo(ctx *zero.Ctx, dirName, pSeriesName, pProjectName string) error {
	//Parse command
	jsonFileName := dirName + pSeriesName + "-" + pProjectName + ".json"
	jsonfile, jsonErr := os.Open(jsonFileName)
	if jsonErr != nil {
		return jsonErr
	}
	jsonInfo, err1 := os.Stat(jsonFileName)
	if err1 != nil {
		return err1
	}
	jsonSize := jsonInfo.Size()

	jsonData := make([]byte, jsonSize)

	_, readErr := jsonfile.Read(jsonData)
	if readErr != nil {
		return readErr
	}
	jsonClsErr1 := jsonfile.Close()
	if jsonClsErr1 != nil {
		return jsonClsErr1
	}
	jsonNodeErr2 := json.Unmarshal(jsonData, &aps)
	if jsonNodeErr2 != nil {
		return jsonNodeErr2
	}
	info := fmt.Sprintf("系列名：%s\n项目名：%s\n状态：%s\n统筹：%s\n时轴：%s\n初翻：%s\n校对：%s\n特效：%s\n压制：%s\n备注：%s",
		aps.ProjectSeriesName, aps.ProjectName, aps.ProjectState, aps.ProjectAdminName,
		aps.ProjectTimeline, aps.ProjectTranslatorName, aps.ProjectEditorName,
		aps.ProjectEffectorName, aps.ProjectEncoderName, aps.Remark)
	ctx.SendChain(message.Text(info))
	return nil
}

func deleteProject(ctx *zero.Ctx, dirName, pSeriesName, pProjectName string) error {
	jsonFileName := dirName + pSeriesName + "-" + pProjectName + ".json"
	err := os.Remove(jsonFileName)
	if err != nil {
		return err
	}
	info := "成功删除任务：" + pSeriesName + "-" + pProjectName

	ctx.SendChain(message.Text(info))
	return nil
}

func searchProjectName(ctx *zero.Ctx, dirName string) error {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		ctx.SendChain(message.Text("没有项目信息"))
		return nil
	}
	info := "当前进行的项目有：\n"
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext == ".json" {
			info += entry.Name()[:len(entry.Name())-len(".json")] + "\n"
		}
	}
	ctx.SendChain(message.Text(info))
	return nil
}
