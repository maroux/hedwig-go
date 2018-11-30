package hedwig

/**** BEGIN base definitions ****/

/**** END base definitions ****/

/**** BEGIN schema definitions ****/

// TripCreatedV1 represents the data for Hedwig message trip_created v1.*
type TripCreatedV1 struct {
	UserId    string `json:"user_id"`
	VehicleId string `json:"vehicle_id"`
	Vin       string `json:"vin,omitempty"`
}

// NewTripCreatedV1Data creates a new TripCreatedV1 struct
// this method can be used as NewData when registering callback
func NewTripCreatedV1Data() interface{} { return new(TripCreatedV1) }

// TripCreatedV2 represents the data for Hedwig message trip_created v2.*
type TripCreatedV2 struct {
	UserId    string `json:"user_id"`
	VehicleId string `json:"vehicle_id"`
	Vin       string `json:"vin"`
}

// NewTripCreatedV2Data creates a new TripCreatedV2 struct
// this method can be used as NewData when registering callback
func NewTripCreatedV2Data() interface{} { return new(TripCreatedV2) }

/**** END schema definitions ****/