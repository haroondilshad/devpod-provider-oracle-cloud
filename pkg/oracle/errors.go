package oracle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
)

// IsNotFound returns true if the error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.GetHTTPStatusCode() == 404
	}

	return strings.Contains(err.Error(), "not found") || 
		   strings.Contains(err.Error(), "NotFound") || 
		   strings.Contains(err.Error(), "does not exist")
}

// MissingMachineID returns a missing machine id error
func MissingMachineID() error {
	return fmt.Errorf(errMissingMachineID)
}

// MissingServer returns a missing server error
func MissingServer() error {
	return fmt.Errorf(errMissingServer)
}

// MissingVolume returns a missing volume error
func MissingVolume() error {
	return fmt.Errorf(errMissingVolume)
} 