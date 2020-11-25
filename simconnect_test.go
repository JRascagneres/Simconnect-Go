package simconnect

import (
	"strconv"
	"testing"
	"time"
)

// These aren't 'real' tests. This is simply for testing easily within the game.
func TestWork(t *testing.T) {
	instance, _ := NewSimConnect()

	instance.GetReport()

	i := 10
	objID, _ := instance.LoadNonATCAircraft("Generic Airliner Twin Engines Asobo 00", "G-42"+strconv.FormatInt(int64(i), 10), SimconnectDataInitPosition{
		Airspeed:  200,
		Altitude:  235,
		Bank:      0,
		Heading:   0,
		Latitude:  53.34974539799793,
		Longitude: -2.274003348644879,
		OnGround:  false,
		Pitch:     0,
	}, i)

	time.Sleep(5 * time.Second)

	instance.SetDataOnSimObject(*objID, &SetSimObjectData{
		Airspeed: 10,
		Altitude: 400,
		//Bank:      0,
		//Heading:   0,
		Latitude:  53.34974539799793,
		Longitude: -2.274003348644879,
		OnGround:  1,
		//Pitch:     0,
	})

	time.Sleep(10 * time.Second)
}
