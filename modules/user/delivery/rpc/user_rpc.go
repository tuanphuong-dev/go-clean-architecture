package rpc

import (
	"context"
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"go-clean-arch/proto/pb"
)

type UserRPC struct {
	pb.UnimplementedUserServiceServer
	usecase domain.UserUsecase
}

func NewUserRPC(usecase domain.UserUsecase) *UserRPC {
	return &UserRPC{usecase: usecase}
}

func (s *UserRPC) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	createReq := &domain.UserCreateRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}
	user, err := s.usecase.Create(ctx, createReq)
	if err != nil {
		return nil, common.ToGRPCError(err)
	}
	return &pb.CreateUserResponse{User: common.ToPbUser(user)}, nil
}

func (s *UserRPC) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.GetUserResponse, error) {
	user, err := s.usecase.FindByID(ctx, req.Id, nil)
	if err != nil || user == nil {
		return nil, err
	}
	return &pb.GetUserResponse{User: common.ToPbUser(user)}, nil
}

func (s *UserRPC) GetUserByFilter(ctx context.Context, req *pb.GetUserByFilterRequest) (*pb.GetUserResponse, error) {
	filter := common.ToDomainUserFilter(req.Filter)
	option := common.ToDomainFindOneOption(req.Option)
	user, err := s.usecase.FindOne(ctx, filter, option)
	if err != nil || user == nil {
		return nil, err
	}
	return &pb.GetUserResponse{User: common.ToPbUser(user)}, nil
}

func (s *UserRPC) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	updateReq := &domain.UserUpdateRequest{}
	if req.Username != "" {
		updateReq.Username = &req.Username
	}
	if req.Email != "" {
		updateReq.Email = &req.Email
	}
	if req.FirstName != "" {
		updateReq.FirstName = &req.FirstName
	}
	if req.LastName != "" {
		updateReq.LastName = &req.LastName
	}
	st := domain.UserStatus(req.Status.String())
	if req.Status != pb.UserStatus_USER_STATUS_UNSPECIFIED {
		updateReq.Status = &st
	}
	err := s.usecase.Update(ctx, req.Id, updateReq)
	if err != nil {
		return nil, err
	}
	user, _ := s.usecase.FindByID(ctx, req.Id, nil)
	return &pb.UpdateUserResponse{User: common.ToPbUser(user)}, nil
}

func (s *UserRPC) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	filter := &domain.UserFilter{}
	option := &domain.FindPageOption{
		Page:    int(req.Page),
		PerPage: int(req.PerPage),
	}
	users, pagination, err := s.usecase.FindPage(ctx, filter, option)
	if err != nil {
		return nil, err
	}
	var pbUsers []*pb.User
	for _, u := range users {
		pbUsers = append(pbUsers, common.ToPbUser(u))
	}
	return &pb.ListUsersResponse{
		Users: pbUsers,
		Total: int32(pagination.TotalItems),
	}, nil
}
