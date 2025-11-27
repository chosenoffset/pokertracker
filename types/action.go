package types

type Action struct {
	ID         int64
	HandID     int64
	Street     int
	Seat       int
	ActionType string
	Amount     int
	Sequence   int
}
