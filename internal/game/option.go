package game

const (
	// OptionNone means no options
	OptionNone Option = 0
	// OptionPercival includes PlayerTypePercival in the game
	OptionPercival Option = 1 << iota
	// OptionMorgana includes PlayerTypePercival,PlayerTypeMorgana in the game
	OptionMorgana = (1 << 0) | (1 << 1)
)

// Option represents the Game options, whether to include certain PlayerTypes or not.
// it's stored in the form of bitmasks
type Option uint16

// Add adds an option to (o Option)
func (o Option) Add(option Option) Option {
	return o | option
}

// Remove removes an option from (o Option)
func (o Option) Remove(option Option) Option {
	return o &^ option
}

// Toggle toggles an option in (o Option)
func (o Option) Toggle(option Option) Option {
	return o ^ option
}

// Has returns a boolean indicating if (o Option) contains (option Option)
func (o Option) Has(option Option) bool {
	return o&option != 0
}
