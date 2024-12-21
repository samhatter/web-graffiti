package main

type DirectoryTransfer struct {
	Directory string `json:"directory"`
	FileCount int `json:"fileCount"`
	Files []FileTransfer `json:"files"`
}

type FileTransfer struct {
	Id string `json:"id"`
	FileName string `json:"filename"`
	State string `json:"state"`
	Size int `json:"size"`
	RequestedAt string `json:"requestedAt"`
	UserName string `json:"username"`
	Direction string `json:"direction"`
}

type UserTransfer struct {
	Username string `json:"username"`
	Directories []DirectoryTransfer `json:"directories"`
}

type SearchRequest struct {
	Id string `json:"id"`
	SearchText string `json:"searchText"`
}

type SearchFile struct {
	FileName string `json:"filename"`
	Size int `json:"size"`
	IsLocked bool `json:"isLocked"`
}

type SearchUserFiles struct {
	FileCount int `json:"fileCount"`
	Files []SearchFile `json:"files"`
	UserName string `json:"username"`
}

type SearchStatus struct {
	IsComplete bool `json:"isComplete"`
}

type DownloadRequest struct {
	FileName string `json:"filename"`
	Size int `json:"size"`
}

type DirectoryShare struct {
	Name string `json:"name"`
	FileCount int `json:"filecount"`
	Files []FileShare `json:"files"`
}

type FileShare struct {
	FileName string `json:"filename"`
	Size int `json:"size"`
}
