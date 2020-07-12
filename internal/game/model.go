package game

import (
	"encoding/json"
	"errors"
	"math/rand"

	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
)

// Game is the game struct that gets called by lobby.Lobby
// Game.Status gets set when 3 Rounds have been successful or a failure
type Game struct {
	Type    Type     `json:"type"`
	Players []Player `json:"players"`
	Rounds  [5]Round `json:"rounds"`
	Option  Option   `json:"option"`
	Status  Status   `json:"-"`

	captain int
	log     logger.Logger
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

// Type is same as lobby.Type, defining it here prevents an import loop.
type Type uint8

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
	// StatusLost means resistance lost to spides
	StatusLost
	// StatusWon means resistance won to spies
	StatusWon
)

// Status is the result of the game.
type Status uint8

var (
	// ErrInvalidClients occurs when len(clients) < 5 || len(clients) > 10
	ErrInvalidClients = errors.New("game contains less than 5 players or more than 10 players")
)

// NewGame returns a new pointer to Game struct by providing a slice of clients, the type of game and the game options.
func NewGame(log logger.Logger, clients []client.Client, t uint8, o Option) (*Game, error) {

	g := &Game{
		log:    log,
		Type:   Type(t),
		Option: o,
	}

	if len(clients) < 5 || len(clients) > 10 {
		return nil, ErrInvalidClients
	}

	for _, v := range clients {
		p := newPlayer(v)
		g.Players = append(g.Players, p)
	}

	playerLen := len(g.Players)
	if playerLen > 8 {
		playerLen = 8
	}

	g.Rounds = RoundDefaults[playerLen]

	g.assignRoles()

	return g, nil
}

// getStatus returns the status of the game. if it's finished or not.
func (g *Game) getStatus() Status {

	resistance := 0
	spies := 0

	for _, v := range g.Rounds {

		concul := v.GetConculsion()

		if concul == StatusLost {
			// we failed 5 rounds, so the game is over
			if !v.Missions[4].IsAccepted() {
				return StatusLost
			}

			spies++
		} else if concul == StatusWon {
			resistance++
		}
	}

	if spies == 3 {
		return StatusLost
	} else if resistance == 3 {
		return StatusWon
	}

	return StatusDefault
}

// assignRoles assigns the roles for the players.
func (g *Game) assignRoles() {

	amount := 0
	playerIndex := []int{}
	for k := range g.Players {
		amount++
		playerIndex = append(playerIndex, k)
	}

	spies := 0
	if amount >= 5 && amount <= 6 {
		spies = 2
	} else if amount >= 7 && amount <= 9 {
		spies = 3
	} else if amount == 10 {
		spies = 4
	}

	// deleteIndex is a helper function to delete an index from the array.
	// it's crucial in this operation, Example:
	// deleteIndex([]int{1,2,3,4}, 2) => []int{1,2,4}
	deleteIndex := func(arr []int, i int) []int {
		if i > len(arr) || i == -1 {
			g.log.Danger("deleteIndex out of bounds")
			return arr
		}

		arr[len(arr)-1], arr[i] = arr[i], arr[len(arr)-1]
		return arr[:len(arr)-1]
	}

	for i := 0; i < spies; i++ {
		intn := len(playerIndex)

		spyindex := rand.Intn(intn)
		// get the random spy index from the amount of players
		g.Players[playerIndex[spyindex]].Type = PlayerTypeSpy
		// assign that player to be a spy
		playerIndex = deleteIndex(playerIndex, spyindex)
	}

	for _, v := range playerIndex {
		g.Players[v].Type = PlayerTypeResistance
	}

	if g.Type == TypeAvalon {
		intn := len(playerIndex)
		if intn <= 0 {
			return
		}

		merlin := rand.Intn(intn)
		g.Players[playerIndex[merlin]].Type = PlayerTypeMerlin
		playerIndex = deleteIndex(playerIndex, merlin)
	}

	op := g.Option

	percival := -1
	if op.Has(OptionPercival) {
		intn := len(playerIndex)
		if intn <= 0 {
			return
		}

		percival = rand.Intn(intn)
		g.Players[playerIndex[percival]].Type = PlayerTypePercival
		playerIndex = deleteIndex(playerIndex, percival)
	}

	if op.Has(OptionMorgana) {
		intn := len(playerIndex)
		if intn <= 0 {
			return
		}

		if percival == -1 {
			percival = rand.Intn(intn)
			g.Players[playerIndex[percival]].Type = PlayerTypePercival
			playerIndex = deleteIndex(playerIndex, percival)
		}

		intn = len(playerIndex)
		if intn <= 0 {
			return
		}

		morgana := rand.Intn(len(playerIndex))

		g.Players[playerIndex[morgana]].Type = PlayerTypeMorgana
		playerIndex = deleteIndex(playerIndex, morgana)
	}
}

// SetCaptain sets a player as a captain and adds captain Commands.
func (g *Game) SetCaptain(i int) bool {

	if len(g.Players) > i {
		g.captain = i
		return true
	}

	return false
}

// GetCaptain returns the captain as Player
func (g *Game) GetCaptain() Player {
	var c Player

	if len(g.Players) > g.captain {
		return g.Players[g.captain]
	}

	return c
}

// Broadcast is a method for sending a message to all clients.
func (g *Game) Broadcast(ms conn.MessageSend) {
	bytes, err := json.Marshal(ms)
	if err != nil {
		g.log.Warn("json.Marshal: %v", err)
		return
	}

	for _, v := range g.Players {
		// we need to check because most of the slice slots are empty
		if v.IsValid() {
			v.Conn.WriteBytes(bytes)
		}
	}
}

// Run runs the game and sets chan<- Status when the game is done.
func (g *Game) Run(s chan<- Status) {
	defer func(s chan<- Status) {
		s <- g.getStatus()
	}(s)

	g.Send()
	for ri := 0; ri < 5; ri++ {
		if g.runRound(ri) {
			break
		}
	}

}

// Send sends the game information to all the players
func (g *Game) Send() {

	maskGame := &Game{}
	*maskGame = *g

	maskPlayers := func(p Player) (arr []Player) {
		for _, v := range g.Players {

			if !v.IsValid() {
				break
			}

			if p.Type == PlayerTypeResistance {
				// if the original player is resistance
				// then mask every player
				v.Type = PlayerTypeResistance
			} else if p.Type == PlayerTypePercival {
				// if the original player is percival
				// then reveal merlin and morgana as possible spies
				// and mask all other players
				v.Type = PlayerTypeResistance
				if v.Type == PlayerTypeMerlin {
					v.Type = PlayerTypeSpy
				} else if v.Type == PlayerTypeMorgana {
					v.Type = PlayerTypeSpy
				}
			} else if p.Type == PlayerTypeSpy {
				// if the original player is a spy
				// then mask merlin and percival
				if v.Type == PlayerTypeMerlin || v.Type == PlayerTypePercival {
					v.Type = PlayerTypeResistance
				}
			} else if p.Type == PlayerTypeMerlin {
				// if the original player is merlin
				// then mask percival and morgana
				if v.Type == PlayerTypePercival || v.Type == PlayerTypeMorgana {
					v.Type = PlayerTypeResistance
				}
			}

			// add the newly modified player to arr
			arr = append(arr, v)
		}

		return arr
	}

	for _, v := range g.Players {
		if v.IsValid() {
			maskGame.Players = maskPlayers(v)

			v.Conn.WriteMessage(conn.MessageSend{
				Group: "game",
				Name:  "get",
				Body:  maskGame,
			})
		}
	}

}

// GetPlayer returns a player by it's ID
func (g *Game) GetPlayer(id string) Player {

	if i := g.GetPlayerIndex(id); i >= 0 {
		return g.Players[i]
	}

	return Player{}
}

// GetPlayerIndex returns a player's index by it's ID
func (g *Game) GetPlayerIndex(id string) int {
	for k, v := range g.Players {
		if v.ID == id {
			return k
		}
	}

	return -1
}
