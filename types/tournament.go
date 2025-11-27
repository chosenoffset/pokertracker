package types

import "time"

type Tournament struct {
	ID          int64
	BuyIn       int
	StartedAt   time.Time
	HeroSeat    int
	InitialBTN  int
	FinishPlace int
}
