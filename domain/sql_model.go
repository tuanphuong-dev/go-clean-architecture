package domain

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

type SQLModel struct {
	ID        string `json:"id" gorm:"type:varchar(36);primary_key;default:gen_random_uuid()"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime:milli"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime:milli"`
	DeletedAt int64  `json:"deleted_at" gorm:"index"`
}

type FindOneOption struct {
	Preloads []string `json:"preloads" form:"preloads"`
	Sort     []string `json:"sort" form:"sort"`
}

type FindManyOption struct {
	Preloads []string `json:"preloads" form:"preloads"`
	Sort     []string `json:"sort" form:"sort"`
	Limit    *int     `json:"limit" form:"limit" default:"10"`
	Offset   *int     `json:"offset" form:"offset" default:"0"`
}

type FindPageOption struct {
	Preloads []string `json:"preloads" form:"preloads"`
	Sort     []string `json:"sort" form:"sort"`
	Page     int      `json:"page" form:"page" default:"1"`
	PerPage  int      `json:"per_page" form:"per_page" default:"10"`
}

type JSONB map[string]any

func (j JSONB) Value() (driver.Value, error) {
	val, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(val), nil
}

func (j *JSONB) Scan(input interface{}) error {
	b, ok := input.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, j)
}

type StringSlice []string

func NewStringSlice(s []string) StringSlice {
	return StringSlice(s)
}

func (s StringSlice) Value() (driver.Value, error) {
	val, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(val), nil
}

func (s *StringSlice) Scan(input interface{}) error {
	b, ok := input.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, s)
}
