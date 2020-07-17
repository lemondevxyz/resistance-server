package game

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"github.com/toms1441/resistance-server/internal/conn"
	"github.com/toms1441/resistance-server/internal/logger"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewGame returns a new pointer to Game struct by providing a slice of clients, the type of game and the game options.
func NewGame(clients map[string]conn.Conn, t uint8, o Option) (*Game, error) {

	g := &Game{
		log:    logger.NullLogger(),
		Type:   Type(t),
		Option: o,
	}

	if o.Has(OptionPercival) && t != TypeAvalon.Common() {
		return nil, errors.New("percival must be equipped with type avalon")
	}

	g.Players = map[string]Player{}
	g.playerids = []string{}

	if len(clients) < 5 || len(clients) > 10 {
		return nil, ErrInvalidClients
	}

	for _, v := range clients {
		p := newPlayer(v)
		g.Players[v.GetClient().ID] = p
		g.playerids = append(g.playerids, v.GetClient().ID)
	}

	playerLen := len(g.Players)
	if playerLen > 8 {
		playerLen = 8
	}

	g.Rounds = RoundDefaults[playerLen]

	g.assignRoles()

	return g, nil
}

func (g *Game) SetLogger(log logger.Logger) {
	g.log = log
	log.Info("Logger set successfully.")
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

	amount := len(g.playerids)
	playerIndex := g.playerids

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
	deleteIndex := func(arr []string, i int) []string {
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
		p, ok := g.Players[playerIndex[spyindex]]
		if ok {
			p.Type = PlayerTypeSpy
			// assign that player to be a spy
			g.Players[playerIndex[spyindex]] = p
			// then delete that id from the string array
			playerIndex = deleteIndex(playerIndex, spyindex)
		}
	}

	for _, v := range playerIndex {
		p, ok := g.Players[v]
		if ok {
			p.Type = PlayerTypeResistance
			g.Players[v] = p
		}
	}

	if g.Type == TypeAvalon {
		intn := len(playerIndex)
		if intn <= 0 {
			return
		}

		merlin := rand.Intn(intn)
		p, ok := g.Players[playerIndex[merlin]]
		if ok {
			p.Type = PlayerTypeMerlin

			g.Players[playerIndex[merlin]] = p
			playerIndex = deleteIndex(playerIndex, merlin)
		}
	}

	op := g.Option

	percival := -1
	if op.Has(OptionPercival) {
		intn := len(playerIndex)
		if intn <= 0 {
			return
		}

		percival = rand.Intn(intn)
		p, ok := g.Players[playerIndex[percival]]
		if ok {
			p.Type = PlayerTypePercival

			g.Players[playerIndex[percival]] = p
			playerIndex = deleteIndex(playerIndex, percival)
		}
	}

	if op.Has(OptionMorgana) {
		intn := len(playerIndex)
		if intn <= 0 {
			return
		}

		if percival == -1 {
			percival = rand.Intn(intn)
			p, ok := g.Players[playerIndex[percival]]
			if ok {
				p.Type = PlayerTypePercival

				g.Players[playerIndex[percival]] = p
				playerIndex = deleteIndex(playerIndex, percival)
			}
		}

		intn = len(playerIndex)
		if intn <= 0 {
			return
		}

		morgana := rand.Intn(len(playerIndex))

		p, ok := g.Players[playerIndex[morgana]]
		if ok {
			p.Type = PlayerTypeMorgana

			g.Players[playerIndex[morgana]] = p
			playerIndex = deleteIndex(playerIndex, morgana)
		}
	}
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

	maskGame := Game{}
	maskGame = *g

	maskPlayers := func(p Player) (arr map[string]Player) {
		id := p.GetClient().ID
		arr = map[string]Player{}

		for k, v := range g.Players {

			if !v.IsValid() {
				break
			}

			// only change the type if the current player != player in loop
			if id != k {
				if p.Type == PlayerTypeResistance {
					// if the original player is resistance
					// then mask every player
					v.Type = PlayerTypeResistance
				} else if p.Type == PlayerTypeSpy {
					// if the original player is a spy
					// then mask merlin and percival
					// so show resistance and morgana and fellow spies :)
					if v.Type == PlayerTypeMerlin || v.Type == PlayerTypePercival {
						v.Type = PlayerTypeResistance
					}
				} else if p.Type == PlayerTypeMerlin {
					// if the original player is merlin
					// then mask percival and morgana
					if v.Type == PlayerTypePercival || v.Type == PlayerTypeMorgana {
						v.Type = PlayerTypeResistance
					}
				} else if p.Type == PlayerTypePercival {
					// if the original player is percival
					// then reveal merlin and morgana as possible spies
					// and mask all other players
					if v.Type == PlayerTypeMerlin || v.Type == PlayerTypeMorgana {
						v.Type = PlayerTypeMerlin
					} else if v.Type != PlayerTypeMerlin {
						v.Type = PlayerTypeResistance
					}
				} else if p.Type == PlayerTypeMorgana {
					// if the original player is morgana
					// then hide everybody except spies
					if v.Type != PlayerTypeSpy {
						v.Type = PlayerTypeResistance
					}
				}
			}

			// add the newly modified player to arr
			arr[v.GetClient().ID] = v
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

func (g *Game) SetCaptain() {
	ids := []string{}

	for k := range g.Players {
		ids = append(ids, k)
	}

	g.captain = ids[rand.Intn(len(ids))]
}
