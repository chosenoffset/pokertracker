package ui

import (
	"fmt"
	"strings"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// formatCard converts a card string like "Ah" to "A♥"
func formatCard(card string) string {
	if len(card) < 2 {
		return card
	}

	rank := string(card[0])
	suit := strings.ToLower(string(card[1]))

	var suitSymbol string
	switch suit {
	case "h":
		suitSymbol = "♥"
	case "d":
		suitSymbol = "♦"
	case "c":
		suitSymbol = "♣"
	case "s":
		suitSymbol = "♠"
	default:
		suitSymbol = suit
	}

	return rank + suitSymbol
}

// formatCards parses a card string like "Ah7c2d" and returns formatted cards
func formatCards(cardString string) []string {
	if cardString == "" {
		return nil
	}

	var cards []string
	cardString = strings.TrimSpace(cardString)

	// Parse cards - expecting format like "Ah7c2d" or "10hKs"
	i := 0
	for i < len(cardString) {
		if i+1 < len(cardString) {
			// Check for 10
			if cardString[i] == '1' && i+2 < len(cardString) && cardString[i+1] == '0' {
				cards = append(cards, formatCard(cardString[i:i+3]))
				i += 3
			} else {
				cards = append(cards, formatCard(cardString[i:i+2]))
				i += 2
			}
		} else {
			break
		}
	}

	return cards
}

// buildCardDisplay creates a large, centered display of cards with suit symbols
func buildCardDisplay(heroCards, flop, turn, river string) fyne.CanvasObject {
	var elements []fyne.CanvasObject

	// Hero cards section
	if heroCards != "" {
		heroLabel := widget.NewLabel("Hero:")
		heroLabel.TextStyle.Bold = true

		heroCardsFormatted := formatCards(heroCards)
		heroCardsText := canvas.NewText(strings.Join(heroCardsFormatted, " "), color.White)
		heroCardsText.TextSize = 32
		heroCardsText.TextStyle.Bold = true

		elements = append(elements,
			container.NewCenter(heroLabel),
			container.NewCenter(heroCardsText),
			widget.NewSeparator(),
		)
	}

	// Board cards section
	var boardCards []string
	if flop != "" {
		boardCards = append(boardCards, formatCards(flop)...)
	}
	if turn != "" {
		boardCards = append(boardCards, formatCards(turn)...)
	}
	if river != "" {
		boardCards = append(boardCards, formatCards(river)...)
	}

	if len(boardCards) > 0 {
		boardLabel := widget.NewLabel("Board:")
		boardLabel.TextStyle.Bold = true

		boardText := canvas.NewText(strings.Join(boardCards, "  "), color.White)
		boardText.TextSize = 40
		boardText.TextStyle.Bold = true

		elements = append(elements,
			container.NewCenter(boardLabel),
			container.NewCenter(boardText),
		)
	}

	if len(elements) == 0 {
		return widget.NewLabel("")
	}

	return container.NewVBox(elements...)
}

// buildPlayerBox creates a player info box showing name and stack
func buildPlayerBox(ui *UI, seat int) fyne.CanvasObject {
	gs := ui.gameState
	player := gs.Players[seat-1]

	if !player.IsAlive {
		return widget.NewLabel("")
	}

	blinds := gs.Blinds()
	stackBB := float64(player.Stack) / float64(blinds.BigBlind)

	nameLabel := widget.NewLabel(player.Name)
	nameLabel.TextStyle.Bold = true

	stackLabel := widget.NewLabel(fmt.Sprintf("%.1f BB", stackBB))

	// Highlight if action is on this player
	box := container.NewVBox(nameLabel, stackLabel)
	if gs.ActionOn == seat {
		nameLabel.TextStyle.Bold = true
		// Could add border or background color here
	}

	return container.NewCenter(box)
}

// buildTableLayout creates the 8-seat poker table layout
func buildTableLayout(ui *UI, centerContent fyne.CanvasObject) fyne.CanvasObject {
	// Seat positions (8-max):
	// Seat 1: Bottom center (hero)
	// Seat 2: Bottom left
	// Seat 3: Left middle
	// Seat 4: Top left
	// Seat 5: Top center
	// Seat 6: Top right
	// Seat 7: Right middle
	// Seat 8: Bottom right

	seat1 := buildPlayerBox(ui, 1) // Bottom center
	seat2 := buildPlayerBox(ui, 2) // Bottom left
	seat3 := buildPlayerBox(ui, 3) // Left middle
	seat4 := buildPlayerBox(ui, 4) // Top left
	seat5 := buildPlayerBox(ui, 5) // Top center
	seat6 := buildPlayerBox(ui, 6) // Top right
	seat7 := buildPlayerBox(ui, 7) // Right middle
	seat8 := buildPlayerBox(ui, 8) // Bottom right

	// Top row: seats 4, 5, 6 - evenly spaced across width
	topRow := container.NewBorder(
		nil, nil,
		seat4, // left
		seat6, // right
		container.NewCenter(seat5), // center
	)

	// Middle row: seat 3, center content, seat 7
	middleRow := container.NewBorder(
		nil, nil,
		seat3, // left
		seat7, // right
		centerContent,
	)

	// Bottom row: seats 2, 1, 8 - evenly spaced across width
	bottomRow := container.NewBorder(
		nil, nil,
		seat2, // left
		seat8, // right
		container.NewCenter(seat1), // center
	)

	// Combine all rows
	return container.NewBorder(
		topRow,    // top
		bottomRow, // bottom
		nil, nil,
		middleRow, // center
	)
}

func ShowHandEntry(ui *UI) {
	gs := ui.gameState

	// If we're starting fresh (not mid-hand), start the new hand
	if gs.CurrentHandID == 0 {
		gs.StartNewHand()

		hand, err := ui.db.CreateHand(gs.TournamentID, gs.HandNum, gs.Level, gs.ButtonSeat)
		if err != nil {
			fmt.Println("Error creating hand:", err)
			return
		}
		gs.CurrentHandID = hand.ID
	}

	// Top bar - hand info
	blinds := gs.Blinds()
	infoLabel := widget.NewLabel(fmt.Sprintf(
		"Hand #%d | Level %d: %d/%d/%d",
		gs.HandNum, gs.Level+1, blinds.SmallBlind, blinds.BigBlind, blinds.Ante,
	))

	levelUpBtn := widget.NewButton("Level Up", func() {
		// Check if we're at the very start of a hand (no actions taken yet)
		// We detect this by checking if the pot exactly equals initial blinds/antes
		oldBlinds := gs.Blinds()
		expectedInitialPot := oldBlinds.SmallBlind + oldBlinds.BigBlind + (oldBlinds.Ante * 8)

		if gs.Street == 0 && gs.Pot == expectedInitialPot {
			// We're at the start of the hand - need to adjust blinds/antes to new level
			gs.LevelUp()
			newBlinds := gs.Blinds()

			// Calculate the additional chips needed for new level
			anteDiff := (newBlinds.Ante - oldBlinds.Ante) * 8
			sbDiff := newBlinds.SmallBlind - oldBlinds.SmallBlind
			bbDiff := newBlinds.BigBlind - oldBlinds.BigBlind

			// Add difference to pot
			gs.Pot += anteDiff + sbDiff + bbDiff

			// Deduct additional antes from all alive players
			for i := 0; i < 8; i++ {
				if gs.Players[i].IsAlive {
					gs.Players[i].Stack -= (newBlinds.Ante - oldBlinds.Ante)
				}
			}

			// Deduct additional blinds from blind positions
			sbSeat := gs.SeatForPosition("SB")
			bbSeat := gs.SeatForPosition("BB")
			gs.Players[sbSeat-1].Stack -= sbDiff
			gs.Players[bbSeat-1].Stack -= bbDiff

			// Update street invested and current bet to reflect new blinds
			gs.StreetInvested[sbSeat-1] = newBlinds.SmallBlind
			gs.StreetInvested[bbSeat-1] = newBlinds.BigBlind
			gs.CurrentBet = newBlinds.BigBlind

			ui.refreshHandEntry()
		} else {
			// Mid-hand or after actions - just change level for next hand
			gs.LevelUp()
			ui.refreshHandEntry()
		}
	})

	topBar := container.NewBorder(nil, nil, infoLabel, levelUpBtn)

	// Pot and bet info
	currentBetBB := float64(gs.CurrentBet) / float64(blinds.BigBlind)
	potBB := float64(gs.Pot) / float64(blinds.BigBlind)

	potLabel := widget.NewLabel(fmt.Sprintf("Pot: %.1f BB (%d chips)", potBB, gs.Pot))
	betLabel := widget.NewLabel(fmt.Sprintf("Current Bet: %.1f BB (%d chips)", currentBetBB, gs.CurrentBet))

	potInfo := container.NewHBox(potLabel, widget.NewLabel("  |  "), betLabel)

	// Street indicator
	streetNames := []string{"Preflop", "Flop", "Turn", "River"}
	streetLabel := widget.NewLabel(fmt.Sprintf("Street: %s", streetNames[gs.Street]))

	// Build card display for center of table
	cardDisplay := buildCardDisplay(gs.HeroCards, gs.Flop, gs.Turn, gs.River)

	// Build full table layout with 8 seats around the cards
	tableLayout := buildTableLayout(ui, cardDisplay)

	// Check if hand is over (only one player left)
	playersInHand := 0
	lastPlayerSeat := 0
	for i := 0; i < 8; i++ {
		if gs.InHand[i] {
			playersInHand++
			lastPlayerSeat = i + 1
		}
	}

	var actionArea fyne.CanvasObject

	if playersInHand == 1 {
		// Hand is over, one player wins
		winnerName := gs.Players[lastPlayerSeat-1].Name
		actionArea = container.NewVBox(
			widget.NewLabel(fmt.Sprintf("%s wins the pot!", winnerName)),
			widget.NewButton("End Hand", func() {
				ui.endHand(lastPlayerSeat)
			}),
		)
	} else if ui.isStreetComplete() {
		// Street is complete, need next street or showdown
		if gs.Street >= 3 {
			// River complete, need showdown
			actionArea = BuildShowdownUI(ui)
		} else {
			// Prompt for next street
			actionArea = BuildNextStreetUI(ui)
		}
	} else {
		// Action needed from current player
		actionArea = buildActionUI(ui)
	}

	// Navigation
	homeBtn := widget.NewButton("Back to Home", func() {
		ui.ShowHome()
	})

	// Card input section at bottom
	heroCardsEntry := widget.NewEntry()
	heroCardsEntry.SetText(gs.HeroCards)
	heroCardsEntry.SetPlaceHolder("e.g., AhKd")
	heroCardsEntry.OnChanged = func(s string) {
		gs.HeroCards = s
		// Don't refresh on every keystroke - it steals focus
	}

	var boardInputs fyne.CanvasObject
	if gs.Street >= 1 {
		flopEntry := widget.NewEntry()
		flopEntry.SetText(gs.Flop)
		flopEntry.SetPlaceHolder("e.g., Ah7c2d")
		flopEntry.OnChanged = func(s string) {
			gs.Flop = s
			// Don't refresh on every keystroke - it steals focus
		}

		if gs.Street >= 2 {
			turnEntry := widget.NewEntry()
			turnEntry.SetText(gs.Turn)
			turnEntry.SetPlaceHolder("e.g., Ks")
			turnEntry.OnChanged = func(s string) {
				gs.Turn = s
				// Don't refresh on every keystroke - it steals focus
			}

			if gs.Street >= 3 {
				riverEntry := widget.NewEntry()
				riverEntry.SetText(gs.River)
				riverEntry.SetPlaceHolder("e.g., 3d")
				riverEntry.OnChanged = func(s string) {
					gs.River = s
					// Don't refresh on every keystroke - it steals focus
				}

				boardInputs = container.NewVBox(
					container.NewBorder(nil, nil, widget.NewLabel("Flop:"), nil, flopEntry),
					container.NewBorder(nil, nil, widget.NewLabel("Turn:"), nil, turnEntry),
					container.NewBorder(nil, nil, widget.NewLabel("River:"), nil, riverEntry),
				)
			} else {
				boardInputs = container.NewVBox(
					container.NewBorder(nil, nil, widget.NewLabel("Flop:"), nil, flopEntry),
					container.NewBorder(nil, nil, widget.NewLabel("Turn:"), nil, turnEntry),
				)
			}
		} else {
			boardInputs = container.NewBorder(nil, nil, widget.NewLabel("Flop:"), nil, flopEntry)
		}
	} else {
		boardInputs = widget.NewLabel("")
	}

	cardInputSection := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabel("Card Input:"),
		container.NewBorder(nil, nil, widget.NewLabel("Hero:"), nil, heroCardsEntry),
		boardInputs,
	)

	// Top section with info
	topSection := container.NewVBox(
		topBar,
		widget.NewSeparator(),
		potInfo,
		streetLabel,
	)

	// Bottom section with inputs and actions
	bottomSection := container.NewVBox(
		cardInputSection,
		widget.NewSeparator(),
		actionArea,
		widget.NewSeparator(),
		homeBtn,
	)

	// Use border layout: top info, center table, bottom inputs/actions
	content := container.NewBorder(
		topSection,
		bottomSection,
		nil,
		nil,
		tableLayout,
	)

	ui.window.SetContent(content)
}
