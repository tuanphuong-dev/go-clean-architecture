package domain

type RoleID string

const (
	RoleIDSuperAdmin RoleID = "super_admin"
	RoleIDAdmin      RoleID = "admin"
	RoleIDUser       RoleID = "user"
	RoleIDGuest      RoleID = "guest"
)

type Role struct {
	ID          RoleID `json:"id" gorm:"type:varchar(20);primary_key"`
	Name        string `json:"name" gorm:"type:varchar(50);not null"`
	Description string `json:"description" gorm:"type:varchar(255)"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime:milli"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime:milli"`
	DeletedAt   int64  `json:"deleted_at" gorm:"index"`
}
