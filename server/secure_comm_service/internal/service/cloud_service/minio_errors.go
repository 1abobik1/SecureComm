package cloud_service

import "fmt"

type OperationError struct {
	ObjectID string
	Err      error
}

func (oe OperationError) Error() string {
	return fmt.Sprintf("object %s: %v", oe.ObjectID, oe.Err)
}
