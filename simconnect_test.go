package simconnect

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	simconnect_data "github.com/JRascagneres/Simconnect-Go/simconnect-data"
)

func TestExample(t *testing.T) {
	instance, err := NewSimConnect("data")
	if err != nil {
		panic(err)
	}

	report, err := instance.GetReport()
	if err != nil {
		panic(err)
	}

	fmt.Printf("User Altitude: %f\n", report.Altitude)

	err = instance.Close()
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
	instance, _ := NewSimConnect("data")

	objID, _ := instance.LoadParkedATCAircraft("Boeing 747-8i Asobo", "G-420", "EGCC", 100)

	time.Sleep(5 * time.Second)

	//instance.setAircraftFlightPlan(*objID)

	//objID, _ := instance.createEnrouteATCAircraft("Boeing 747-8i Asobo", "G-420", 1111111, "C:\\Users\\Jacques\\Desktop\\EGCCLFPG", 0, false, 50)
	//
	instance.SetAircraftFlightPlan(*objID, 1000, "C:\\Users\\Jacques\\Desktop\\EGCCLFPG")

	data, _ := instance.GetReportOnObjectID(*objID)

	time.Sleep(5 * time.Second)

	instance.RemoveAIObject(*objID, 10001)

	time.Sleep(60 * time.Second)

	fmt.Println(data.Altitude)
	time.Sleep(10 * time.Second)

}
