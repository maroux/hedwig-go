package hedwig

/**** BEGIN base definitions ****/

type Vehicle10 struct {
	ID string `json:"id"`
}

type Vehicle20 struct {
	ID string `json:"id"`
}

/**** END base definitions ****/

/**** BEGIN schema definitions ****/

// TripCreatedV1 represents the data for Hedwig message trip_created v1.*
type TripCreatedV1 struct {
	UserID  string    `json:"user_id"`
	Vehicle Vehicle10 `json:"vehicle"`
	Vin     string    `json:"vin,omitempty"`
}

// NewTripCreatedV1Data creates a new TripCreatedV1 struct
// this method can be used as NewData when registering callback
func NewTripCreatedV1Data() interface{} { return new(TripCreatedV1) }

// TripCreatedV2 represents the data for Hedwig message trip_created v2.*
type TripCreatedV2 struct {
	UserID  string    `json:"user_id"`
	Vehicle Vehicle20 `json:"vehicle"`
	Vin     string    `json:"vin"`
}

// NewTripCreatedV2Data creates a new TripCreatedV2 struct
// this method can be used as NewData when registering callback
func NewTripCreatedV2Data() interface{} { return new(TripCreatedV2) }

/**** END schema definitions ****/
