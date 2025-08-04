package common

import (
	"go-clean-arch/domain"
	"go-clean-arch/proto/pb"
)

func ToPbUser(u *domain.User) *pb.User {
	if u == nil {
		return nil
	}
	var status pb.UserStatus
	switch u.Status {
	case domain.UserSTTWaitingVerify:
		status = pb.UserStatus_USER_STATUS_WAITING_VERIFY
	case domain.UserSTTActive:
		status = pb.UserStatus_USER_STATUS_ACTIVE
	case domain.UserSTTBanned:
		status = pb.UserStatus_USER_STATUS_BANNED
	default:
		status = pb.UserStatus_USER_STATUS_UNSPECIFIED
	}
	return &pb.User{
		Id:        u.ID,
		Email:     u.Email,
		Password:  u.Password,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Status:    status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		DeletedAt: u.DeletedAt,
	}
}

func ToDomainUser(u *pb.User) *domain.User {
	if u == nil {
		return nil
	}
	var status domain.UserStatus
	switch u.Status {
	case pb.UserStatus_USER_STATUS_WAITING_VERIFY:
		status = domain.UserSTTWaitingVerify
	case pb.UserStatus_USER_STATUS_ACTIVE:
		status = domain.UserSTTActive
	case pb.UserStatus_USER_STATUS_BANNED:
		status = domain.UserSTTBanned
	default:
		status = domain.UserStatus("")
	}
	return &domain.User{
		SQLModel: domain.SQLModel{
			ID:        u.Id,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
			DeletedAt: u.DeletedAt,
		},
		Email:     u.Email,
		Password:  u.Password,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Status:    status,
	}
}

func ToDomainUserFilter(req *pb.UserFilter) *domain.UserFilter {
	filter := &domain.UserFilter{
		ID:             req.Id,
		IDNe:           req.IdNe,
		IDIn:           req.IdIn,
		Email:          req.Email,
		Username:       req.Username,
		Active:         req.Active,
		Blocked:        req.Blocked,
		HasRoles:       req.HasRoles,
		SearchTerm:     req.SearchTerm,
		SearchFields:   req.SearchFields,
		IncludeDeleted: req.IncludeDeleted,
	}
	return filter
}

func ToPbFindOneOption(opt *domain.FindOneOption) *pb.FindOneOption {
	if opt == nil {
		return nil
	}
	return &pb.FindOneOption{
		Preloads: opt.Preloads,
		Sort:     opt.Sort,
	}
}

func ToDomainFindOneOption(opt *pb.FindOneOption) *domain.FindOneOption {
	if opt == nil {
		return nil
	}
	return &domain.FindOneOption{
		Preloads: opt.Preloads,
		Sort:     opt.Sort,
	}
}

func ToPbUserFilter(filter *domain.UserFilter) *pb.UserFilter {
	if filter == nil {
		return nil
	}
	return &pb.UserFilter{
		Id:             filter.ID,
		IdNe:           filter.IDNe,
		IdIn:           filter.IDIn,
		Email:          filter.Email,
		Username:       filter.Username,
		Active:         filter.Active,
		Blocked:        filter.Blocked,
		HasRoles:       filter.HasRoles,
		SearchTerm:     filter.SearchTerm,
		SearchFields:   filter.SearchFields,
		IncludeDeleted: filter.IncludeDeleted,
	}
}
