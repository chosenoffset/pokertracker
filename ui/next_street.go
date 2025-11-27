package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func BuildNextStreetUI(ui *UI) fyne.CanvasObject {
	gs := ui.gameState
	streetNames := []string{"Flop", "Turn", "River"}
	nextStreet := streetNames[gs.Street] // gs.Street is 0-indexed, so this gives next street name

	cardEntry := widget.NewEntry()
	if gs.Street == 0 {
		cardEntry.SetPlaceHolder("e.g., Ah7c2d")
	} else {
		cardEntry.SetPlaceHolder("e.g., Ks")
	}

	continueBtn := widget.NewButton(fmt.Sprintf("Deal %s", nextStreet), func() {
		switch gs.Street {
		case 0:
			gs.Flop = cardEntry.Text
		case 1:
			gs.Turn = cardEntry.Text
		case 2:
			gs.River = cardEntry.Text
		}
		gs.NextStreet()
		ui.refreshHandEntry()
	})

	return container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Enter %s cards:", nextStreet)),
		cardEntry,
		continueBtn,
	)
}
