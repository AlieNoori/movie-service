package model

type (
	// RecordID defines a record id. Together with RecordType
	// identifies unique records across all types.
	RecordID string

	// RecordType defines a record type. Together with RecordI
	// identifies unique records across all types.
	RecordType string
)

// Existing record types.
const (
	RecordTypeMovie RecordType = "movie"
)

type (
	// UserID defines a user id.
	UserID string
	// RatingValue defines a value of a rating record.
	RatingValue int
)

// Rating defines an individual rating created by a user f
// some record.
type Rating struct {
	RecordID   string      `json:"record_id"`
	RecordType string      `json:"record_type"`
	UserID     UserID      `json:"user_id"`
	Value      RatingValue `json:"vlaue"`
}
