package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/chosenoffset/pokertracker/gameState"
)

func BuildShowdownUI(ui *UI) fyne.CanvasObject {
	gs := ui.gameState

	// Calculate all pots
	pots := gs.CalculatePots()

	// If only one pot, use simple UI
	if len(pots) == 0 || len(pots) == 1 {
		return buildSimpleShowdownUI(ui)
	}

	// Multiple pots - build multi-pot UI
	return buildMultiPotShowdownUI(ui, pots)
}

func buildSimpleShowdownUI(ui *UI) fyne.CanvasObject {
	gs := ui.gameState
	var playerOptions []string
	for i := 0; i < 8; i++ {
		if gs.InHand[i] {
			pos := gs.GetPosition(i + 1)
			playerOptions = append(playerOptions, fmt.Sprintf("%s (%s)", pos, gs.Players[i].Name))
		}
	}

	winnerSelect := widget.NewSelect(playerOptions, func(s string) {})
	if len(playerOptions) > 0 {
		winnerSelect.SetSelected(playerOptions[0])
	}

	awardPot := func() {
		for i := 0; i < 8; i++ {
			if gs.InHand[i] {
				pos := gs.GetPosition(i + 1)
				selected := fmt.Sprintf("%s (%s)", pos, gs.Players[i].Name)
				if winnerSelect.Selected == selected {
					ui.endHand(i + 1)
					return
				}
			}
		}
	}

	awardBtn := widget.NewButton("Award Pot", awardPot)

	return container.NewVBox(
		widget.NewLabel("Showdown - Select Winner:"),
		winnerSelect,
		awardBtn,
	)
}

func buildMultiPotShowdownUI(ui *UI, pots []gameState.Pot) fyne.CanvasObject {
	gs := ui.gameState

	// Track winner selections for each pot
	potWinners := make([]int, len(pots))

	var elements []fyne.CanvasObject
	elements = append(elements, widget.NewLabel("Multiple Pots - Select Winners:"))
	elements = append(elements, widget.NewSeparator())

	// Build UI for each pot
	for potIdx, pot := range pots {
		// Pot label
		potName := "Main Pot"
		if potIdx > 0 {
			potName = fmt.Sprintf("Side Pot %d", potIdx)
		}

		blinds := gs.Blinds()
		potBB := float64(pot.Amount) / float64(blinds.BigBlind)

		potLabel := widget.NewLabel(fmt.Sprintf("%s: %.1f BB (%d chips)", potName, potBB, pot.Amount))
		potLabel.TextStyle.Bold = true
		elements = append(elements, potLabel)

		// Build player options for this pot (only eligible players)
		var playerOptions []string
		var seatMap []int // Maps option index to seat number

		for _, seat := range pot.EligibleSeats {
			pos := gs.GetPosition(seat)
			playerOptions = append(playerOptions, fmt.Sprintf("%s (%s)", pos, gs.Players[seat-1].Name))
			seatMap = append(seatMap, seat)
		}

		// Create closure to capture potIdx and seatMap
		localPotIdx := potIdx
		localSeatMap := seatMap

		winnerSelect := widget.NewSelect(playerOptions, func(s string) {
			// Find which seat was selected
			for i, opt := range playerOptions {
				if opt == s {
					potWinners[localPotIdx] = localSeatMap[i]
					break
				}
			}
		})

		if len(playerOptions) > 0 {
			winnerSelect.SetSelected(playerOptions[0])
			potWinners[localPotIdx] = seatMap[0]
		}

		elements = append(elements, winnerSelect)
		elements = append(elements, widget.NewSeparator())
	}

	// Award all pots button
	awardBtn := widget.NewButton("Award All Pots", func() {
		// Award each pot to its winner
		for potIdx, pot := range pots {
			winnerSeat := potWinners[potIdx]
			if winnerSeat > 0 {
				gs.Players[winnerSeat-1].Stack += pot.Amount
			}
		}
		gs.Pot = 0

		// End the hand (pass first pot winner as the "winner" for DB purposes)
		ui.endHandMultiPot(potWinners[0])
	})

	elements = append(elements, awardBtn)

	return container.NewVBox(elements...)
}
