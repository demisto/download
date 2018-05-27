package domain

import "time"

type Download struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	SHA256     string    `json:"sha256"`
	GitHash    string    `json:"gitHash" db:"git_hash"`
	ModifyDate time.Time `json:"modifyDate" db:"modify_date"`
}

type DownloadLog struct {
	Username   string    `json:"username"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	ModifyDate time.Time `json:"modifyDate" db:"modify_date"`
}
