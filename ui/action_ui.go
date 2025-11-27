package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func buildActionUI(ui *UI) fyne.CanvasObject {
	gs := ui.gameState
	seat := gs.ActionOn
	player := gs.Players[seat-1]
	pos := gs.GetPosition(seat)
	blinds := gs.Blinds()

	playerStackBB := float64(player.Stack) / float64(blinds.BigBlind)
	invested := gs.StreetInvested[seat-1]
	investedBB := float64(invested) / float64(blinds.BigBlind)
	toCallBB := float64(gs.CurrentBet-invested) / float64(blinds.BigBlind)

	promptLabel := widget.NewLabel(fmt.Sprintf(
		"%s (%s) - Stack: %.1f BB - Invested: %.1f BB - To Call: %.1f BB",
		pos, player.Name, playerStackBB, investedBB, toCallBB,
	))

	// Build valid action buttons
	var buttons []fyne.CanvasObject

	// Fold is always valid
	buttons = append(buttons, widget.NewButton("Fold", func() {
		ui.recordAction(seat, "fold", 0)
	}))

	// Check only if no bet to call
	if gs.CurrentBet == invested {
		buttons = append(buttons, widget.NewButton("Check", func() {
			ui.recordAction(seat, "check", 0)
		}))
	}

	// Call if there's a bet to call
	if gs.CurrentBet > invested {
		buttons = append(buttons, widget.NewButton("Call", func() {
			ui.recordAction(seat, "call", 0)
		}))
	}

	// Limp (preflop only, when no raise yet - current bet is just BB)
	if gs.Street == 0 && gs.CurrentBet == blinds.BigBlind && invested < blinds.BigBlind {
		buttons = append(buttons, widget.NewButton("Limp", func() {
			ui.recordAction(seat, "call", 0) // Limp is just calling the BB
		}))
	}

	// Raise - needs amount input
	raiseEntry := widget.NewEntry()
	raiseEntry.SetPlaceHolder("BB amount")

	raiseBtn := widget.NewButton("Raise To", func() {
		bbAmount := parseFloatOr(raiseEntry.Text, 0)
		chipAmount := int(bbAmount * float64(blinds.BigBlind))
		ui.recordAction(seat, "raise", chipAmount)
	})

	buttons = append(buttons, container.NewHBox(raiseBtn, raiseEntry))

	// All-in
	buttons = append(buttons, widget.NewButton("All-In", func() {
		ui.recordAction(seat, "allin", 0)
	}))

	buttonRow := container.NewHBox(buttons...)

	return container.NewVBox(
		widget.NewLabel("Action on:"),
		promptLabel,
		buttonRow,
	)
}
