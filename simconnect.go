package simconnect

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"
	"time"
	"unsafe"

	simconnect_data "github.com/JRascagneres/Simconnect-Go/simconnect-data"

	_ "embed"
)

type SimconnectInstance struct {
	handle           unsafe.Pointer // handle
	definitionMap    map[string]uint32
	nextDefinitionID uint32

	definitionMapMutex sync.Mutex
}

// Report contains data for a given sim object
type Report struct {
	simconnect_data.RecvSimobjectDataByType
	Title                        [256]byte `name:"Title"`
	ATCAirline                   [128]byte `name:"ATC Airline"`
	ATCFlightNumber              [128]byte `name:"ATC Flight Number"`
	ATCID                        [128]byte `name:"ATC ID"`
	Kohlsman                     float64   `name:"Kohlsman setting hg" unit:"inHg"`
	Altitude                     float64   `name:"Plane Altitude" unit:"feet"`
	AltitudeAboveGround          float64   `name:"Plane Alt Above Ground" unit:"feet"`
	Latitude                     float64   `name:"Plane Latitude" unit:"degrees"`
	Longitude                    float64   `name:"Plane Longitude" unit:"degrees"`
	Airspeed                     float64   `name:"Airspeed Indicated" unit:"knot"`
	AirspeedBarberPole           float64   `name:"Airspeed Barber Pole" unit:"knot"`
	GroundSpeed                  float64   `name:"Ground Velocity" unit:"knots"`
	OnGround                     int32     `name:"Sim On Ground" unit:"bool"`
	Heading                      float32   `name:"Plane Heading Degrees True"`
	HeadingMag                   float32   `name:"Plane Heading Degrees Magnetic"`
	Pitch                        float32   `name:"Plane Pitch Degrees"`
	Bank                         float32   `name:"Plane Bank Degrees"`
	GForce                       float32   `name:"G Force"`
	VerticalSpeedRelativeToWorld float32   `name:"Velocity World Y" unit:"Feet per second"`
	VerticalSpeedAircraft        float32   `name:"Vertical Speed" unit:"Feet per second"`
	FuelTotal                    float32   `name:"Fuel Total Quantity Weight" unit:"kg"`
	WindSpeed                    float32   `name:"Ambient Wind Velocity" unit:"knot"`
	WindDirection                float32   `name:"Ambient Wind Direction" unit:"radians"`
	FuelCapacity                 float32   `name:"FUEL TOTAL CAPACITY" unit:"gallons"`
	FuelWeightPerGallon          float32   `name:"FUEL WEIGHT PER GALLON" unit:"kg"`
	FuelFlow                     float32   `name:"ESTIMATED FUEL FLOW" unit:"kilograms per second"`
	AmbientTemperature           float32   `name:"Ambient Temperature" unit:"Celsius"`
	AmbientPressure              float32   `name:"Ambient Pressure" unit:"inHg"`
	Parked                       int32     `name:"Plane In Parking State"`
	Engine1Combustion            int32     `name:"General Eng Combustion:1" unit:"bool"`
	Engine2Combustion            int32     `name:"General Eng Combustion:2" unit:"bool"`
	Engine3Combustion            int32     `name:"General Eng Combustion:3" unit:"bool"`
	Engine4Combustion            int32     `name:"General Eng Combustion:4" unit:"bool"`
	EngineCount                  int32     `name:"Number Of Engines"`
}

type SetSimObjectDataExpose struct {
	Airspeed  float64
	Altitude  float64
	Bank      float32
	Heading   float32
	Latitude  float64
	Longitude float64
	OnGround  bool
	Pitch     float32
}

var (
	procSimconnectOpen                       *syscall.LazyProc
	procSimconnectClose                      *syscall.LazyProc
	procSimconnectRequestDataOnSimObjectType *syscall.LazyProc
	procSimconnectRequestDataOnSimObject     *syscall.LazyProc
	procSimconnectAddtodatadefinition        *syscall.LazyProc
	procSimconnectGetnextdispatch            *syscall.LazyProc
	procSimconnectFlightplanLoad             *syscall.LazyProc
	procSimconnectAICreateParkedATCAircraft  *syscall.LazyProc
	procSimconnectAICreateNonATCAircraft     *syscall.LazyProc
	procSimconnectSetDataOnSimObject         *syscall.LazyProc
	procSimconnectCreateEnrouteATCAircraft   *syscall.LazyProc
	procSimconnectAISetAircraftFlightPlan    *syscall.LazyProc
	procSimconnectAIRemoveObject             *syscall.LazyProc
)

func (instance *SimconnectInstance) getDefinitionID(input interface{}) (defID uint32, created bool) {
	structName := reflect.TypeOf(input).Elem().Name()

	instance.definitionMapMutex.Lock()
	defer instance.definitionMapMutex.Unlock()

	id, ok := instance.definitionMap[structName]
	if !ok {
		instance.definitionMap[structName] = instance.nextDefinitionID
		instance.nextDefinitionID++
		return instance.definitionMap[structName], true
	}

	return id, false
}

// Made request to DLL to actually register a data definition
func (instance *SimconnectInstance) addToDataDefinitions(definitionID uint32, name, unit string, dataType uint32) error {
	nameParam := []byte(name + "\x00")
	unitParam := []byte(unit + "\x00")

	args := []uintptr{
		uintptr(instance.handle),
		uintptr(definitionID),
		uintptr(unsafe.Pointer(&nameParam[0])),
		uintptr(0),
		uintptr(dataType),
		uintptr(float32(0)),
		uintptr(0xffffffff),
	}
	if unit != "" {
		args[3] = uintptr(unsafe.Pointer(&unitParam[0]))
	}

	r1, _, err := procSimconnectAddtodatadefinition.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("add to data definition failed for %s error: %d %s", name, r1, err)
	}

	return nil
}

func (instance *SimconnectInstance) registerDataDefinition(input interface{}) error {
	definitionID, created := instance.getDefinitionID(input)
	if !created {
		return nil
	}

	v := reflect.ValueOf(input).Elem()
	for j := 1; j < v.NumField(); j++ {
		fieldName := v.Type().Field(j).Name
		nameTag, _ := v.Type().Field(j).Tag.Lookup("name")
		unitTag, _ := v.Type().Field(j).Tag.Lookup("unit")

		fieldType := v.Field(j).Kind().String()
		if fieldType == "array" {
			fieldType = fmt.Sprintf("[%d]byte", v.Field(j).Type().Len())
		}

		if nameTag == "" {
			return fmt.Errorf("name tag not found %s", fieldName)
		}

		dataType, err := derefDataType(fieldType)
		if err != nil {
			return fmt.Errorf("error derefing datatype: %v", err)
		}

		err = instance.addToDataDefinitions(definitionID, nameTag, unitTag, dataType)
		if err != nil {
			return fmt.Errorf("error adding data definition: %v", err)
		}
	}

	return nil
}

func (instance *SimconnectInstance) requestDataOnSimObjectType(requestID, defineID, radius, simObjectType uint32) error {
	args := []uintptr{
		uintptr(instance.handle),
		uintptr(requestID),
		uintptr(defineID),
		uintptr(radius),
		uintptr(simObjectType),
	}

	r1, _, err := procSimconnectRequestDataOnSimObjectType.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("requestData for requestID %d defineID %d error: %d %v",
			requestID, defineID, r1, err)
	}

	return nil
}

func (instance *SimconnectInstance) requestDataOnSimObject(requestID, defineID, objectID, period uint32) error {
	args := []uintptr{
		uintptr(instance.handle),
		uintptr(requestID),
		uintptr(defineID),
		uintptr(objectID),
		uintptr(period),
	}

	r1, _, err := procSimconnectRequestDataOnSimObject.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("requestData for requestID %d defineID %d objectID %d error: %d %v", requestID, defineID, objectID, r1, err)
	}

	return nil
}

func (instance *SimconnectInstance) getData() (unsafe.Pointer, error) {
	var ppData unsafe.Pointer
	var ppDataLength uint32

	r1, _, err := procSimconnectGetnextdispatch.Call(
		uintptr(instance.handle),
		uintptr(unsafe.Pointer(&ppData)),
		uintptr(unsafe.Pointer(&ppDataLength)),
	)

	if r1 < 0 {
		return nil, fmt.Errorf("GetNextDispatch error: %d %v", r1, err)
	}

	if uint32(r1) == simconnect_data.E_FAIL {
		// No new message
		return nil, nil
	}

	return ppData, nil
}

func (instance *SimconnectInstance) processData() (unsafe.Pointer, *simconnect_data.Recv, error) {
	var ppData unsafe.Pointer
	var err error
	var recvInfo *simconnect_data.Recv
	loopErr := retryFunc(20, time.Millisecond*20, func() (bool, error) {

		ppData, err = instance.getData()
		if err != nil {
			return true, nil
		}
		if ppData == nil {
			return true, err
		}

		recvInfo = (*simconnect_data.Recv)(ppData)

		if recvInfo.ID == simconnect_data.RECV_ID_EXCEPTION {
			return true, errors.New("exception")
		}

		return false, nil
	})
	if loopErr != nil {
		return nil, nil, loopErr
	}

	return ppData, recvInfo, nil
}

func (instance *SimconnectInstance) processConnectionOpenData() error {
	ppData, recvInfo, err := instance.processData()
	if err != nil {
		return err
	}
	switch recvInfo.ID {
	case simconnect_data.RECV_ID_EXCEPTION:
		return fmt.Errorf("received exception")
	case simconnect_data.RECV_ID_OPEN:
		recvOpen := *(*simconnect_data.RecvOpen)(ppData)
		fmt.Println("SIMCONNECT_RECV_ID_OPEN", fmt.Sprintf("%s", recvOpen.ApplicationName))
		return nil
	default:
		return fmt.Errorf("processConnectionOpenData() hit default")
	}
}

func (instance *SimconnectInstance) processSimObjectTypeData() (interface{}, error) {
	ppData, recvInfo, err := instance.processData()
	if err != nil {
		return nil, err
	}
	switch recvInfo.ID {
	case simconnect_data.RECV_ID_SIMOBJECT_DATA_BYTYPE:
		recvData := *(*simconnect_data.RecvSimobjectDataByType)(ppData)

		instance.definitionMapMutex.Lock()
		defer instance.definitionMapMutex.Unlock()

		switch recvData.RequestID {
		case instance.definitionMap["Report"]:
			report2 := (*Report)(ppData)
			return report2, nil
		}
	case simconnect_data.RECV_ID_SIMOBJECT_DATA:
		report2 := (*Report)(ppData)
		return report2, nil
	case simconnect_data.RECV_ID_ASSIGNED_OBJECT_ID:
		recvData := *(*simconnect_data.RecvAssignedObjectID)(ppData)
		return recvData.ObjectID, nil
	default:
		recvData := *(*simconnect_data.RecvSimobjectDataByType)(ppData)
		return nil, fmt.Errorf("processSimObjectTypeData() hit default recvInfo: %v ppData: %+v", recvInfo, recvData)
	}

	return nil, fmt.Errorf("FAIL")
}

// GetReport returns Report struct containing current user data
func (instance *SimconnectInstance) GetReport() (*Report, error) {
	report := &Report{}
	err := instance.registerDataDefinition(report)
	if err != nil {
		return nil, err
	}
	definitionID, _ := instance.getDefinitionID(report)
	err = instance.requestDataOnSimObjectType(
		definitionID,
		definitionID,
		0,
		simconnect_data.SIMOBJECT_TYPE_USER,
	)
	if err != nil {
		return nil, err
	}

	reportData, err := instance.processSimObjectTypeData()
	if err != nil {
		return nil, err
	}

	return reportData.(*Report), nil
}

// GetReportOnObjectID returns a Report struct containing the data for the Object ID passed in
func (instance *SimconnectInstance) GetReportOnObjectID(objectID uint32) (*Report, error) {
	report := &Report{}
	err := instance.registerDataDefinition(report)
	if err != nil {
		return nil, err
	}
	definitionID, _ := instance.getDefinitionID(report)
	err = instance.requestDataOnSimObject(definitionID, definitionID, objectID, simconnect_data.SIMCONNECT_PERIOD_ONCE)

	if err != nil {
		return nil, err
	}

	reportData, err := instance.processSimObjectTypeData()
	if err != nil {
		return nil, err
	}

	return reportData.(*Report), nil

}

func (instance *SimconnectInstance) openConnection(simconnectName string) error {
	args := []uintptr{
		uintptr(unsafe.Pointer(&instance.handle)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(simconnectName))),
		0,
		0,
		0,
		0,
	}

	r1, _, err := procSimconnectOpen.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("open connect failed, error: %d %v", r1, err)
	}

	return nil
}

func (instance *SimconnectInstance) closeConnection() error {
	r1, _, err := procSimconnectClose.Call(uintptr(instance.handle))
	if int32(r1) < 0 {
		return fmt.Errorf("close connection failed, error %d %v", r1, err)
	}
	return nil
}

// Close will end the connection to the SimConnect API
func (instance *SimconnectInstance) Close() error {
	return instance.closeConnection()
}

// LoadFlightPlan will load the supplied flight plan path into the users aircraft. FlightPlanPath must be a pln but the
// .pln extension must not be supplied with the flight plan.
func (instance *SimconnectInstance) LoadFlightPlan(flightPlanPath string) error {

	flightPlanPathArg := []byte(flightPlanPath + "\x00")

	args := []uintptr{
		uintptr(instance.handle),
		uintptr(unsafe.Pointer(&flightPlanPathArg[0])),
	}
	r1, _, err := procSimconnectFlightplanLoad.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("error: %d %v", r1, err)
	}

	return nil
}

// LoadParkedATCAircraft will load a parked ATC aircraft with the specified parameters. See SimConnect API reference.
func (instance *SimconnectInstance) LoadParkedATCAircraft(containerTitle, tailNumber, airportICAO string, requestID int) (*uint32, error) {
	containerTitleArg := []byte(containerTitle + "\x00")
	tailNumberArg := []byte(tailNumber + "\x00")
	airportICAOArg := []byte(airportICAO + "\x00")

	args := []uintptr{
		uintptr(instance.handle),
		uintptr(unsafe.Pointer(&containerTitleArg[0])),
		uintptr(unsafe.Pointer(&tailNumberArg[0])),
		uintptr(unsafe.Pointer(&airportICAOArg[0])),
		uintptr(unsafe.Pointer(&requestID)),
	}
	r1, _, err := procSimconnectAICreateParkedATCAircraft.Call(args...)
	if int32(r1) < 0 {
		return nil, fmt.Errorf("error: %d %v", r1, err)
	}
	objectIDInterface, err := instance.processSimObjectTypeData()
	if err != nil {
		return nil, err
	}
	objectID := objectIDInterface.(uint32)
	return &objectID, nil
}

// LoadNonATCAircraft will load a non ATC (vfr) aircraft with the specified parameters. See SimConnect API reference.
func (instance *SimconnectInstance) LoadNonATCAircraft(containerTitle, tailNumber string, initPos simconnect_data.SimconnectDataInitPosition, requestID int) (*uint32, error) {
	containerTitleArg := []byte(containerTitle + "\x00")
	tailNumberArg := []byte(tailNumber + "\x00")

	args := []uintptr{
		uintptr(instance.handle),
		uintptr(unsafe.Pointer(&containerTitleArg[0])),
		uintptr(unsafe.Pointer(&tailNumberArg[0])),
		uintptr(unsafe.Pointer(&initPos)),
		uintptr(unsafe.Pointer(&requestID)),
	}
	r1, _, err := procSimconnectAICreateNonATCAircraft.Call(args...)
	if int32(r1) < 0 {
		return nil, fmt.Errorf("error: %d %v", r1, err)
	}

	objectIDInterface, err := instance.processSimObjectTypeData()
	if err != nil {
		return nil, err
	}
	objectID := objectIDInterface.(uint32)
	return &objectID, nil
}

func b2i(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// SetDataOnSimObject allows you to set data for a given sim object, 0 can be used to apply the data to the users
// aircraft. See SimConnect API reference.
func (instance *SimconnectInstance) SetDataOnSimObject(objectID uint32, data []SetSimObjectDataExpose) error {
	InternalSimObjectData := struct {
		simconnect_data.RecvSimobjectDataByType
		Airspeed  float64 `name:"Airspeed Indicated" unit:"knot"`
		Altitude  float64 `name:"Plane Altitude" unit:"feet"`
		Bank      float64 `name:"Plane Bank Degrees"`
		Heading   float64 `name:"Plane Heading Degrees True"`
		Latitude  float64 `name:"Plane Latitude" unit:"degrees"`
		Longitude float64 `name:"Plane Longitude" unit:"degrees"`
		OnGround  float64 `name:"Sim On Ground" unit:"bool"`
		Pitch     float64 `name:"Plane Pitch Degrees"`
	}{}

	buf := make([][8]float64, len(data))

	for i := 0; i < len(data); i++ {
		dataItem := data[i]
		buf[i] = [8]float64{
			dataItem.Airspeed,
			dataItem.Altitude,
			float64(dataItem.Bank),
			float64(dataItem.Heading),
			dataItem.Latitude,
			dataItem.Longitude,
			b2i(dataItem.OnGround),
			float64(dataItem.Pitch),
		}
	}

	err := instance.registerDataDefinition(&InternalSimObjectData)
	if err != nil {
		return err
	}
	defID, _ := instance.getDefinitionID(&InternalSimObjectData)

	return instance.setDataOnSimObject(defID, objectID, 0, uint32(len(data)), uint32(8*8), unsafe.Pointer(&buf[0]))
}

func (instance *SimconnectInstance) setDataOnSimObject(defID, objectID, flags, arrayCount, size uint32, byteArray unsafe.Pointer) error {
	args := []uintptr{
		uintptr(instance.handle),
		uintptr(defID),
		uintptr(objectID),
		uintptr(flags),
		uintptr(arrayCount),
		uintptr(size),
		uintptr(byteArray),
	}

	r1, _, err := procSimconnectSetDataOnSimObject.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("setDataOnSimObject for objectID %d error: %d %v", objectID, r1, err)
	}

	return nil
}

// CreateEnrouteATCAircraft allows you to create an ATC already part way through its flight plan. See SimConnect API
// reference.
func (instance *SimconnectInstance) CreateEnrouteATCAircraft(containerTitle, tailNumber string, flightNumber uint32, flightPlanPath string, flightPlanPosition float32, touchAndGo bool, requestID uint32) (*uint32, error) {
	containerTitleArg := []byte(containerTitle + "\x00")
	tailNumberArg := []byte(tailNumber + "\x00")
	pathArg := []byte(flightPlanPath + "\x00")

	args := []uintptr{
		uintptr(instance.handle),
		uintptr(unsafe.Pointer(&containerTitleArg[0])),
		uintptr(unsafe.Pointer(&tailNumberArg[0])),
		uintptr(flightNumber),
		uintptr(unsafe.Pointer(&pathArg[0])),
		uintptr(flightPlanPosition),
		uintptr(b2i(touchAndGo)),
		uintptr(requestID),
	}
	r1, _, err := procSimconnectCreateEnrouteATCAircraft.Call(args...)
	if int32(r1) < 0 {
		return nil, fmt.Errorf("error: %d %v", r1, err)
	}
	objectIDInterface, err := instance.processSimObjectTypeData()
	if err != nil {
		return nil, err
	}

	objectID := objectIDInterface.(uint32)

	return &objectID, nil
}

// SetAircraftFlightPlan allows you to set a flight plan for an existing aircraft. See SimConnect API reference.
func (instance *SimconnectInstance) SetAircraftFlightPlan(objectID, requestID uint32, flightPlanPath string) error {
	pathArg := []byte(flightPlanPath + "\x00")

	args := []uintptr{
		uintptr(instance.handle),
		uintptr(objectID),
		uintptr(unsafe.Pointer(&pathArg[0])),
		uintptr(requestID),
	}

	r1, _, err := procSimconnectAISetAircraftFlightPlan.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("error: %d %v", r1, err)
	}

	return nil
}

// RemoveAIObject will remove an AI object from the sim. See SimConnect API reference.
func (instance *SimconnectInstance) RemoveAIObject(objectID, requestID uint32) error {
	args := []uintptr{
		uintptr(instance.handle),
		uintptr(objectID),
		uintptr(requestID),
	}

	r1, _, err := procSimconnectAIRemoveObject.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("error: %d %v", r1, err)
	}

	return nil
}

// NewSimConnect returns a new instance of SimConnect which will be used to call the methods.
func NewSimConnect(simconnectName string) (*SimconnectInstance, error) {
	dllPath := filepath.Join("simconnect-data", "SimConnect.dll")

	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		//go:embed "simconnect-data/SimConnect.dll"
		var buf []byte

		dir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, err
		}
		dllPath = filepath.Join(dir, "SimConnect.dll")

		if err := ioutil.WriteFile(dllPath, buf, 0644); err != nil {
			return nil, err
		}
	}

	mod := syscall.NewLazyDLL(dllPath)
	err := mod.Load()
	if err != nil {
		return nil, err
	}

	procSimconnectOpen = mod.NewProc("SimConnect_Open")
	procSimconnectClose = mod.NewProc("SimConnect_Close")
	procSimconnectRequestDataOnSimObjectType = mod.NewProc("SimConnect_RequestDataOnSimObjectType")
	procSimconnectRequestDataOnSimObject = mod.NewProc("SimConnect_RequestDataOnSimObject")
	procSimconnectAddtodatadefinition = mod.NewProc("SimConnect_AddToDataDefinition")
	procSimconnectGetnextdispatch = mod.NewProc("SimConnect_GetNextDispatch")
	procSimconnectFlightplanLoad = mod.NewProc("SimConnect_FlightPlanLoad")
	procSimconnectAICreateParkedATCAircraft = mod.NewProc("SimConnect_AICreateParkedATCAircraft")
	procSimconnectAICreateNonATCAircraft = mod.NewProc("SimConnect_AICreateNonATCAircraft")
	procSimconnectSetDataOnSimObject = mod.NewProc("SimConnect_SetDataOnSimObject")
	procSimconnectCreateEnrouteATCAircraft = mod.NewProc("SimConnect_AICreateEnrouteATCAircraft")
	procSimconnectAISetAircraftFlightPlan = mod.NewProc("SimConnect_AISetAircraftFlightPlan")
	procSimconnectAIRemoveObject = mod.NewProc("SimConnect_AIRemoveObject")

	instance := SimconnectInstance{
		nextDefinitionID: 0,
		definitionMap:    map[string]uint32{},
	}

	err = instance.openConnection(simconnectName)
	if err != nil {
		return nil, err
	}

	err = instance.processConnectionOpenData()
	if err != nil {
		return nil, err
	}

	return &instance, nil
}
