package service

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	minio_service "github.com/UnicomAI/wanwu/internal/agent-service/service/minio-service"
	"github.com/UnicomAI/wanwu/pkg/util"
)

const (
	baseSkillDir = "tmp/skills"
)

type SkillDir struct {
	SkillDir  string //技能所在地址
	OutputDir string //此次运行技能输出地址
	InputDir  string //此次运行技能输入地址
}

// CreateSkillDir 创建技能运行目录
func CreateSkillDir(runId string, skill *request.SkillToolInfo, uploadFile []string, skillParams *SkillParams) (*SkillDir, error) {
	runDir := baseSkillDir + "/" + runId
	skillDir, err := buildSkillDir(skill)
	if err != nil {
		return nil, err
	}
	//创建inputDir
	var inputDir = runDir + "/inputDir"
	if err := util.MkDir(inputDir); err != nil {
		return nil, err
	}
	if len(uploadFile) > 0 {
		for _, file := range uploadFile {
			err := minio_service.DownloadFileToLocal(context.Background(), file, inputDir+util.NewRandomFile(file))
			if err != nil {
				return nil, err
			}
		}
	}
	//下载输入文件
	downloadInputFile(inputDir, skillParams)
	//创建outputDir
	var outputDir = runDir + "/outputDir"
	if err := util.MkDir(outputDir); err != nil {
		return nil, err
	}
	return &SkillDir{
		SkillDir:  skillDir,
		OutputDir: outputDir,
		InputDir:  inputDir,
	}, nil
}

// 构建skill目录
func buildSkillDir(skill *request.SkillToolInfo) (string, error) {
	if skill.SkillType == request.SkillTypeBuiltIn {
		return skill.ObjectPath, nil
	}
	return buildCustomSkillDir(skill)
}

// 构建自定义skill目录
func buildCustomSkillDir(skill *request.SkillToolInfo) (string, error) {
	var skillTempDir = baseSkillDir + "/" + skill.SkillId
	exist, err := util.FileExist(skillTempDir)
	if err != nil {
		return "", err
	}
	var skillDir string
	if exist {
		skillDir = skillTempDir
	} else {
		unzipSkill, err := downloadAndUnzipSkill(skillTempDir, skill.ObjectPath)
		if err != nil {
			return "", err
		}
		skillDir = unzipSkill
	}

	fileList, _ := util.DirFileList(skillDir, true, true)
	if len(fileList) > 0 {
		for _, file := range fileList {
			if strings.ToLower(filepath.Base(file)) == "skill.md" {
				return filepath.Dir(file), nil
			}
		}
	}
	return skillDir, nil
}

// 下载并解压skill
func downloadAndUnzipSkill(skillTempDir, skillUrl string) (string, error) {
	//其实这个minio-wanwu的前缀没有实际作用只是为了保持一个http链接格式
	var localFilePath = skillTempDir + "/" + filepath.Base(skillUrl)
	err := minio_service.DownloadFileToLocal(context.Background(), "http://minio-wanwu:9000/"+skillUrl, localFilePath)
	if err != nil {
		return "", err
	}
	return util.UnzipDir(context.Background(), localFilePath, skillTempDir)
}
