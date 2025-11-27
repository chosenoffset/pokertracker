CREATE TABLE IF NOT EXISTS tournaments
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    buy_in       INTEGER  NOT NULL,
    started_at   DATETIME NOT NULL,
    hero_seat    INTEGER  NOT NULL,
    initial_btn  INTEGER  NOT NULL,
    finish_place INTEGER  DEFAULT 0
);

CREATE TABLE IF NOT EXISTS players
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id   INTEGER NOT NULL REFERENCES tournaments (id),
    seat            INTEGER NOT NULL,
    name            TEXT    NOT NULL,
    starting_stack  INTEGER NOT NULL,
    busted_hand_num INTEGER DEFAULT 0,
    UNIQUE (tournament_id, seat)
);

CREATE TABLE IF NOT EXISTS hands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tournament_id INTEGER NOT NULL REFERENCES tournaments(id),
    hand_num INTEGER NOT NULL,
    level INTEGER NOT NULL,
    button_seat INTEGER NOT NULL,
    hero_cards TEXT,
    flop TEXT,
    turn TEXT,
    river TEXT,
    pot_size INTEGER DEFAULT 0,
    winner_seat INTEGER DEFAULT 0,
    notes TEXT,
    UNIQUE(tournament_id, hand_num)
);

CREATE TABLE IF NOT EXISTS actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hand_id INTEGER NOT NULL REFERENCES hands(id),
    street INTEGER NOT NULL,
    seat INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    amount INTEGER DEFAULT 0,
    sequence INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS stack_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hand_id INTEGER NOT NULL REFERENCES hands(id),
    seat INTEGER NOT NULL,
    stack INTEGER NOT NULL,
    UNIQUE(hand_id, seat)
);

CREATE INDEX IF NOT EXISTS idx_hands_tournament ON hands(tournament_id);
CREATE INDEX IF NOT EXISTS idx_actions_hand ON actions(hand_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_hand ON stack_snapshots(hand_id);