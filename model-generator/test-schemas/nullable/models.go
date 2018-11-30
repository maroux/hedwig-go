package hedwig

/**** BEGIN base definitions ****/

type Vehicle struct {
	ID string `json:"id"`
}

/**** END base definitions ****/

/**** BEGIN schema definitions ****/

// TripCreated represents the data for Hedwig message trip_created v1.*
type TripCreated struct {
	UserID  *string `json:"user_id"`
	Vehicle Vehicle `json:"vehicle"`
	Vin     string  `json:"vin"`
}

// NewTripCreatedData creates a new TripCreated struct
// this method can be used as NewData when registering callback
func NewTripCreatedData() interface{} { return new(TripCreated) }

/**** END schema definitions ****/
