package usecase

import (
	"context"
	"errors"
	"go-clean-arch/domain"

	"golang.org/x/crypto/bcrypt"
)

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hashed, password string) bool
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, userID string, option *domain.FindOneOption) (*domain.User, error)
	FindOne(ctx context.Context, filter *domain.UserFilter, option *domain.FindOneOption) (*domain.User, error)
	FindMany(ctx context.Context, filter *domain.UserFilter, option *domain.FindManyOption) ([]*domain.User, error)
	FindPage(ctx context.Context, filter *domain.UserFilter, option *domain.FindPageOption) ([]*domain.User, *domain.Pagination, error)
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, userID string, newPassword string) error
	Delete(ctx context.Context, userID string) error
	Count(ctx context.Context, filter *domain.UserFilter) (int64, error)
}

type userUsecase struct {
	repo   UserRepository
	hasher Hasher
}

func NewUserUsecase(repo UserRepository, hasher Hasher) domain.UserUsecase {
	return &userUsecase{repo: repo, hasher: hasher}
}

func (u *userUsecase) Create(ctx context.Context, req *domain.UserCreateRequest) (*domain.User, error) {
	user := &domain.User{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Status:    domain.UserSTTWaitingVerify,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingByEmail, err := u.repo.FindOne(ctx, &domain.UserFilter{
		Email: &user.Email,
	}, nil)
	if err != nil && !errors.Is(err, domain.ErrRecordNotFound) {
		return nil, domain.ErrInternalServerError.WithError(err.Error())
	}

	if existingByEmail != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	// Hash password and create user
	hashedPassword, err := u.hasher.Hash(user.Password)
	if err != nil {
		return nil, domain.ErrPasswordHashFailed.WithWrap(err)
	}

	user.Password = hashedPassword
	if err := u.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *userUsecase) FindByID(ctx context.Context, userID string, option *domain.FindOneOption) (*domain.User, error) {
	user, err := u.repo.FindByID(ctx, userID, option)
	if err != nil {
		return nil, domain.ErrUserNotFound.WithWrap(err)
	}
	return user, nil
}

func (u *userUsecase) FindByEmail(ctx context.Context, email string, option *domain.FindOneOption) (*domain.User, error) {
	user, err := u.repo.FindOne(ctx, &domain.UserFilter{Email: &email}, option)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound.WithWrap(err)
	}
	return user, nil
}

func (u *userUsecase) FindOne(ctx context.Context, filter *domain.UserFilter, option *domain.FindOneOption) (*domain.User, error) {
	user, err := u.repo.FindOne(ctx, filter, option)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound.WithWrap(err)
	}
	return user, nil
}

func (u *userUsecase) Update(ctx context.Context, userID string, req *domain.UserUpdateRequest) error {
	user, err := u.repo.FindByID(ctx, userID, nil)
	if err != nil || user == nil {
		return domain.ErrUserNotFound.WithWrap(err)
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if err := user.Validate(); err != nil {
		return err
	}
	return u.repo.Update(ctx, user)
}

func (u *userUsecase) ChangePassword(ctx context.Context, req *domain.UserChangePasswordRequest) error {
	user, err := u.repo.FindByID(ctx, req.UserID, nil)
	if err != nil || user == nil {
		return domain.ErrUserNotFound.WithWrap(err)
	}
	// Verify old password (hash check)
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)) != nil {
		return domain.ErrInvalidCredentials.WithError("old password is incorrect")
	}
	// Hash new password before saving
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return domain.ErrPasswordHashFailed.WithWrap(err)
	}
	return u.repo.UpdatePassword(ctx, req.UserID, string(hashed))
}

func (u *userUsecase) FindPage(ctx context.Context, filter *domain.UserFilter, option *domain.FindPageOption) ([]*domain.User, *domain.Pagination, error) {
	return u.repo.FindPage(ctx, filter, option)
}
