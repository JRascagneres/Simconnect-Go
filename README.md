# Simconnect-Go [![GoDoc](https://godoc.org/github.com/JRascagneres/Simconnect-Go?status.svg)](https://godoc.org/github.com/JRascagneres/Simconnect-Go) [![GitHub tag](https://img.shields.io/github/tag/JRascagneres/Simconnect-Go.svg)](https://github.com/JRascagneres/Simconnect-Go/releases) [![license](https://img.shields.io/github/license/JRascagneres/Simconnect-Go.svg)](https://github.com/JRascagneres/Simconnect-Go/blob/master/LICENSE)

This Simconnect library is used to communicate with primary Microsoft Flight Simulator 2020 using Golang. However, given its use of the SimConnect API it should be able to communicate with other flight simulators whcih also support the SimConnect API.

This attempts to be a little more of a high-level library attempting to abstract much of the SimConnect complexity from the user.

## Status

The current version is considered an early release and will therefore keep changing. Breaking changes will be avoided but may happen if required.

## Currently Implemented

- Request Data on User Aircraft (SimConnect_RequestDataOnSimObjectType)
- Request Data on Object (SimConnect_RequestDataOnSimObject)
- Load Flight Plan into Users Aircraft (SimConnect_FlightPlanLoad)
- Create parked ATC AI aircraft (SimConnect_AICreateParkedATCAircraft)
- Create Non ATC AI aircraft SimConnect_AICreateNonATCAircraft()
- Set Data on Object including user (SimConnect_SetDataOnSimObject)
- Create enroute ATC Aircraft (SimConnect_AICreateEnrouteATCAircraft)
- Set Flight Plan for AI ATC Aircraft (SimConnect_AISetAircraftFlightPlan)
- Remove Objects (SimConnect_AIRemoveObject)

## Install

`go get -u github.com/JRascagneres/Simconnect-Go`
and import with
`import (simconnect "github.com/JRascagneres/Simconnect-Go")`

## Very Simple Usage
Very simple example which starts the connection, grabs the current user data and closes the connection.
```
package main

import (
	"fmt"

	simconnect "github.com/JRascagneres/Simconnect-Go"
)

func main() {
	instance, err := simconnect.NewSimConnect("Simconnect-Go")
	if err != nil {
		panic(err)
	}

	report, err := instance.GetReport()
	if err != nil {
		panic(err)
	}

	fmt.Printf("User Altitude: %f\n", report.Altitude)

	if err := instance.Close(); err != nil {
		panic(err)
	}
}
```

## Documentation

All Documentation can be found through the [godoc](https://godoc.org/github.com/JRascagneres/Simconnect-Go)

## Contributing

Issues, feature requests or improvements welcome!

## License
This project is licensed under the [MIT License](LICENSE).
