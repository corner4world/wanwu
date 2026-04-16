package util

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/UnicomAI/wanwu/pkg/log"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ZipDir 将目录打包为 zip 格式数据。
// srcDir: 源目录路径，支持两种模式：
//   - "/path/to/dir"：包含最后一级目录名，zip 内容为 "dir/file1.txt"
//   - "/path/to/dir/."：不包含最后一级目录名，zip 内容为 "file1.txt"
func ZipDir(srcDir string) ([]byte, error) {
	var buf bytes.Buffer
	err := zipDirCore(srcDir, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ZipDirToLocal 将目录打包为 zip 格式文件。
// srcDir: 源目录路径，支持两种模式：
//   - "/path/to/dir"：包含最后一级目录名，zip 内容为 "dir/file1.txt"
//   - "/path/to/dir/."：不包含最后一级目录名，zip 内容为 "file1.txt"
//
// destZipPath: 目标 zip 文件路径，例如: "/path/to/output.zip"
func ZipDirToLocal(srcDir, destZipPath string) error {
	// 创建目标 zip 文件
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("create zip file error: %w", err)
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			log.Errorf("close zip file error: %v", err)
		}
	}()
	return zipDirCore(srcDir, zipFile)
}

func UnzipDir(ctx context.Context, localFilePath string, destDir string) (extractDir string, err error) {
	fileReader, err := zip.OpenReader(localFilePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err1 := fileReader.Close(); err1 != nil {
			log.Errorf("UnzipDir file (%s) close error: %v", localFilePath, err1)
		}
	}()

	for _, f := range fileReader.File {
		var decodeFileName string
		if f.Flags == 0 { //本地编码，默认GBK，转换成UTF-8
			i := bytes.NewReader([]byte(f.Name))
			decoder := transform.NewReader(i, simplifiedchinese.GB18030.NewDecoder())
			content, _ := io.ReadAll(decoder)
			decodeFileName = string(content)
		} else {
			decodeFileName = f.Name
		}
		// 构建完整的文件路径
		destFilePath := filepath.Join(destDir, decodeFileName)
		// 检查是否为目录
		if f.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(destFilePath, f.Mode()); err != nil {
				log.Errorf("UnzipDir create directory (%s) error: %v", destFilePath, err)
			}
			continue
		}
		// 我们需要确保所有的文件夹都已经创建好
		err = os.MkdirAll(filepath.Dir(destFilePath), f.Mode())
		if err != nil {
			return "", err
		}
		//写入文件
		err = writeUnzipFile(f, destFilePath)
		if err != nil {
			return "", err
		}
	}
	return destDir, nil
}

func zipDirCore(srcDir string, writer io.Writer) error {
	zipWriter := zip.NewWriter(writer)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Errorf("close zip writer error: %v", err)
		}
	}()

	skipBase := strings.HasSuffix(srcDir, string(os.PathSeparator)+".")
	srcDir = filepath.Clean(srcDir)

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", srcDir)
	}

	var baseName string
	if skipBase {
		baseName = ""
	} else {
		baseName = filepath.Base(srcDir)
	}

	err := filepath.Walk(srcDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return fmt.Errorf("get relative path failed: %w", err)
		}

		var zipPath string
		if relPath == "." {
			if skipBase {
				return nil
			}
			zipPath = baseName + "/"
		} else {
			if skipBase {
				zipPath = filepath.ToSlash(relPath)
			} else {
				zipPath = filepath.ToSlash(filepath.Join(baseName, relPath))
			}
		}

		if info.IsDir() {
			zipPath += "/"
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("create zip header failed: %w", err)
		}
		header.Name = zipPath
		header.Method = zip.Store

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("create zip entry failed: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open file failed: %w", err)
		}
		_, err = io.Copy(writer, file)
		_ = file.Close()
		if err != nil {
			return fmt.Errorf("write file content failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("walk directory failed: %w", err)
	}
	return nil
}

// writeUnzipFile 写入文件
func writeUnzipFile(zipFile *zip.File, destFilePath string) error {
	//打开目标文件
	destFile, err := os.OpenFile(destFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.Errorf("writeUnzipFile file close error: %v", err)
		}
	}()

	//打开源压缩文件
	sourceFile, err := zipFile.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Errorf("writeUnzipFile file close error: %v", err)
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
}
