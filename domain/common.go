package domain

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

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
	Joins    []string `json:"joins" form:"joins"`
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

type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
}

func NewPagination(page, perPage int, totalItems int64) *Pagination {
	totalPages := int((totalItems + int64(perPage) - 1) / int64(perPage))
	return &Pagination{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		TotalItems: totalItems,
	}
}

type Date time.Time

func (date Date) String() string {
	t := time.Time(date)
	return t.Format("02/01/2006")
}

func (date Date) IsZero() bool {
	t := time.Time(date)
	return t.IsZero()
}

func (date *Date) Scan(value interface{}) (err error) {
	nullTime := &sql.NullTime{}
	err = nullTime.Scan(value)
	*date = Date(nullTime.Time)
	return
}

func (date Date) Value() (driver.Value, error) {
	y, m, d := time.Time(date).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Time(date).Location()), nil
}

func (date Date) GormDataType() string {
	return "date"
}

func (date Date) MarshalJSON() ([]byte, error) {
	t := time.Time(date)
	formatted := t.Format("02/01/2006")
	return []byte(`"` + formatted + `"`), nil
}

func (date *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1]
	t, err := time.Parse("02/01/2006", s)
	if err != nil {
		return fmt.Errorf("date must be dd/mm/yyyy: %w", err)
	}
	*date = Date(t)
	return nil
}
