package deployment

import "errors"

var ErrFunctionAlreadyExists = errors.New("a function with this name already exists for this user")
