package client

import (
	"context"
	"go-clean-arch/common"
	"go-clean-arch/domain"
	"go-clean-arch/proto/pb"

	"google.golang.org/grpc"
)

type UserRPCClient struct {
	client pb.UserServiceClient
}

func NewUserRPCClient(conn *grpc.ClientConn) *UserRPCClient {
	return &UserRPCClient{
		client: pb.NewUserServiceClient(conn),
	}
}

func (c *UserRPCClient) Create(ctx context.Context, req *domain.UserCreateRequest) (*domain.User, error) {
	pbReq := &pb.CreateUserRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}
	resp, err := c.client.CreateUser(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return common.ToDomainUser(resp.User), nil
}

func (c *UserRPCClient) FindOne(ctx context.Context, filter *domain.UserFilter, option *domain.FindOneOption) (*domain.User, error) {
	pbFilter := common.ToPbUserFilter(filter)
	pbOption := common.ToPbFindOneOption(option)

	pbReq := &pb.GetUserByFilterRequest{
		Filter: pbFilter,
		Option: pbOption,
	}

	resp, err := c.client.GetUserByFilter(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return common.ToDomainUser(resp.User), nil
}
