package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"github.com/chosenoffset/pokertracker/database"
	ui2 "github.com/chosenoffset/pokertracker/ui"
)

func main() {
	db, err := database.NewDB("pokertracker.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	a := app.New()
	ui := ui2.NewUI(a, db)
	ui.Run()
}
