package database

import (
	"database/sql"
	_ "embed"
	"time"

	"github.com/chosenoffset/pokertracker/gameState"
	"github.com/chosenoffset/pokertracker/types"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

type DB struct {
	conn *sql.DB
}

func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) init() error {
	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) Close() {
	if db.conn != nil {
		_ = db.conn.Close()
	}
}

func (db *DB) CreateTournament(buyIn int, heroSeat int, initialBTN int) (*types.Tournament, error) {
	result, err := db.conn.Exec(
		`INSERT INTO tournaments (buy_in, started_at, hero_seat, initial_btn) VALUES (?, ?, ?, ?)`,
		buyIn, time.Now(), heroSeat, initialBTN,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &types.Tournament{
		ID:         id,
		BuyIn:      buyIn,
		StartedAt:  time.Now(),
		HeroSeat:   heroSeat,
		InitialBTN: initialBTN,
	}, nil
}

func (db *DB) DeleteTournament(id int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	// Delete in order due to foreign keys
	_, err = tx.Exec(`DELETE FROM stack_snapshots WHERE hand_id IN (SELECT id FROM hands WHERE tournament_id = ?)`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM actions WHERE hand_id IN (SELECT id FROM hands WHERE tournament_id = ?)`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM hands WHERE tournament_id = ?`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM players WHERE tournament_id = ?`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM tournaments WHERE id = ?`, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (db *DB) GetTournament(id int64) (*types.Tournament, error) {
	row := db.conn.QueryRow(
		`SELECT id, buy_in, started_at, hero_seat, initial_btn, finish_place FROM tournaments WHERE id = ?`,
		id,
	)

	var t types.Tournament
	err := row.Scan(&t.ID, &t.BuyIn, &t.StartedAt, &t.HeroSeat, &t.InitialBTN, &t.FinishPlace)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (db *DB) UpdateTournamentFinish(id int64, place int) error {
	_, err := db.conn.Exec(`UPDATE tournaments SET finish_place = ? WHERE id = ?`, place, id)
	return err
}

func (db *DB) ListTournaments() ([]types.Tournament, error) {
	rows, err := db.conn.Query(
		`SELECT id, buy_in, started_at, hero_seat, initial_btn, finish_place FROM tournaments ORDER BY started_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tournaments []types.Tournament
	for rows.Next() {
		var t types.Tournament
		if err := rows.Scan(&t.ID, &t.BuyIn, &t.StartedAt, &t.HeroSeat, &t.InitialBTN, &t.FinishPlace); err != nil {
			return nil, err
		}
		tournaments = append(tournaments, t)
	}
	return tournaments, nil
}

// Player operations

func (db *DB) CreatePlayer(tournamentID int64, seat int, name string, startingStack int) (*types.Player, error) {
	result, err := db.conn.Exec(
		`INSERT INTO players (tournament_id, seat, name, starting_stack) VALUES (?, ?, ?, ?)`,
		tournamentID, seat, name, startingStack,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &types.Player{
		ID:            id,
		TournamentID:  tournamentID,
		Seat:          seat,
		Name:          name,
		StartingStack: startingStack,
	}, nil
}

func (db *DB) GetPlayers(tournamentID int64) ([]types.Player, error) {
	rows, err := db.conn.Query(
		`SELECT id, tournament_id, seat, name, starting_stack, busted_hand_num FROM players WHERE tournament_id = ? ORDER BY seat`,
		tournamentID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var players []types.Player
	for rows.Next() {
		var p types.Player
		if err := rows.Scan(&p.ID, &p.TournamentID, &p.Seat, &p.Name, &p.StartingStack, &p.BustedHandNum); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, nil
}

func (db *DB) BustPlayer(tournamentID int64, seat int, handNum int) error {
	_, err := db.conn.Exec(
		`UPDATE players SET busted_hand_num = ? WHERE tournament_id = ? AND seat = ?`,
		handNum, tournamentID, seat,
	)
	return err
}

// Hand operations

func (db *DB) CreateHand(tournamentID int64, handNum int, level int, buttonSeat int) (*types.Hand, error) {
	result, err := db.conn.Exec(
		`INSERT INTO hands (tournament_id, hand_num, level, button_seat) VALUES (?, ?, ?, ?)`,
		tournamentID, handNum, level, buttonSeat,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &types.Hand{
		ID:           id,
		TournamentID: tournamentID,
		HandNum:      handNum,
		Level:        level,
		ButtonSeat:   buttonSeat,
	}, nil
}

func (db *DB) UpdateHand(hand *types.Hand) error {
	_, err := db.conn.Exec(
		`UPDATE hands SET hero_cards = ?, flop = ?, turn = ?, river = ?, pot_size = ?, winner_seat = ?, notes = ? WHERE id = ?`,
		nullStringValue(hand.HeroCards), nullStringValue(hand.Flop), nullStringValue(hand.Turn), nullStringValue(hand.River), hand.PotSize, hand.WinnerSeat, nullStringValue(hand.Notes), hand.ID,
	)
	return err
}

func nullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// Helper to convert string to sql.NullString
func ToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// Helper to convert sql.NullString to string
func FromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func (db *DB) GetHand(id int64) (*types.Hand, error) {
	row := db.conn.QueryRow(
		`SELECT id, tournament_id, hand_num, level, button_seat, hero_cards, flop, turn, river, pot_size, winner_seat, notes FROM hands WHERE id = ?`,
		id,
	)

	var h types.Hand
	err := row.Scan(&h.ID, &h.TournamentID, &h.HandNum, &h.Level, &h.ButtonSeat, &h.HeroCards, &h.Flop, &h.Turn, &h.River, &h.PotSize, &h.WinnerSeat, &h.Notes)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (db *DB) GetHandsForTournament(tournamentID int64) ([]types.Hand, error) {
	rows, err := db.conn.Query(
		`SELECT id, tournament_id, hand_num, level, button_seat, hero_cards, flop, turn, river, pot_size, winner_seat, notes FROM hands WHERE tournament_id = ? ORDER BY hand_num`,
		tournamentID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var hands []types.Hand
	for rows.Next() {
		var h types.Hand
		if err := rows.Scan(&h.ID, &h.TournamentID, &h.HandNum, &h.Level, &h.ButtonSeat, &h.HeroCards, &h.Flop, &h.Turn, &h.River, &h.PotSize, &h.WinnerSeat, &h.Notes); err != nil {
			return nil, err
		}
		hands = append(hands, h)
	}
	return hands, nil
}

// Action operations

func (db *DB) CreateAction(handID int64, street int, seat int, actionType string, amount int, sequence int) (*types.Action, error) {
	result, err := db.conn.Exec(
		`INSERT INTO actions (hand_id, street, seat, action_type, amount, sequence) VALUES (?, ?, ?, ?, ?, ?)`,
		handID, street, seat, actionType, amount, sequence,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &types.Action{
		ID:         id,
		HandID:     handID,
		Street:     street,
		Seat:       seat,
		ActionType: actionType,
		Amount:     amount,
		Sequence:   sequence,
	}, nil
}

func (db *DB) GetActionsForHand(handID int64) ([]types.Action, error) {
	rows, err := db.conn.Query(
		`SELECT id, hand_id, street, seat, action_type, amount, sequence FROM actions WHERE hand_id = ? ORDER BY street, sequence`,
		handID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var actions []types.Action
	for rows.Next() {
		var a types.Action
		if err := rows.Scan(&a.ID, &a.HandID, &a.Street, &a.Seat, &a.ActionType, &a.Amount, &a.Sequence); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, nil
}

// Stack snapshot operations

func (db *DB) SaveStackSnapshot(handID int64, seat int, stack int) error {
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO stack_snapshots (hand_id, seat, stack) VALUES (?, ?, ?)`,
		handID, seat, stack,
	)
	return err
}

func (db *DB) GetStackSnapshots(handID int64) (map[int]int, error) {
	rows, err := db.conn.Query(
		`SELECT seat, stack FROM stack_snapshots WHERE hand_id = ?`,
		handID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	stacks := make(map[int]int)
	for rows.Next() {
		var seat, stack int
		if err := rows.Scan(&seat, &stack); err != nil {
			return nil, err
		}
		stacks[seat] = stack
	}
	return stacks, nil
}

// Load full game state from DB for a tournament

func (db *DB) LoadGameState(tournamentID int64) (*gameState.GameState, error) {
	tournament, err := db.GetTournament(tournamentID)
	if err != nil {
		return nil, err
	}

	players, err := db.GetPlayers(tournamentID)
	if err != nil {
		return nil, err
	}

	gs := gameState.NewGameState(*tournament, players)

	hands, err := db.GetHandsForTournament(tournamentID)
	if err != nil {
		return nil, err
	}

	if len(hands) > 0 {
		lastHand := hands[len(hands)-1]

		// Check if last hand is complete (has a winner)
		if lastHand.WinnerSeat == 0 {
			// Hand is incomplete - resume it by loading state from the previous complete hand
			// Then set CurrentHandID so we continue this hand instead of creating a new one
			if len(hands) > 1 {
				// Load stacks from the previous complete hand
				previousHand := hands[len(hands)-2]
				stacks, err := db.GetStackSnapshots(previousHand.ID)
				if err != nil {
					return nil, err
				}

				for seat, stack := range stacks {
					gs.Players[seat-1].Stack = stack
				}

				// Use the incomplete hand's state
				gs.HandNum = lastHand.HandNum
				gs.Level = lastHand.Level
				gs.ButtonSeat = lastHand.ButtonSeat
			} else {
				// This is the first hand and it's incomplete - just use its values
				gs.HandNum = lastHand.HandNum
				gs.Level = lastHand.Level
				gs.ButtonSeat = lastHand.ButtonSeat
			}

			// Set CurrentHandID to resume this hand
			gs.CurrentHandID = lastHand.ID

			// Load card state from incomplete hand
			gs.HeroCards = FromNullString(lastHand.HeroCards)
			gs.Flop = FromNullString(lastHand.Flop)
			gs.Turn = FromNullString(lastHand.Turn)
			gs.River = FromNullString(lastHand.River)

			// Replay all actions from this hand to rebuild game state
			actions, err := db.GetActionsForHand(lastHand.ID)
			if err != nil {
				return nil, err
			}

			// Start the hand fresh (posts blinds/antes)
			gs.StartNewHand()

			// Replay each action to rebuild state
			for _, action := range actions {
				// Convert stored amount back to raise/bet amount
				var amount int
				if action.ActionType == "raise" {
					// Stored amount is total put in, convert to raise amount above current bet
					toCall := gs.CurrentBet - gs.StreetInvested[action.Seat-1]
					amount = (action.Amount - toCall) + gs.CurrentBet
				} else if action.ActionType == "bet" {
					amount = action.Amount
				} else if action.ActionType == "call" {
					amount = 0 // call doesn't use amount parameter
				} else if action.ActionType == "allin" {
					amount = 0 // allin uses player's stack
				}

				gs.ApplyAction(action.Seat, action.ActionType, amount)

				// Advance to next street if this was the last action of a street
				// (We'll detect this by checking if the next action has a different street)
			}

			// Handle street transitions based on actions
			currentStreet := 0
			if len(actions) > 0 {
				for i, action := range actions {
					if action.Street > currentStreet {
						gs.NextStreet()
						currentStreet = action.Street
					}
					// Last action determines current street
					if i == len(actions)-1 {
						gs.Street = action.Street
					}
				}
			}
		} else {
			// Last hand is complete - load its state and prepare for next hand
			gs.HandNum = lastHand.HandNum
			gs.Level = lastHand.Level
			gs.ButtonSeat = lastHand.ButtonSeat

			// Advance button for next hand
			gs.AdvanceButton()

			stacks, err := db.GetStackSnapshots(lastHand.ID)
			if err != nil {
				return nil, err
			}

			for seat, stack := range stacks {
				gs.Players[seat-1].Stack = stack
			}

			// CurrentHandID stays 0, next hand will be created when user enters ShowHandEntry
		}
	}

	// Mark busted players
	for _, p := range players {
		if p.BustedHandNum > 0 {
			gs.Players[p.Seat-1].IsAlive = false
		}
	}

	return gs, nil
}
