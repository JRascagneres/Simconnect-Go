package simconnect_data

// Exception Fail ID
const E_FAIL uint32 = 0x80004005

// DataType IDs
const (
	DATATYPE_INVALID      uint32 = iota // invalid data type
	DATATYPE_INT32                      // 32-bit integer number
	DATATYPE_INT64                      // 64-bit integer number
	DATATYPE_FLOAT32                    // 32-bit floating-point number (float)
	DATATYPE_FLOAT64                    // 64-bit floating-point number (double)
	DATATYPE_STRING8                    // 8-byte string
	DATATYPE_STRING32                   // 32-byte string
	DATATYPE_STRING64                   // 64-byte string
	DATATYPE_STRING128                  // 128-byte string
	DATATYPE_STRING256                  // 256-byte string
	DATATYPE_STRING260                  // 260-byte string
	DATATYPE_STRINGV                    // variable-length string
	DATATYPE_INITPOSITION               // see SIMCONNECT_DATA_INITPOSITION
	DATATYPE_MARKERSTATE                // see SIMCONNECT_DATA_MARKERSTATE
	DATATYPE_WAYPOINT                   // see SIMCONNECT_DATA_WAYPOINT
	DATATYPE_LATLONALT                  // see SIMCONNECT_DATA_LATLONALT
	DATATYPE_XYZ                        // see SIMCONNECT_DATA_XYZ

	DATATYPE_MAX // enum limit
)

// Receive ID Types
const (
	RECV_ID_NULL uint32 = iota
	RECV_ID_EXCEPTION
	RECV_ID_OPEN
	RECV_ID_QUIT
	RECV_ID_EVENT
	RECV_ID_EVENT_OBJECT_ADDREMOVE
	RECV_ID_EVENT_FILENAME
	RECV_ID_EVENT_FRAME
	RECV_ID_SIMOBJECT_DATA
	RECV_ID_SIMOBJECT_DATA_BYTYPE
	RECV_ID_WEATHER_OBSERVATION
	RECV_ID_CLOUD_STATE
	RECV_ID_ASSIGNED_OBJECT_ID
	RECV_ID_RESERVED_KEY
	RECV_ID_CUSTOM_ACTION
	RECV_ID_SYSTEM_STATE
	RECV_ID_CLIENT_DATA
	RECV_ID_EVENT_WEATHER_MODE
	RECV_ID_AIRPORT_LIST
	RECV_ID_VOR_LIST
	RECV_ID_NDB_LIST
	RECV_ID_WAYPOINT_LIST
	RECV_ID_EVENT_MULTIPLAYER_SERVER_STARTED
	RECV_ID_EVENT_MULTIPLAYER_CLIENT_STARTED
	RECV_ID_EVENT_MULTIPLAYER_SESSION_ENDED
	RECV_ID_EVENT_RACE_END
	RECV_ID_EVENT_RACE_LAP

	RECV_ID_PICK
)

// SimObject Types
const (
	SIMOBJECT_TYPE_USER uint32 = iota
	SIMOBJECT_TYPE_ALL
	SIMOBJECT_TYPE_AIRCRAFT
	SIMOBJECT_TYPE_HELICOPTER
	SIMOBJECT_TYPE_BOAT
	SIMOBJECT_TYPE_GROUND
)

const (
	SIMCONNECT_PERIOD_NEVER uint32 = iota
	SIMCONNECT_PERIOD_ONCE
	SIMCONNECT_PERIOD_VISUAL_FRAME
	SIMCONNECT_PERIOD_SIM_FRAME
	SIMCONNECT_PERIOD_SECOND
)

type SimconnectDataInitPosition struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	Pitch     float64
	Bank      float64
	Heading   float64
	OnGround  bool
	Airspeed  uint32
}

// Receive Data Struct - Used to get basic dispatch info which can then be handled later
type Recv struct {
	Size    uint32
	Version uint32
	ID      uint32
}

// Receive
type RecvException struct {
	Recv
	Exception uint32 // see SIMCONNECT_EXCEPTION
	SendID    uint32 // see SimConnect_GetLastSentPacketID
	Index     uint32 // index of parameter that was source of error
}

// Used to capture return for open connection - Not yet used
type RecvOpen struct {
	Recv
	ApplicationName         [256]byte
	ApplicationVersionMajor uint32
	ApplicationVersionMinor uint32
	ApplicationBuildMajor   uint32
	ApplicationBuildMinor   uint32
	SimConnectVersionMajor  uint32
	SimConnectVersionMinor  uint32
	SimConnectBuildMajor    uint32
	SimConnectBuildMinor    uint32
	Reserved1               uint32
	Reserved2               uint32
}

// Used to capture return for event - Not yet used
type RecvEvent struct {
	Recv
	GroupID uint32
	EventID uint32
	Data    uint32
}

// Used to store SimObject return data
type RecvSimobjectData struct {
	Recv
	RequestID   uint32
	ObjectID    uint32
	DefineID    uint32
	Flags       uint32
	EntryNumber uint32
	OutOf       uint32
	DefineCount uint32
}

type RecvAssignedObject struct {
	Recv
	RequestID uint32
	ObjectID  uint32
}

type RecvSimobjectDataByType struct {
	RecvSimobjectData
}

type RecvAssignedObjectID struct {
	RecvAssignedObject
}
