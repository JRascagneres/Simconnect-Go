package simconnect

import (
	"errors"
	"fmt"
	"time"

	simconnect_data "github.com/JRascagneres/Simconnect-Go/simconnect-data"
)

func derefDataType(fieldType string) (uint32, error) {
	var dataType uint32
	switch fieldType {
	case "int32", "bool":
		dataType = simconnect_data.DATATYPE_INT32
	case "int64":
		dataType = simconnect_data.DATATYPE_INT64
	case "float32":
		dataType = simconnect_data.DATATYPE_FLOAT32
	case "float64":
		dataType = simconnect_data.DATATYPE_FLOAT64
	case "[8]byte":
		dataType = simconnect_data.DATATYPE_STRING8
	case "[32]byte":
		dataType = simconnect_data.DATATYPE_STRING32
	case "[64]byte":
		dataType = simconnect_data.DATATYPE_STRING64
	case "[128]byte":
		dataType = simconnect_data.DATATYPE_STRING128
	case "[256]byte":
		dataType = simconnect_data.DATATYPE_STRING256
	case "[260]byte":
		dataType = simconnect_data.DATATYPE_STRING260
	default:
		return 0, fmt.Errorf("DATATYPE not implemented: %s", fieldType)
	}

	return dataType, nil
}

func retryFunc(maxRetryCount int, waitDuration time.Duration, dataFunc func() (bool, error)) error {
	numAttempts := 1

	for {
		shouldRetry, _ := dataFunc()
		if !shouldRetry {
			return nil
		}

		numAttempts++

		if numAttempts >= maxRetryCount {
			return errors.New("timeout exceeded err")
		}

		time.Sleep(waitDuration)
	}
}
