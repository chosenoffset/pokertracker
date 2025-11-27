package types

import "database/sql"

type Hand struct {
	ID           int64
	TournamentID int64
	HandNum      int
	Level        int
	ButtonSeat   int
	HeroCards    sql.NullString
	Flop         sql.NullString
	Turn         sql.NullString
	River        sql.NullString
	PotSize      int
	WinnerSeat   int
	Notes        sql.NullString
}
