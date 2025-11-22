package model

type (
	// RecordID defines a record id. Together with RecordType
	// identifies unique records across all types.
	RecordID string

	// RecordType defines a record type. Together with RecordI
	// identifies unique records across all types.
	RecordType string

	// UserID defines a user id.
	UserID string

	// RatingValue defines a value of a rating record.
	RatingValue int
)

// Existing record types.
const (
	RecordTypeMovie RecordType = "movie"
)

// Rating defines an individual rating created by a user for some record.
type Rating struct {
	RecordID   RecordID    `json:"recordId"`
	RecordType RecordType  `json:"recordType"`
	UserID     UserID      `json:"userId"`
	Value      RatingValue `json:"value"`
}
