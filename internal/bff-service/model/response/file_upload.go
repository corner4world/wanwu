package response

type CheckFileResp struct {
	Status int `json:"status"` //0:不存在，1：已完成
}

type UploadFileResp struct {
	Status int `json:"status"` //0:上传失败，1：上传成功
}

type MergeFileResp struct {
	OriginalFileName string `json:"originalFileName"` //原始文件名
	FileName         string `json:"fileName"`         //合并后文件名(在minio中的文件名)
	FilePath         string `json:"filePath"`         //minio文件的完整路径
}

type CleanFileResp struct {
	Status int `json:"status"` //0:清除失败，1：已完成
}

type DeleteFileResp struct {
	Status int `json:"status"` //0:删除失败，1：已完成
}

type CheckFileListResp struct {
	UploadedFileSequences []int `json:"uploadedFileSequences"` //已经上传成功的切片文件序号列表
}

type ProxyUploadFileResp struct {
	DownloadLink string `json:"download_link"` //上传文件链接
}

type DirectUploadFilesResp struct {
	Files []*DirectUploadFileInfo `json:"files"`
}

type DirectUploadFileInfo struct {
	FileName string `json:"fileName"` //原始文件名
	FileId   string `json:"fileId"`   //上传文件名
	FilePath string `json:"filePath"` //minio文件的完整路径
	FileSize int64  `json:"fileSize"` //文件大小
}
