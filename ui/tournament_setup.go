package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/chosenoffset/pokertracker/gameState"
	"github.com/chosenoffset/pokertracker/types"
)

func ShowTournamentSetup(ui *UI) {
	buttonSeatSelect := widget.NewSelect(
		[]string{"1", "2", "3", "4", "5", "6", "7", "8"},
		func(s string) {},
	)
	buttonSeatSelect.SetSelected("1")

	startingHandEntry := widget.NewEntry()
	startingHandEntry.SetText("1")

	// Player name and stack entries
	playerNames := make([]*widget.Entry, 8)
	playerStacks := make([]*widget.Entry, 8)
	playerRows := make([]fyne.CanvasObject, 8)

	for i := 0; i < 8; i++ {
		playerNames[i] = widget.NewEntry()
		playerStacks[i] = widget.NewEntry()
		playerStacks[i].SetText("100")

		if i == 0 {
			playerNames[i].SetText("ChosenOffset")
		} else {
			playerNames[i].SetPlaceHolder(fmt.Sprintf("Villain %d", i))
		}

		playerRows[i] = container.NewBorder(
			nil, nil,
			widget.NewLabel(fmt.Sprintf("Seat %d:", i+1)),
			container.NewHBox(
				widget.NewLabel("Stack:"),
				container.NewGridWrap(fyne.NewSize(80, 36), playerStacks[i]),
			),
			playerNames[i],
		)
	}

	form := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Button Seat:"),
			buttonSeatSelect,
		),
		container.NewHBox(
			widget.NewLabel("Starting Hand #:"),
			container.NewGridWrap(fyne.NewSize(60, 36), startingHandEntry),
		),
		widget.NewSeparator(),
		widget.NewLabel("Players:"),
	)

	for _, row := range playerRows {
		form.Add(row)
	}

	cancelBtn := widget.NewButton("Cancel", func() {
		ui.ShowHome()
	})

	startBtn := widget.NewButton("Start Tournament", func() {
		buttonSeat := selectedIndex(buttonSeatSelect) + 1
		startingHand := parseIntOr(startingHandEntry.Text, 1)

		tournament, err := ui.db.CreateTournament(500, 1, buttonSeat)
		if err != nil {
			fmt.Println("Error creating tournament:", err)
			return
		}

		var players []types.Player
		for i := 0; i < 8; i++ {
			name := playerNames[i].Text
			if name == "" {
				name = fmt.Sprintf("Villain %d", i)
			}

			stack := parseFloatOr(playerStacks[i].Text, 100.0)

			fmt.Printf("Creating player %s with stack %d\n", name, stack)
			fmt.Printf("Stack converted to chips: %d\n", stack*50)
			//Yea, magic number, but that's what I'm running with for now.  50 is level 1 BB, so convert BB to chips before storing in DB
			player, err := ui.db.CreatePlayer(tournament.ID, i+1, name, int(stack*50))
			if err != nil {
				fmt.Println("Error creating player:", err)
				return
			}
			players = append(players, *player)
		}

		ui.gameState = gameState.NewGameState(*tournament, players)
		// Set HandNum to one less than starting hand because StartNewHand() increments it
		ui.gameState.HandNum = startingHand - 1
		ShowHandEntry(ui)
	})

	buttons := container.NewHBox(cancelBtn, startBtn)

	content := container.NewBorder(
		widget.NewLabel("New Tournament Setup"),
		buttons,
		nil,
		nil,
		container.NewVScroll(form),
	)

	ui.window.SetContent(content)
}
