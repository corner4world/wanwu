package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildFileTree 构建目录的树形结构字符串。
// dirPath: 目录路径
// maxDepth: 最大遍历深度，<= 0 表示无限制
// showRoot: 是否显示根目录名称
// 返回类似 tree 命令输出的字符串，如：
//
//	dir/
//	├─ file1.txt
//	└─ subdir/
//	   └─ file2.txt
func BuildFileTree(dirPath string, maxDepth int, showRoot bool) string {
	if maxDepth <= 0 {
		maxDepth = -1
	}
	var sb strings.Builder
	if showRoot {
		sb.WriteString(filepath.Base(dirPath) + "/\n")
	}
	buildFileTree(&sb, dirPath, "", maxDepth)
	return sb.String()
}

// buildFileTree 递归构建文件树。
// sb: 字符串构建器，用于累积输出
// dirPath: 当前遍历的目录路径
// prefix: 当前行的前缀缩进（包含连接线）
// remainingDepth: 剩余遍历深度，0 停止，-1 无限制
func buildFileTree(sb *strings.Builder, dirPath string, prefix string, remainingDepth int) {
	if remainingDepth == 0 {
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Fprintf(sb, "Error reading directory %s: %v\n", dirPath, err)
		return
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1
		connector := getFileTreeConnector(isLast)
		name := entry.Name()

		if entry.IsDir() {
			fmt.Fprintf(sb, "%s%s%s/\n", prefix, connector, name)
			if remainingDepth < 0 || remainingDepth > 1 {
				newPrefix := getFileTreeNewPrefix(prefix, isLast)
				newPath := filepath.Join(dirPath, name)
				newDepth := remainingDepth - 1
				if remainingDepth < 0 {
					newDepth = -1
				}
				buildFileTree(sb, newPath, newPrefix, newDepth)
			}
		} else {
			fmt.Fprintf(sb, "%s%s%s\n", prefix, connector, name)
		}
	}
}

// getFileTreeConnector 获取文件树连接符。
// isLast: 是否为当前目录下的最后一项
// 返回 "└─ " (最后一项) 或 "├─ " (非最后一项)
func getFileTreeConnector(isLast bool) string {
	if isLast {
		return "└─ "
	}
	return "├─ "
}

// getFileTreeNewPrefix 计算子目录的前缀缩进。
// prefix: 当前前缀
// isLast: 父目录项是否为最后一项
// 返回新的前缀：父项为最后一项时用空格延续缩进，否则用竖线连接
func getFileTreeNewPrefix(prefix string, isLast bool) string {
	if isLast {
		return prefix + "   "
	}
	return prefix + "│  "
}
