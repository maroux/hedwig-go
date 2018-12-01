package hedwig

/**** BEGIN base definitions ****/

/**** END base definitions ****/

/**** BEGIN schema definitions ****/

// TripCreatedV1 represents the data for Hedwig message trip_created v1.*
type TripCreatedV1 struct {
	UserID    string `json:"user_id"`
	VehicleID string `json:"vehicle_id"`
	Vin       string `json:"vin"`
}

// NewTripCreatedV1Data creates a new TripCreatedV1 struct
// this method can be used as NewData when registering callback
func NewTripCreatedV1Data() interface{} { return new(TripCreatedV1) }

/**** END schema definitions ****/
