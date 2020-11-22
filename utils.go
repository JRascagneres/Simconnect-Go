package simconnect

import "fmt"

func derefDataType(fieldType string) (uint32, error) {
	var dataType uint32
	switch fieldType {
	case "int32", "bool":
		dataType = DATATYPE_INT32
	case "int64":
		dataType = DATATYPE_INT64
	case "float32":
		dataType = DATATYPE_FLOAT32
	case "float64":
		dataType = DATATYPE_FLOAT64
	case "[8]byte":
		dataType = DATATYPE_STRING8
	case "[32]byte":
		dataType = DATATYPE_STRING32
	case "[64]byte":
		dataType = DATATYPE_STRING64
	case "[128]byte":
		dataType = DATATYPE_STRING128
	case "[256]byte":
		dataType = DATATYPE_STRING256
	case "[260]byte":
		dataType = DATATYPE_STRING260
	default:
		return 0, fmt.Errorf("DATATYPE not implemented: %s", fieldType)
	}

	return dataType, nil
}
