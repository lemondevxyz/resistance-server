package game

import (
	"errors"
	"sync"

	"github.com/toms1441/resistance-server/internal/logger"
)

// Game is the game struct that gets called by lobby.Lobby
// Game.Status gets set when 3 Rounds have been successful or a failure
type Game struct {
	Type    Type              `json:"type"`
	Rounds  [5]Round          `json:"rounds"`
	Option  Option            `json:"option"`
	Status  Status            `json:"status"`
	Players map[string]Player `json:"players"`

	// so we have more consistent captains
	// it's basically Players but sorted alphabetically
	playerids []string

	captain string
	log     logger.Logger

	mtx sync.Mutex
}

const (
	// TypeBasic has spies and resistance.
	TypeBasic Type = iota
	// TypeOriginal no documentation yet.
	TypeOriginal
	// TypeAvalon same as TypeBasic only with merlin.
	// Merlin is a character that sees all spies and resistance.
	TypeAvalon
	// TypeHunter no documentation
	TypeHunter
	// TypeTrumpmode no documentation
	TypeTrumpmode
)

var typeStrings = map[Type]string{
	TypeBasic:     "Basic",
	TypeOriginal:  "Original",
	TypeAvalon:    "Avalon",
	TypeHunter:    "Hunter",
	TypeTrumpmode: "Trumpmode",
}

// Type is same as lobby.Type, defining it here prevents an import loop.
type Type uint8

// Common returns a uint8 representation of the type
func (t Type) Common() uint8 {
	return uint8(t)
}

func (t Type) String() string {
	val, ok := typeStrings[t]
	if !ok {
		return ""
	}

	return val
}

// Round is a round in the game, there are 5 rounds per game.
type Round struct {
	// missions that are in the round, 5 per each round
	Missions [5]Mission `json:"missions"`
	// the exact amount of assignees that are to be selected in the round
	Assignees uint8 `json:"assignees"`
	// the minimum amount of failure required in the round
	MinFailure uint8 `json:"minfailure"`
	// the amount of players that wanted to fail the mission
	Failure uint8 `json:"failure"`
	// players that voted failure
	// by id
	failure []string
	// players that voted success
	// by id
	success []string
}

// Mission is a struct that's used in the Rounds. For every round there are 5 Missions maximum, if all missions(5) are a failure the game is lost.
// Mission gets set whenever a captain picks X amount of players. Once they are picked, an event gets sent to every player whether they wanna accept the mission(i.e proceed with the mission) or decline the mission.
type Mission struct {
	// players that accepted the mission
	Accept []string `json:"accept"`
	// players that declined the mission
	Decline []string `json:"decline"`
	// players that were in the mission
	Assignees []string `json:"assignees"`
	// all of the above is a slice of player ids
}

const (
	// StatusDefault is the default status
	StatusDefault Status = iota
	// StatusLost means resistance lost to spies
	StatusLost
	// StatusWon means resistance won to spies
	StatusWon
)

// Status is the result of the game.
type Status uint8

var statusString = map[Status]string{
	StatusDefault: "Default",
	StatusLost:    "Spies win",
	StatusWon:     "Resistance win",
}

func (s Status) String() string {
	val, ok := statusString[s]
	if !ok {
		return ""
	}

	return val
}

var (
	// ErrInvalidClients occurs when len(clients) < 5 || len(clients) > 10
	ErrInvalidClients = errors.New("game contains less than 5 players or more than 10 players")
)
