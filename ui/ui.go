package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/chosenoffset/pokertracker/database"
	"github.com/chosenoffset/pokertracker/gameState"
	"github.com/chosenoffset/pokertracker/types"
)

type UI struct {
	app    fyne.App
	window fyne.Window
	db     *database.DB

	// Current state
	gameState *gameState.GameState
}

func NewUI(app fyne.App, db *database.DB) *UI {
	ui := &UI{
		app:    app,
		window: app.NewWindow("Poker Tracker"),
		db:     db,
	}
	ui.window.Resize(fyne.NewSize(800, 600))
	return ui
}

func (ui *UI) ShowHome() {
	tournaments, err := ui.db.ListTournaments()
	if err != nil {
		tournaments = []types.Tournament{}
	}

	var listItems []fyne.CanvasObject
	for _, t := range tournaments {
		t := t
		result := "In Progress"
		if t.FinishPlace > 0 {
			result = fmt.Sprintf("Finished %s", ordinal(t.FinishPlace))
		}

		label := widget.NewLabel(fmt.Sprintf("%s - $%.2f - %s",
			t.StartedAt.Format("Jan 2 3:04 PM"),
			float64(t.BuyIn)/100,
			result,
		))

		openBtn := widget.NewButton("Open", func() {
			ui.openTournament(t.ID)
		})

		deleteBtn := widget.NewButton("Delete", func() {
			_ = ui.db.DeleteTournament(t.ID)
			ui.ShowHome()
		})

		row := container.NewBorder(nil, nil, nil, container.NewHBox(openBtn, deleteBtn), label)
		listItems = append(listItems, row)
	}

	var listContainer fyne.CanvasObject
	if len(listItems) == 0 {
		listContainer = widget.NewLabel("No tournaments yet")
	} else {
		listContainer = container.NewVBox(listItems...)
	}

	newBtn := widget.NewButton("New Tournament", func() {
		ShowTournamentSetup(ui)
	})

	content := container.NewBorder(
		widget.NewLabel("Tournaments"),
		newBtn,
		nil,
		nil,
		container.NewVScroll(listContainer),
	)

	ui.window.SetContent(content)
}

func (ui *UI) openTournament(id int64) {
	gs, err := ui.db.LoadGameState(id)
	if err != nil {
		fmt.Println(err)
		return
	}
	ui.gameState = gs
	ShowHandEntry(ui)
}

func selectedIndex(sel *widget.Select) int {
	for i, opt := range sel.Options {
		if opt == sel.Selected {
			return i
		}
	}
	return 0
}

func parseIntOr(s string, fallback int) int {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return fallback
	}
	return n
}

func parseFloatOr(s string, fallback float64) float64 {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return fallback
	}
	return f
}

func (ui *UI) recordAction(seat int, action string, amount int) {
	gs := ui.gameState

	// Get current sequence number for this street
	actions, _ := ui.db.GetActionsForHand(gs.CurrentHandID)
	sequence := 0
	for _, a := range actions {
		if a.Street == gs.Street && a.Sequence >= sequence {
			sequence = a.Sequence + 1
		}
	}

	// Determine the amount to store in DB based on action type
	var dbAmount int
	if action == "raise" {
		// For raise, amount is the total "raise to" in chips
		// Store the total amount put in (call + raise)
		toCall := gs.CurrentBet - gs.StreetInvested[seat-1]
		raiseAmount := amount - gs.CurrentBet // How much above current bet
		dbAmount = toCall + raiseAmount
		gs.ApplyAction(seat, action, raiseAmount)
	} else if action == "call" {
		// Store the amount called
		dbAmount = gs.CurrentBet - gs.StreetInvested[seat-1]
		gs.ApplyAction(seat, action, amount)
	} else if action == "bet" {
		// Store the bet amount
		dbAmount = amount
		gs.ApplyAction(seat, action, amount)
	} else if action == "allin" {
		// Store the all-in amount (entire stack)
		dbAmount = gs.Players[seat-1].Stack
		gs.ApplyAction(seat, action, amount)
	} else {
		// fold, check - no amount
		dbAmount = 0
		gs.ApplyAction(seat, action, amount)
	}

	// Save action to DB
	_, _ = ui.db.CreateAction(gs.CurrentHandID, gs.Street, seat, action, dbAmount, sequence)

	ui.refreshHandEntry()
}

func (ui *UI) endHand(winnerSeat int) {
	gs := ui.gameState

	// Save pot size BEFORE awarding it
	finalPotSize := gs.Pot

	hand, _ := ui.db.GetHand(gs.CurrentHandID)
	if hand != nil {
		hand.HeroCards = database.ToNullString(gs.HeroCards)
		hand.Flop = database.ToNullString(gs.Flop)
		hand.Turn = database.ToNullString(gs.Turn)
		hand.River = database.ToNullString(gs.River)
		hand.WinnerSeat = winnerSeat
		hand.PotSize = finalPotSize
		_ = ui.db.UpdateHand(hand)
	}

	// Now award the pot (this sets gs.Pot = 0)
	gs.AwardPot(winnerSeat)

	for i := 0; i < 8; i++ {
		if gs.Players[i].IsAlive {
			_ = ui.db.SaveStackSnapshot(gs.CurrentHandID, i+1, gs.Players[i].Stack)
		}
	}

	for i := 0; i < 8; i++ {
		if gs.Players[i].IsAlive && gs.Players[i].Stack == 0 {
			gs.BustPlayer(i + 1)
			_ = ui.db.BustPlayer(gs.TournamentID, i+1, gs.HandNum)
		}
	}

	// Clear current hand and advance button for next hand
	gs.CurrentHandID = 0
	gs.AdvanceButton()
	// DON'T increment HandNum here - StartNewHand() does it

	// Return to hand entry - it will create the next hand when ready
	ShowHandEntry(ui)
}

func (ui *UI) endHandMultiPot(primaryWinnerSeat int) {
	gs := ui.gameState

	// Save pot size (already awarded in the multi-pot UI, so use sum of TotalInvested)
	finalPotSize := 0
	for i := 0; i < 8; i++ {
		finalPotSize += gs.TotalInvested[i]
	}

	hand, _ := ui.db.GetHand(gs.CurrentHandID)
	if hand != nil {
		hand.HeroCards = database.ToNullString(gs.HeroCards)
		hand.Flop = database.ToNullString(gs.Flop)
		hand.Turn = database.ToNullString(gs.Turn)
		hand.River = database.ToNullString(gs.River)
		hand.WinnerSeat = primaryWinnerSeat // Store main pot winner
		hand.PotSize = finalPotSize
		_ = ui.db.UpdateHand(hand)
	}

	// Pot already awarded in multi-pot UI, just set to 0
	gs.Pot = 0

	for i := 0; i < 8; i++ {
		if gs.Players[i].IsAlive {
			_ = ui.db.SaveStackSnapshot(gs.CurrentHandID, i+1, gs.Players[i].Stack)
		}
	}

	for i := 0; i < 8; i++ {
		if gs.Players[i].IsAlive && gs.Players[i].Stack == 0 {
			gs.BustPlayer(i + 1)
			_ = ui.db.BustPlayer(gs.TournamentID, i+1, gs.HandNum)
		}
	}

	// Clear current hand and advance button for next hand
	gs.CurrentHandID = 0
	gs.AdvanceButton()
	// DON'T increment HandNum here - StartNewHand() does it

	// Return to hand entry - it will create the next hand when ready
	ShowHandEntry(ui)
}

func (ui *UI) refreshHandEntry() {
	ShowHandEntry(ui)
}

func (ui *UI) isStreetComplete() bool {
	for i := 0; i < 8; i++ {
		if !ui.gameState.InHand[i] {
			continue
		}
		if ui.gameState.Players[i].Stack == 0 {
			// All-in, doesn't need to act
			continue
		}
		// Must have acted AND matched the current bet
		if !ui.gameState.HasActedThisStreet[i] || ui.gameState.StreetInvested[i] < ui.gameState.CurrentBet {
			return false
		}
	}

	return true
}

func (ui *UI) Run() {
	ui.ShowHome()
	ui.window.ShowAndRun()
}

func ordinal(n int) string {
	suffix := "th"
	switch n {
	case 1:
		suffix = "st"
	case 2:
		suffix = "nd"
	case 3:
		suffix = "rd"
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
