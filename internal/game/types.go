// Package game holds the rules of the game. Everything that decides an
// outcome (prices, gold, patience) lives here and is deterministic; the
// language model only translates player text into an Intent and words the
// merchant's reply.
package game

import "github.com/sebastianrozmus/neuro-processor/internal/world"

// Item is a single ware offered by a merchant, or brought to one.
type Item struct {
	ID          string
	Name        string
	Description string
	Kind        world.Kind
	// Aliases are lowercase word stems used to spot the item in player text.
	Aliases   []string
	ListPrice int
	// MinPrice is the lowest price the merchant accepts. It never leaves
	// this package: the model is not told what it is.
	MinPrice int
	// Unlimited items stay in stock after a sale.
	Unlimited bool
}

// Merchant is an NPC the player can haggle with.
type Merchant struct {
	Name     string
	Persona  string
	Greeting string
	Items    []Item
	Patience int
	// Currency is the phrase put after an amount, e.g. "szt. złota".
	Currency string
	// Kinds are the kinds of items the merchant deals in and buys back.
	Kinds []world.Kind
}

// Action is what the player is trying to do in a single turn.
type Action string

const (
	ActionOffer  Action = "offer"
	ActionAccept Action = "accept"
	ActionAsk    Action = "ask"
	ActionChat   Action = "chat"
	ActionLeave  Action = "leave"
	// ActionSell is the player offering one of their own things, for the
	// amount they name or for whatever the merchant gives.
	ActionSell Action = "sell"
)

// Actions lists every action the model may choose from.
var Actions = []Action{ActionOffer, ActionAccept, ActionAsk, ActionChat, ActionLeave, ActionSell}

// Intent is the structured reading of one player message.
type Intent struct {
	Action Action `json:"action"`
	ItemID string `json:"item_id"`
	// Amount is the sum offered for the whole lot.
	Amount int `json:"amount"`
	// Quantity is how many pieces the player wants; 0 when they did not say.
	Quantity int `json:"quantity"`
	// Persuasion rates how convincing the message was, from -2 to 2.
	Persuasion int `json:"persuasion"`
}

// Outcome is the result of applying an Intent to the game state.
type Outcome string

const (
	OutcomeSold     Outcome = "sold"
	OutcomeCounter  Outcome = "counter"
	OutcomeInsulted Outcome = "insulted"
	OutcomeNoGold   Outcome = "no_gold"
	// OutcomeShort means the player wants more pieces than there are.
	OutcomeShort      Outcome = "short"
	OutcomeUnclear    Outcome = "unclear"
	OutcomeInfo       Outcome = "info"
	OutcomeWares      Outcome = "wares"
	OutcomeChat       Outcome = "chat"
	OutcomeFarewell   Outcome = "farewell"
	OutcomeWalkedAway Outcome = "walked_away"
	// OutcomeBought means the merchant buys the player's item at Price;
	// OutcomeBid that they name what they would pay for it, after the
	// player asked for too much when Offer is set; OutcomeUnwanted that
	// they do not deal in such things.
	OutcomeBought   Outcome = "bought"
	OutcomeBid      Outcome = "bid"
	OutcomeUnwanted Outcome = "unwanted"
	// OutcomeScene is not a decision of a talk at all: it asks for the
	// narration of a round of combat the rules have already resolved.
	OutcomeScene Outcome = "scene"
)

// Mood describes how the merchant feels after a turn.
type Mood string

const (
	MoodPleased   Mood = "zadowolona"
	MoodCalm      Mood = "spokojna"
	MoodImpatient Mood = "zniecierpliwiona"
	MoodAnnoyed   Mood = "poirytowana"
	MoodOffended  Mood = "obrażona"
)

// Ware is the public view of an item: what the player (and the model) may see.
type Ware struct {
	ID          string
	Name        string
	Description string
	Kind        world.Kind
	Aliases     []string
	Price       int
}

// Decision is everything the rules decided in one turn.
type Decision struct {
	Outcome Outcome
	Item    Ware
	// Quantity is how many pieces the decision is about.
	Quantity int
	// Offer is the amount the player proposed for all of them, or asked
	// for when selling; 0 if none.
	Offer int
	// Price is the sale price for OutcomeSold, otherwise the current ask,
	// in both cases for all the pieces together.
	Price int
	Gold  int
	Mood  Mood
	Wares []Ware
	// Currency is the phrase put after an amount.
	Currency string
	// Scene is what happened in a round of combat, for OutcomeScene.
	Scene string
}

// Line is one entry of the dialogue history.
type Line struct {
	Player bool
	Text   string
}
