package simconnect

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

// These aren't 'real' tests. This is simply for testing easily within the game.
func TestWork(t *testing.T) {
	instance, _ := NewSimConnect()

	instance.GetReport()

	for i := 0; i < 100; i++ {
		fmt.Println(instance.LoadNonATCAircraft("Generic Airliner Twin Engines Asobo 00", "G-42"+strconv.FormatInt(int64(i), 10), SimconnectDataInitPosition{
			Latitude:  53.34974539799793,
			Longitude: -2.274003348644879,
			Altitude:  235,
			Pitch:     0,
			Bank:      0,
			Heading:   0,
			OnGround:  false,
			Airspeed:  200,
		}, i))
	}

	time.Sleep(10 * time.Second)
}
