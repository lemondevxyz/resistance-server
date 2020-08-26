package game

import (
	"encoding/json"

	"github.com/toms1441/resistance-server/internal/conn"
)

// Player is a wrapper around client, containing the playertype.
type Player struct {
	conn.Conn
	Type PlayerType `json:"type"`
}

func (p Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Type)
}

const (
	// PlayerTypeDefault is the default PlayerType.
	PlayerTypeDefault PlayerType = iota
	// PlayerTypeResistance is the team representing good.
	PlayerTypeResistance
	// PlayerTypeSpy is the team representing evil.
	// Spies can tell if player is a member of their team.
	PlayerTypeSpy
	// PlayerTypeMerlin is a part of PlayerTypeResistance
	// Merlin can tell if a player is a spy or not.
	PlayerTypeMerlin
	// PlayerTypePercival is a part of PlayerTypeResistance.
	// Percival sees two possible suspects as Merlin, One of which is Morgana and the other is the Real Merlin.
	PlayerTypePercival
	// PlayerTypeMorgana is a part of PlayerTypeSpy, which means Morgana knows who are the spies and who are resistance.
	// Morgana appears as a Merlin to Percival.
	PlayerTypeMorgana
	/*
	*	these will be added later
	*	PLAYER_TYPE_OBERON
	*	PLAYER_TYPE_MORDRED
	*	PLAYER_TYPE_LANCELOT
	*	PLAYER_TYPE_LADY // lady of the lake
	*	PLAYER_TYPE_EXCALIBUR
	*	PLAYER_TYPE_NOREBO
	*	PLAYER_TYPE_PALM
	*	PLAYER_TYPE_QUICKDRAW
	 */
)

// PlayerType is a uint8 representation of the player type.
// Values are between PlayerTypeDefault and PlayerTypeMorgana
type PlayerType uint8

func newPlayer(c conn.Conn) Player {
	return Player{
		Conn: c,
		Type: PlayerTypeDefault,
	}
}

// IsValid is a function that returns a boolean value representing the validity of the Player.
func (p Player) IsValid() bool {
	if p.GetClient().IsValid() {
		if p.Type != PlayerTypeDefault {
			return true
		}
	}

	return false
}
