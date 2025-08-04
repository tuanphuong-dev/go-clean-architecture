package common

import (
	"errors"
	"go-clean-arch/domain"
	"go-clean-arch/proto/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func IsRecordNotFound(err error) bool {
	return errors.Is(err, domain.ErrRecordNotFound)
}

func ToGRPCError(err error) error {
	if de, ok := err.(*domain.DetailedError); ok {
		st := status.New(codes.InvalidArgument, de.Error())
		st, _ = st.WithDetails(&pb.DetailError{
			Id:      de.IDField,
			Code:    int32(de.StatusCodeField),
			Status:  de.StatusDescField,
			Reason:  de.ReasonField,
			Debug:   de.DebugField,
			Message: de.ErrorField,
			// Details: map[string]string{}, // add if needed
		})
		return st.Err()
	}
	return status.Error(codes.Internal, err.Error())
}

func IsDetailError(err error) (*domain.DetailedError, bool) {
	de, ok := err.(*domain.DetailedError)
	if ok {
		return de, true
	}

	// If it's not a DetailedError, check if it's a gRPC status error
	st, ok := status.FromError(err)
	if !ok {
		return nil, false
	}
	for _, detail := range st.Details() {
		if pbErr, ok := detail.(*pb.DetailError); ok {
			return &domain.DetailedError{
				IDField:         pbErr.Id,
				StatusCodeField: int(pbErr.Code),
				StatusDescField: pbErr.Status,
				ReasonField:     pbErr.Reason,
				DebugField:      pbErr.Debug,
				ErrorField:      pbErr.Message,
				// DetailsField:  map[string]interface{}{}, // add if needed
			}, true
		}
	}
	return nil, false
}
