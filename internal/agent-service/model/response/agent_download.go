package response

type DownloadFileInfo struct {
	FileName string `json:"fileName"`
	FilePath string `json:"filePath"`
	FileSize int64  `json:"fileSize"`
}

type DownloadContext struct {
	DownloadMap  map[string][]*DownloadFileInfo
	DownloadList [][]*DownloadFileInfo
}

func NewDownloadContext() *DownloadContext {
	return &DownloadContext{
		DownloadMap:  make(map[string][]*DownloadFileInfo),
		DownloadList: make([][]*DownloadFileInfo, 0),
	}
}

func (d *DownloadContext) AddDownloadFile(toolId string, fileList []*DownloadFileInfo) {
	if fileList == nil {
		fileList = make([]*DownloadFileInfo, 0)
	}
	d.DownloadMap[toolId] = fileList
	d.DownloadList = append(d.DownloadList, fileList)
}
