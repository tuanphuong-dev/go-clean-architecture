package domain

import (
	"context"
	"net/http"
)

/****************************
*        User errors        *
****************************/
var (
	ErrUserNotFound = &DetailedError{
		IDField:         "USER_NOT_FOUND",
		StatusDescField: http.StatusText(http.StatusNotFound),
		ErrorField:      "User not found",
		StatusCodeField: http.StatusNotFound,
	}
	ErrEmailAlreadyExists = &DetailedError{
		IDField:         "EMAIL_ALREADY_EXISTS",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "User with this email already exists",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrUsernameAlreadyExists = &DetailedError{
		IDField:         "USERNAME_ALREADY_EXISTS",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "User with this username already exists",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrUserCreationFailed = &DetailedError{
		IDField:         "USER_CREATION_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to create user",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrUserUpdateFailed = &DetailedError{
		IDField:         "USER_UPDATE_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to update user",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrUserDeletionFailed = &DetailedError{
		IDField:         "USER_DELETION_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to delete user",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrUserValidationFailed = &DetailedError{
		IDField:         "USER_VALIDATION_FAILED",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "User validation failed",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrPasswordHashFailed = &DetailedError{
		IDField:         "PASSWORD_HASH_FAILED",
		StatusDescField: http.StatusText(http.StatusInternalServerError),
		ErrorField:      "Failed to hash password",
		StatusCodeField: http.StatusInternalServerError,
	}
	ErrInvalidUserStatus = &DetailedError{
		IDField:         "INVALID_USER_STATUS",
		StatusDescField: http.StatusText(http.StatusBadRequest),
		ErrorField:      "Invalid user status",
		StatusCodeField: http.StatusBadRequest,
	}
	ErrUserInactive = &DetailedError{
		IDField:         "USER_INACTIVE",
		StatusDescField: http.StatusText(http.StatusForbidden),
		ErrorField:      "User account is inactive",
		StatusCodeField: http.StatusForbidden,
	}
	ErrUserBanned = &DetailedError{
		IDField:         "USER_BANNED",
		StatusDescField: http.StatusText(http.StatusForbidden),
		ErrorField:      "User account is banned",
		StatusCodeField: http.StatusForbidden,
	}
)

/***************************************
*       User entities and types       *
***************************************/
type UserStatus string

const (
	UserSTTWaitingVerify UserStatus = "waiting_verify"
	UserSTTActive        UserStatus = "active"
	UserSTTBanned        UserStatus = "banned"
)

type User struct {
	SQLModel
	Email     string     `json:"email" gorm:"type:varchar(100);unique;not null"`
	Password  string     `json:"-" gorm:"type:varchar(60);not null"`
	FirstName string     `json:"first_name" gorm:"type:varchar(50);not null"`
	LastName  string     `json:"last_name" gorm:"type:varchar(50);not null"`
	Status    UserStatus `json:"status" gorm:"type:varchar(20);default:'waiting_verify'"`
	Roles     []*Role    `json:"roles" gorm:"many2many:user_roles;"`
}

func (u *User) Validate() error {
	if u.Email == "" {
		return ErrUserValidationFailed.WithError("email must be not empty")
	}
	if u.FirstName == "" {
		return ErrUserValidationFailed.WithError("first_name must be not empty")
	}
	if u.LastName == "" {
		return ErrUserValidationFailed.WithError("last_name must be not empty")
	}
	switch u.Status {
	case UserSTTWaitingVerify, UserSTTActive, UserSTTBanned:
		// valid
	default:
		return ErrInvalidUserStatus
	}
	return nil
}

func (u *User) HasAnyRole(roleIDs ...RoleID) bool {
	if len(u.Roles) == 0 || len(roleIDs) == 0 {
		return false
	}
	roleIDSet := make(map[RoleID]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		roleIDSet[id] = struct{}{}
	}
	for _, role := range u.Roles {
		if role != nil {
			if _, ok := roleIDSet[role.ID]; ok {
				return true
			}
		}
	}
	return false
}

func (u *User) IsBanned() bool {
	return u.Status == UserSTTBanned
}

func (u *User) IsActive() bool {
	return u.Status == UserSTTActive
}

type UserFilter struct {
	ID             *string  `json:"id" form:"id"`
	IDNe           *string  `json:"id_ne" form:"id_ne"`
	IDIn           []string `json:"id_in" form:"id_in"`
	Email          *string  `json:"email" form:"email"`
	Username       *string  `json:"username" form:"username"`
	Active         *bool    `json:"active" form:"active"`
	Blocked        *bool    `json:"blocked" form:"blocked"`
	HasRoles       []string `json:"has_roles" form:"has_roles"`
	SearchTerm     *string  `json:"search_term" form:"search_term"`
	SearchFields   []string `json:"search_fields" form:"search_fields"`
	IncludeDeleted *bool    `json:"include_deleted" form:"include_deleted"`
}

/**********************************************
*       User usecase interfaces and types      *
**********************************************/
type UserUsecase interface {
	Create(ctx context.Context, req *UserCreateRequest) (*User, error)
	FindByID(ctx context.Context, userID string, option *FindOneOption) (*User, error)
	FindByEmail(ctx context.Context, userID string, option *FindOneOption) (*User, error)
	FindOne(ctx context.Context, filter *UserFilter, option *FindOneOption) (*User, error)
	Update(ctx context.Context, userID string, req *UserUpdateRequest) error
	ChangePassword(ctx context.Context, req *UserChangePasswordRequest) error
	FindPage(ctx context.Context, filter *UserFilter, option *FindPageOption) ([]*User, *Pagination, error)
}

type UserCreateRequest struct {
	Username  string `json:"username" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type UserUpdateRequest struct {
	Username  *string     `json:"username,omitempty"`
	Email     *string     `json:"email,omitempty"`
	FirstName *string     `json:"first_name,omitempty"`
	LastName  *string     `json:"last_name,omitempty"`
	Status    *UserStatus `json:"status,omitempty"`
}

type UserChangePasswordRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}
