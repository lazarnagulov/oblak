package deployment

import (
	"errors"
	"fmt"
)

var ErrFunctionAlreadyExists = errors.New("a function with this name already exists for this user")
var ErrAccessTokenInvalid = errors.New("access token is invalid or expired")

type ErrVerificationFailed struct {
	Reason string
}

func (e *ErrVerificationFailed) Error() string {
	return fmt.Sprintf("Artifact failed security verification: %s", e.Reason)
}
