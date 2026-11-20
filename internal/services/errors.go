package services

import (
	"errors"

	"google.golang.org/grpc/status"
)

func IsFromGRPC(err error, target error) bool {
	// Standard check
	if errors.Is(err, target) {
		return true
	}

	// GRPC error check

	target_desc := ""
	if target != nil {
		target_desc = target.Error()
	}

	st, ok := status.FromError(err)
	if ok {
		return st.Message() == target_desc
	}

	return false
}
