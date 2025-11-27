package types

type Player struct {
	ID            int64
	TournamentID  int64
	Seat          int
	Name          string
	StartingStack int
	BustedHandNum int
}
