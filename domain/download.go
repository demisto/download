package domain

import "time"

type Download struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	ModifyDate time.Time `json:"modifyDate" db:"modify_date"`
}
