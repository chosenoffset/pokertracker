package gameState

import (
	"github.com/chosenoffset/pokertracker/types"
)

type BlindLevel struct {
	SmallBlind int
	BigBlind   int
	Ante       int
}

var BlindStructure = []BlindLevel{
	{25, 50, 5},
	{40, 80, 10},
	{50, 100, 12},
	{75, 150, 20},
	{100, 200, 25},
	{125, 250, 30},
	{150, 300, 40},
	{200, 400, 50},
	{250, 500, 60},
	{300, 600, 70},
}

var Positions8Max = []string{"BTN", "SB", "BB", "UTG", "UTG+1", "MP", "HJ", "CO"}

type PlayerState struct {
	Seat    int
	Name    string
	Stack   int
	IsAlive bool
}

type GameState struct {
	TournamentID int64
	Level        int
	HandNum      int
	ButtonSeat   int
	HeroSeat     int
	Players      [8]PlayerState

	// Current hand state
	CurrentHandID int64
	Street        int
	Pot           int
	CurrentBet    int
	ActionOn      int
	HeroCards     string
	Flop          string
	Turn          string
	River         string

	// Track who's still in this hand
	InHand             [8]bool
	StreetInvested     [8]int // chips invested this street per player
	TotalInvested      [8]int // total chips invested in entire hand per player
	HasActedThisStreet [8]bool
}

func NewGameState(t types.Tournament, players []types.Player) *GameState {
	gs := &GameState{
		TournamentID: t.ID,
		Level:        0,
		HandNum:      0,
		ButtonSeat:   t.InitialBTN,
		HeroSeat:     t.HeroSeat,
	}

	for _, p := range players {
		idx := p.Seat - 1
		gs.Players[idx] = PlayerState{
			Seat:    p.Seat,
			Name:    p.Name,
			Stack:   p.StartingStack,
			IsAlive: true,
		}
	}

	return gs
}

func (gs *GameState) Blinds() BlindLevel {
	return BlindStructure[gs.Level]
}

func (gs *GameState) LevelUp() {
	if gs.Level < len(BlindStructure)-1 {
		gs.Level++
	}
}

func (gs *GameState) AdvanceButton() {
	next := gs.ButtonSeat
	for {
		next = (next % 8) + 1
		if gs.Players[next-1].IsAlive {
			gs.ButtonSeat = next
			return
		}
	}
}

func (gs *GameState) GetPosition(seat int) string {
	var activeSeats []int
	current := gs.ButtonSeat
	for i := 0; i < 8; i++ {
		if gs.Players[current-1].IsAlive {
			activeSeats = append(activeSeats, current)
		}
		current = (current % 8) + 1
	}

	for i, s := range activeSeats {
		if s == seat {
			if i < len(Positions8Max) {
				return Positions8Max[i]
			}
		}
	}

	return "?"
}

func (gs *GameState) SeatForPosition(pos string) int {
	var activeSeats []int
	current := gs.ButtonSeat

	for i := 0; i < 8; i++ {
		if gs.Players[current-1].IsAlive {
			activeSeats = append(activeSeats, current)
		}
		current = (current % 8) + 1
	}

	for i, p := range Positions8Max {
		if p == pos && i < len(activeSeats) {
			return activeSeats[i]
		}
	}

	return 0
}

func (gs *GameState) StackInBB(seat int) float64 {
	return float64(gs.Players[seat-1].Stack) / float64(gs.Blinds().BigBlind)
}

func (gs *GameState) StartNewHand() {
	gs.HandNum++
	gs.Street = 0
	gs.Pot = 0
	gs.CurrentBet = 0
	gs.HeroCards = ""
	gs.Flop = ""
	gs.Turn = ""
	gs.River = ""
	gs.StreetInvested = [8]int{}
	gs.TotalInvested = [8]int{}

	// Everyone alive is in the hand
	for i := 0; i < 8; i++ {
		gs.InHand[i] = gs.Players[i].IsAlive
	}

	// Post antes
	blinds := gs.Blinds()
	for i := 0; i < 8; i++ {
		if gs.Players[i].IsAlive {
			gs.Players[i].Stack -= blinds.Ante
			gs.Pot += blinds.Ante
			gs.TotalInvested[i] = blinds.Ante
		}
	}

	// Post blinds
	sbSeat := gs.SeatForPosition("SB")
	bbSeat := gs.SeatForPosition("BB")

	gs.Players[sbSeat-1].Stack -= blinds.SmallBlind
	gs.StreetInvested[sbSeat-1] = blinds.SmallBlind
	gs.TotalInvested[sbSeat-1] += blinds.SmallBlind
	gs.Pot += blinds.SmallBlind

	gs.Players[bbSeat-1].Stack -= blinds.BigBlind
	gs.StreetInvested[bbSeat-1] = blinds.BigBlind
	gs.TotalInvested[bbSeat-1] += blinds.BigBlind
	gs.Pot += blinds.BigBlind

	gs.CurrentBet = blinds.BigBlind
	gs.ActionOn = gs.SeatForPosition("UTG")
}

func (gs *GameState) NextStreet() {
	gs.Street++
	gs.CurrentBet = 0
	gs.StreetInvested = [8]int{}
	gs.HasActedThisStreet = [8]bool{} // Reset

	// Action starts at first active player after button
	current := gs.ButtonSeat
	start := current
	for {
		current = (current % 8) + 1
		if gs.InHand[current-1] && gs.Players[current-1].Stack > 0 {
			gs.ActionOn = current
			return
		}
		// If we've looped back to start, no one has chips - all players all-in
		if current == start {
			gs.ActionOn = 0 // No action needed, everyone is all-in
			return
		}
	}
}

func (gs *GameState) ApplyAction(seat int, action string, amount int) {
	idx := seat - 1
	gs.HasActedThisStreet[idx] = true // Mark as acted

	switch action {
	case "fold":
		gs.InHand[idx] = false
	case "check":
		// nothing changes
	case "call":
		toCall := gs.CurrentBet - gs.StreetInvested[idx]
		gs.Players[idx].Stack -= toCall
		gs.StreetInvested[idx] = gs.CurrentBet
		gs.TotalInvested[idx] += toCall
		gs.Pot += toCall
	case "bet", "raise":
		toCall := gs.CurrentBet - gs.StreetInvested[idx]
		totalPut := toCall + amount
		gs.Players[idx].Stack -= totalPut
		gs.StreetInvested[idx] += totalPut
		gs.TotalInvested[idx] += totalPut
		gs.CurrentBet = gs.StreetInvested[idx]
		gs.Pot += totalPut

		// Reset HasActed for everyone else still in hand (they need to respond to the raise)
		for i := 0; i < 8; i++ {
			if i != idx && gs.InHand[i] && gs.Players[i].Stack > 0 {
				gs.HasActedThisStreet[i] = false
			}
		}
	case "allin":
		allinAmount := gs.Players[idx].Stack
		gs.Pot += allinAmount
		gs.StreetInvested[idx] += allinAmount
		gs.TotalInvested[idx] += allinAmount
		if gs.StreetInvested[idx] > gs.CurrentBet {
			gs.CurrentBet = gs.StreetInvested[idx]
			// Reset HasActed for others if this raises the bet
			for i := 0; i < 8; i++ {
				if i != idx && gs.InHand[i] && gs.Players[i].Stack > 0 {
					gs.HasActedThisStreet[i] = false
				}
			}
		}
		gs.Players[idx].Stack = 0
	}

	gs.AdvanceAction()
}

func (gs *GameState) AdvanceAction() {
	start := gs.ActionOn
	current := start
	for {
		current = (current % 8) + 1
		if current == start {
			return // back to start, street might be over
		}
		if gs.InHand[current-1] && gs.Players[current-1].Stack > 0 {
			gs.ActionOn = current
			return
		}
	}
}

func (gs *GameState) BustPlayer(seat int) {
	gs.Players[seat-1].IsAlive = false
	gs.Players[seat-1].Stack = 0
}

func (gs *GameState) AwardPot(seat int) {
	gs.Players[seat-1].Stack += gs.Pot
	gs.Pot = 0
}

// Pot represents a main pot or side pot
type Pot struct {
	Amount        int   // Chips in this pot
	EligibleSeats []int // Seats that can contest this pot
	Cap           int   // Investment cap for this pot (0 for main pot with no all-ins)
}

// CalculatePots calculates all pots (main and side) based on player investments
func (gs *GameState) CalculatePots() []Pot {
	var pots []Pot

	// Collect unique investment amounts from players still in hand
	type investment struct {
		seat   int
		amount int
	}

	var investments []investment
	for i := 0; i < 8; i++ {
		if gs.InHand[i] && gs.TotalInvested[i] > 0 {
			investments = append(investments, investment{seat: i + 1, amount: gs.TotalInvested[i]})
		}
	}

	if len(investments) == 0 {
		return pots
	}

	// Sort by investment amount (ascending)
	for i := 0; i < len(investments); i++ {
		for j := i + 1; j < len(investments); j++ {
			if investments[i].amount > investments[j].amount {
				investments[i], investments[j] = investments[j], investments[i]
			}
		}
	}

	// Track what's already been allocated
	allocated := [8]int{}
	previousCap := 0

	// Build pots from smallest investment to largest
	for i := 0; i < len(investments); i++ {
		currentCap := investments[i].amount

		// Skip if this investment level is same as previous (already processed)
		if currentCap == previousCap {
			continue
		}

		// Calculate pot amount: (cap - previousCap) × number of eligible players
		potAmount := 0
		var eligibleSeats []int

		for j := 0; j < 8; j++ {
			if gs.InHand[j] && gs.TotalInvested[j] >= currentCap {
				contribution := currentCap - allocated[j]
				potAmount += contribution
				allocated[j] = currentCap
				eligibleSeats = append(eligibleSeats, j+1)
			}
		}

		if potAmount > 0 && len(eligibleSeats) > 0 {
			pots = append(pots, Pot{
				Amount:        potAmount,
				EligibleSeats: eligibleSeats,
				Cap:           currentCap,
			})
		}

		previousCap = currentCap
	}

	return pots
}
