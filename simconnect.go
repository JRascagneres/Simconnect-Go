package simconnect

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"
	"unsafe"
)

type SimconnectInstance struct {
	handle           unsafe.Pointer // handle
	definitionMap    map[string]uint32
	nextDefinitionID uint32
}

type Report struct {
	RecvSimobjectDataByType
	Title               [256]byte `name:"Title"`
	Kohlsman            float64   `name:"Kohlsman setting hg" unit:"inHg"`
	Altitude            float64   `name:"Plane Altitude" unit:"feet"`
	AltitudeAboveGround float64   `name:"Plane Alt Above Ground" unit:"feet"`
	Latitude            float64   `name:"Plane Latitude" unit:"degrees"`
	Longitude           float64   `name:"Plane Longitude" unit:"degrees"`
	Airspeed            float64   `name:"Airspeed Indicated" unit:"knot"`
	AirspeedBarberPole  float64   `name:"Airspeed Barber Pole" unit:"knot"`
	GroundSpeed         float64   `name:"Ground Velocity" unit:"knots"`
	OnGround            bool      `name:"Sim On Ground" unit:"bool"`
	Heading             float32   `name:"Plane Heading Degrees True"`
	HeadingMag          float32   `name:"Plane Heading Degrees Magnetic"`
	Pitch               float32   `name:"Plane Pitch Degrees"`
	Bank                float32   `name:"Plane Bank Degrees"`
	GForce              float32   `name:"G Force"`
	VerticalSpeed       float32   `name:"Velocity World Y" unit:"Feet per second"`
	FuelTotal           float32   `name:"Fuel Total Quantity Weight" unit:"kg"`
	WindSpeed           float32   `name:"Ambient Wind Velocity" unit:"knot"`
	WindDirection       float32   `name:"Ambient Wind Direction" unit:"radians"`
	FuelCapacity        float32   `name:"FUEL TOTAL CAPACITY" unit:"gallons"`
	FuelWeightPerGallon float32   `name:"FUEL WEIGHT PER GALLON" unit:"kg"`
	FuelFlow            float32   `name:"ESTIMATED FUEL FLOW" unit:"kilograms per second"`
	AmbientTemperature  float32   `name:"Ambient Temperature" unit:"Celsius"`
	AmbientPressure     float32   `name:"Ambient Pressure" unit:"inHg"`
	Parked              bool      `name:"Plane In Parking State"`
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
	procSimconnectOpen                      *syscall.LazyProc
	procSimconnectClose                     *syscall.LazyProc
	procSimconnectRequestDataOnSimObject    *syscall.LazyProc
	procSimconnectAddtodatadefinition       *syscall.LazyProc
	procSimconnectGetnextdispatch           *syscall.LazyProc
	procSimconnectFlightplanLoad            *syscall.LazyProc
	procSimconnectAICreateParkedATCAircraft *syscall.LazyProc
	procSimconnectAICreateNonATCAircraft    *syscall.LazyProc
	procSimconnectSetDataOnSimObject        *syscall.LazyProc
)

func (instance *SimconnectInstance) getDefinitionID(input interface{}) (defID uint32, created bool) {
	structName := reflect.TypeOf(input).Elem().Name()

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

	r1, _, err := procSimconnectRequestDataOnSimObject.Call(args...)
	if int32(r1) < 0 {
		return fmt.Errorf("requestData for requestID %d defineID %d error: %d %v",
			requestID, defineID, r1, err)
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

	if uint32(r1) == E_FAIL {
		// No new message
		return nil, nil
	}

	return ppData, nil
}

func (instance *SimconnectInstance) processData() (unsafe.Pointer, *Recv, error) {
	for {
		time.Sleep(100 * time.Millisecond)
		ppData, err := instance.getData()
		if err != nil {
			return nil, nil, err
		}
		if ppData == nil {
			fmt.Println("Retrying....")
			continue
		}

		recvInfo := (*Recv)(ppData)
		return ppData, recvInfo, nil
	}
}

func (instance *SimconnectInstance) processConnectionOpenData() error {
	ppData, recvInfo, err := instance.processData()
	if err != nil {
		return err
	}
	switch recvInfo.ID {
	case RECV_ID_EXCEPTION:
		return fmt.Errorf("received exception")
	case RECV_ID_OPEN:
		recvOpen := *(*RecvOpen)(ppData)
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
	case RECV_ID_SIMOBJECT_DATA_BYTYPE:
		recvData := *(*RecvSimobjectDataByType)(ppData)
		switch recvData.RequestID {
		case instance.definitionMap["Report"]:
			report2 := (*Report)(ppData)
			return report2, nil
		}
	case RECV_ID_ASSIGNED_OBJECT_ID:
		recvData := *(*RecvAssignedObjectID)(ppData)
		return recvData.ObjectID, nil
	default:
		return nil, fmt.Errorf("processSimObjectTypeData() hit default")
	}

	return nil, fmt.Errorf("FAIL")
}

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
		SIMOBJECT_TYPE_USER,
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

func (instance *SimconnectInstance) openConnection() error {
	args := []uintptr{
		uintptr(unsafe.Pointer(&instance.handle)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("test"))),
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

// Public call to closeConnection () - In case we need to modify behaviour later
func (instance *SimconnectInstance) Close() error {
	return instance.closeConnection()
}

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

func (instance *SimconnectInstance) LoadNonATCAircraft(containerTitle, tailNumber string, initPos SimconnectDataInitPosition, requestID int) (*uint32, error) {
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

func (instance *SimconnectInstance) SetDataOnSimObject(objectID uint32, data *SetSimObjectDataExpose) {
	InternalSimObjectData := struct {
		Airspeed  float64 `name:"Airspeed Indicated" unit:"knot"`
		Altitude  float64 `name:"Plane Altitude" unit:"feet"`
		Bank      float64 `name:"Plane Bank Degrees"`
		Heading   float64 `name:"Plane Heading Degrees True"`
		Latitude  float64 `name:"Plane Latitude" unit:"degrees"`
		Longitude float64 `name:"Plane Longitude" unit:"degrees"`
		OnGround  float64 `name:"Sim On Ground" unit:"bool"`
		Pitch     float64 `name:"Plane Pitch Degrees"`
	}{}

	buf := [8]float64{
		data.Airspeed,
		data.Altitude,
		float64(data.Bank),
		float64(data.Heading),
		data.Latitude,
		data.Longitude,
		b2i(data.OnGround),
		float64(data.Pitch),
	}

	err := instance.registerDataDefinition(InternalSimObjectData)
	if err != nil {
		fmt.Println(err)
		return
	}
	defID, _ := instance.getDefinitionID(InternalSimObjectData)

	instance.setDataOnSimObject(defID, objectID, 0, 0, 8*8, unsafe.Pointer(&buf[0]))
}

func (instance *SimconnectInstance) setDataOnSimObject(defID, objectID, flags, arrayCount, size uint32, byteArray unsafe.Pointer) {
	args := []uintptr{
		uintptr(instance.handle),
		uintptr(defID),
		uintptr(objectID),
		uintptr(flags),
		uintptr(arrayCount),
		uintptr(size),
		uintptr(byteArray),
	}

	r1, r2, err := procSimconnectSetDataOnSimObject.Call(args...)
	fmt.Println(r1)
	fmt.Println(r2)
	fmt.Println(err)
}

func NewSimConnect() (*SimconnectInstance, error) {
	dllPath := filepath.Join("simconnect", "SimConnect.dll")

	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		buf := MustAsset("SimConnect.dll")

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
	procSimconnectRequestDataOnSimObject = mod.NewProc("SimConnect_RequestDataOnSimObjectType")
	procSimconnectAddtodatadefinition = mod.NewProc("SimConnect_AddToDataDefinition")
	procSimconnectGetnextdispatch = mod.NewProc("SimConnect_GetNextDispatch")
	procSimconnectFlightplanLoad = mod.NewProc("SimConnect_FlightPlanLoad")
	procSimconnectAICreateParkedATCAircraft = mod.NewProc("SimConnect_AICreateParkedATCAircraft")
	procSimconnectAICreateNonATCAircraft = mod.NewProc("SimConnect_AICreateNonATCAircraft")
	procSimconnectSetDataOnSimObject = mod.NewProc("SimConnect_SetDataOnSimObject")

	instance := SimconnectInstance{
		nextDefinitionID: 0,
		definitionMap:    map[string]uint32{},
	}

	err = instance.openConnection()
	if err != nil {
		return nil, err
	}

	err = instance.processConnectionOpenData()
	if err != nil {
		return nil, err
	}

	return &instance, nil
}
