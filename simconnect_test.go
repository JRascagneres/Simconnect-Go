package simconnect

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	simconnect_data "github.com/JRascagneres/Simconnect-Go/simconnect-data"
)

func TestExample(t *testing.T) {
	instance, err := NewSimConnect("data")
	require.NoError(t, err)

	report, err := instance.GetReport()
	require.NoError(t, err)

	fmt.Printf("User Altitude: %f\n", report.Altitude)
	fmt.Printf("Eng 1: %v\n", report.Engine1Combustion)
	fmt.Printf("Eng 2: %v\n", report.Engine2Combustion)
	fmt.Printf("Eng 3: %v\n", report.Engine3Combustion)
	fmt.Printf("Eng 4: %v\n", report.Engine4Combustion)
	fmt.Printf("Engine Count: %v\n", report.EngineCount)

	err = instance.Close()
	assert.NoError(t, err)
}

// These aren't 'real' tests. This is simply for testing easily within the game.
func TestWork(t *testing.T) {
	instance, _ := NewSimConnect("data")

	instance.GetReport()

	i := 10
	objID, _ := instance.LoadNonATCAircraft("Boeing 747-8i Asobo", "G-42"+strconv.FormatInt(int64(i), 10), simconnect_data.SimconnectDataInitPosition{
		Airspeed:  200,
		Altitude:  235,
		Bank:      0,
		Heading:   0,
		Latitude:  53.34974539799793,
		Longitude: -2.274003348644879,
		OnGround:  false,
		Pitch:     0,
	}, i)

	time.Sleep(10 * time.Second)

	instance.SetDataOnSimObject(*objID, []SetSimObjectDataExpose{{
		Airspeed:  200,
		Altitude:  400,
		Bank:      0,
		Heading:   0,
		Latitude:  53.34974539799793,
		Longitude: -2.274003348644879,
		OnGround:  false,
		Pitch:     10,
	}})

	data, _ := instance.GetReportOnObjectID(*objID)
	fmt.Println(data.Altitude)
	time.Sleep(10 * time.Second)
}

func TestWork2(t *testing.T) {
	instance, err := NewSimConnect("data")
	require.NoError(t, err)

	objID, err := instance.LoadParkedATCAircraft("Boeing 747-8i Asobo", "G-420", "EGCC", 100)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	err = instance.SetAircraftFlightPlan(*objID, 1000, "C:\\Users\\Jacques\\Desktop\\EGCCLFPG")
	require.NoError(t, err)

	data, _ := instance.GetReportOnObjectID(*objID)

	time.Sleep(5 * time.Second)

	err = instance.RemoveAIObject(*objID, 10001)
	require.NoError(t, err)

	time.Sleep(60 * time.Second)

	fmt.Println(data.Altitude)
	time.Sleep(10 * time.Second)
}

func TestSystemEvent(t *testing.T) {
	instance, err := NewSimConnect("test")
	require.NoError(t, err)

	err = instance.SubscribeToSystemEvent(10, "4sec")
	assert.NoError(t, err)

	dataChan, errChan := instance.processEventData(nil)

	if data, open := <-dataChan; open {
		fmt.Println(data)
	}

	if err, open := <-errChan; open {
		fmt.Println(err)
	}

}

func TestRadioSet(t *testing.T) {
	instance, err := NewSimConnect(t.Name())
	require.NoError(t, err)

	report, err := instance.GetReport()
	require.NoError(t, err)
	require.NotNil(t, report)

	type Events struct {
		SetComRadioHz uint32
	}

	events := &Events{
		SetComRadioHz: 1,
	}

	report, err = instance.GetReport()
	assert.NoError(t, err)

	err = instance.MapClientEventToSimEvent(events.SetComRadioHz, "COM_RADIO_SET_HZ")
	require.NoError(t, err)

	err = instance.TransmitClientID(events.SetComRadioHz, 124850000)
	require.NoError(t, err)

}
