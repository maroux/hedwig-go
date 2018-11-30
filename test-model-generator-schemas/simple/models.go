package hedwig

/**** BEGIN base definitions ****/

/**** END base definitions ****/

/**** BEGIN schema definitions ****/

// TripCreated represents the data for Hedwig message trip_created v1.*
type TripCreated struct {
	UserId    string `json:"user_id"`
	VehicleId string `json:"vehicle_id"`
	Vin       string `json:"vin"`
}

// NewTripCreatedData creates a new TripCreated struct
// this method can be used as NewData when registering callback
func NewTripCreatedData() interface{} { return new(TripCreated) }

/**** END schema definitions ****/
