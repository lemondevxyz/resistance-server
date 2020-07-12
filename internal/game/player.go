package game

import "github.com/toms1441/resistance-server/internal/client"

// Player is a wrapper around client, containing the playertype.
type Player struct {
	client.Client `json:"-"`
	ID            string     `json:"id"`
	Type          PlayerType `json:"type"`
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

func newPlayer(c client.Client) Player {
	return Player{
		Client: c,
		ID:     c.ID,
		Type:   PlayerTypeDefault,
	}
}

// IsValid is a function that returns a boolean value representing the validity of the Player.
func (p Player) IsValid() bool {
	if p.Client.IsValid() {
		if p.Type != PlayerTypeDefault {
			return true
		}
	}

	return false
}
