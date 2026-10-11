package game

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The inspector is a window into the whole realm for whoever runs it: the
// map with everyone on it, the knowledge graph with who knows what, the
// bestiary, the items, the keepers' jobs. It reads everything and may put
// right what a model wrote badly: a fact, a description, a persona.

// Inspection is the realm as the inspector shows it.
type Inspection struct {
	Name     string  `json:"name"`
	Mode     string  `json:"mode"`
	Intro    string  `json:"intro"`
	Currency string  `json:"currency"`
	Style    string  `json:"style,omitempty"`
	Day      int     `json:"day"`
	Season   string  `json:"season"`
	Weather  Weather `json:"weather"`
	Won      bool    `json:"won"`
	Lost     bool    `json:"lost"`

	Width  int           `json:"width"`
	Height int           `json:"height"`
	Tiles  []TileInfo    `json:"tiles"`
	Places []PlaceInfo   `json:"places"`
	Towns  []world.Point `json:"towns"`
	Lair   world.Point   `json:"lair"`

	Players    []PlayerInfo `json:"players"`
	Keepers    []KeeperInfo `json:"keepers"`
	Facts      []FactInfo   `json:"facts"`
	Beasts     []BeastInfo  `json:"beasts"`
	Items      []ItemInfo   `json:"items"`
	Companions []MateInfo   `json:"companions"`
	Quests     []QuestInfo  `json:"quests"`

	Caravan *world.Point   `json:"caravan,omitempty"`
	Heir    string         `json:"heir,omitempty"`
	Markup  map[string]int `json:"markup,omitempty"`
	Flags   []string       `json:"flags,omitempty"`
	// Rewrites says whether a model is there to reword descriptions.
	Rewrites bool `json:"rewrites"`
	// Rejected are the latest lines of the model that the rules refused,
	// newest last.
	Rejected []RejectionInfo `json:"rejected"`
}

// RejectionInfo is one refused reply: who asked whom what, and what the
// model offered instead of the canned line that was spoken.
type RejectionInfo struct {
	Day      int       `json:"day"`
	Player   string    `json:"player"`
	Keeper   string    `json:"keeper"`
	Said     string    `json:"said"`
	Hint     string    `json:"hint"`
	Attempts []Attempt `json:"attempts"`
}

// rejectionsKept is how many refused replies the realm remembers.
const rejectionsKept = 40

// Reject remembers a reply the rules refused, for the inspector.
func (r *Realm) Reject(name string, request NarrateRequest, rejection *Rejection) {
	if rejection == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.rejections = append(r.rejections, RejectionInfo{
		Day: r.day, Player: name, Keeper: request.Merchant.Name, Said: request.Text, Hint: request.Hint, Attempts: rejection.Attempts,
	})

	if len(r.rejections) > rejectionsKept {
		r.rejections = r.rejections[len(r.rejections)-rejectionsKept:]
	}
}

// TileInfo is one square of the map with everything on it.
type TileInfo struct {
	Place    int      `json:"place"`
	Town     int      `json:"town"`
	Site     int      `json:"site"`
	Revealed bool     `json:"revealed"`
	Monsters []string `json:"monsters,omitempty"`
	Boss     bool     `json:"boss,omitempty"`
	Ground   []string `json:"ground,omitempty"`
	Caravan  bool     `json:"caravan,omitempty"`
	Players  []string `json:"players,omitempty"`
}

// PlaceInfo is a place as the story has left it.
type PlaceInfo struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Terrain     string `json:"terrain"`
	Danger      int    `json:"danger"`
}

// PlayerInfo is a hero with what the inspector may touch.
type PlayerInfo struct {
	Name      string         `json:"name"`
	Avatar    string         `json:"avatar"`
	Connected bool           `json:"connected"`
	Pos       world.Point    `json:"pos"`
	Level     int            `json:"level"`
	XP        int            `json:"xp"`
	HP        int            `json:"hp"`
	MaxHP     int            `json:"max_hp"`
	Mana      int            `json:"mana"`
	MaxMana   int            `json:"max_mana"`
	Gold      int            `json:"gold"`
	Attack    int            `json:"attack"`
	Defense   int            `json:"defense"`
	Weapon    string         `json:"weapon"`
	Armor     string         `json:"armor"`
	Charm     string         `json:"charm"`
	Bag       []string       `json:"bag"`
	Spells    []string       `json:"spells"`
	Party     []string       `json:"party"`
	Talking   string         `json:"talking,omitempty"`
	Hour      string         `json:"hour"`
	Standing  []StandingView `json:"standing,omitempty"`
	Log       []string       `json:"log"`
}

// KeeperInfo is somebody who can be talked to and knows things.
type KeeperInfo struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Building string `json:"building"`
	Kind     string `json:"kind"`
	Persona  string `json:"persona"`
	Greeting string `json:"greeting,omitempty"`
	Knows    int    `json:"knows"`
}

// FactInfo is a fact with everyone attached to it.
type FactInfo struct {
	ID      int               `json:"id"`
	Subject string            `json:"subject"`
	Text    string            `json:"text"`
	Source  string            `json:"source"`
	Day     int               `json:"day"`
	Known   []string          `json:"known"`
	Heard   []string          `json:"heard"`
	Pointed map[string]string `json:"pointed,omitempty"`
}

// BeastInfo is a kind of monster with how many of it there are.
type BeastInfo struct {
	Kind        int    `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Shape       string `json:"shape"`
	Tier        int    `json:"tier"`
	Boss        bool   `json:"boss"`
	HP          int    `json:"hp"`
	Attack      int    `json:"attack"`
	Trait       Trait  `json:"trait"`
	Alive       int    `json:"alive"`
	Slain       int    `json:"slain"`
}

// ItemInfo is a kind of item.
type ItemInfo struct {
	Index       int    `json:"index"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	Effect      string `json:"effect"`
	Tier        int    `json:"tier"`
	Price       int    `json:"price"`
	Power       string `json:"power"`
}

// MateInfo is a hero for hire.
type MateInfo struct {
	Kind        int    `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        string `json:"role"`
	Tier        int    `json:"tier"`
	Legend      bool   `json:"legend"`
	Inn         string `json:"inn"`
	HiredBy     string `json:"hired_by,omitempty"`
}

// QuestInfo is a keeper's job.
type QuestInfo struct {
	Keeper   string `json:"keeper"`
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	Text     string `json:"text"`
	Reward   int    `json:"reward"`
	Taker    string `json:"taker,omitempty"`
	Finished bool   `json:"finished"`
	Paid     bool   `json:"paid"`
	Day      int    `json:"day"`
}

// Inspect takes stock of the whole realm.
func (r *Realm) Inspect() Inspection {
	r.mu.Lock()
	defer r.mu.Unlock()

	built := r.world
	report := Inspection{
		Name: built.Name, Mode: string(built.Mode), Intro: built.Intro, Currency: built.Currency,
		Day: r.day, Season: r.season(), Weather: r.weather, Won: r.won, Lost: r.lost,
		Width: world.Width, Height: world.Height, Towns: built.Towns, Lair: built.Lair,
		Markup: r.markup,
	}

	if style := built.Style; !style.Empty() {
		report.Style = style.Look
	}

	for index, place := range r.places {
		report.Places = append(report.Places, PlaceInfo{Index: index, Name: place.Name, Description: place.Description, Terrain: string(place.Terrain), Danger: place.Danger})
	}

	if !built.Mode.Rooms() {
		report.Tiles = make([]TileInfo, len(built.Tiles))
		for index, tile := range built.Tiles {
			info := TileInfo{Place: tile.Place, Town: tile.Town, Site: tile.Site, Revealed: index < len(r.revealed) && r.revealed[index]}
			if index < len(r.monsters) {
				for _, beast := range r.monsters[index] {
					info.Monsters = append(info.Monsters, beast.name)
					info.Boss = info.Boss || beast.boss
				}
			}

			if index < len(r.ground) {
				for _, item := range r.ground[index] {
					info.Ground = append(info.Ground, item.Name)
				}
			}

			report.Tiles[index] = info
		}

		if r.caravan != nil {
			pos := r.caravan.pos
			report.Caravan = &pos
			report.Tiles[pos.Y*world.Width+pos.X].Caravan = true
		}
	}

	for _, name := range r.order {
		hero := r.players[name]
		view := r.playerView(hero)
		info := PlayerInfo{
			Name: name, Avatar: hero.avatar, Connected: hero.connected, Pos: hero.pos, Level: view.Level, XP: view.XP,
			HP: view.HP, MaxHP: view.MaxHP, Mana: view.Mana, MaxMana: view.MaxMana, Gold: view.Gold, Attack: view.Attack, Defense: view.Defense,
			Weapon: view.Weapon, Armor: view.Armor, Charm: view.Charm, Hour: r.hourName(hero), Standing: view.Standing,
			Bag: names(view.Bag), Spells: names(view.Spells), Log: append([]string(nil), hero.log...),
		}

		for _, member := range view.Party {
			info.Party = append(info.Party, member.Name)
		}

		if hero.shop >= 0 {
			info.Talking = r.keeperName(hero.shop)
		}

		report.Players = append(report.Players, info)

		if len(report.Tiles) > 0 {
			tile := hero.pos.Y*world.Width + hero.pos.X
			if tile >= 0 && tile < len(report.Tiles) {
				report.Tiles[tile].Players = append(report.Tiles[tile].Players, name)
			}
		}
	}

	for _, index := range r.everyone() {
		info := KeeperInfo{Index: index, Name: r.keeperName(index), Knows: len(r.knownBy(index))}
		switch {
		case r.investigation():
			person := built.People[index]
			info.Building, info.Persona = "świadek", person.Description
		case r.caravan != nil && index == r.caravanShop():
			info.Building, info.Kind, info.Persona = "karawana", string(world.ShopCaravan), caravanLine()
		default:
			shop := built.Shops[index]
			info.Building, info.Kind, info.Persona, info.Greeting = shop.Name, string(shop.Kind), shop.Persona, shop.Greeting
		}

		report.Keepers = append(report.Keepers, info)
	}

	for _, fact := range r.wisdom.facts {
		report.Facts = append(report.Facts, r.factInfo(fact))
	}

	alive := map[string]int{}
	for _, tile := range r.monsters {
		for _, beast := range tile {
			alive[beast.name]++
		}
	}

	var shown *player
	if len(r.order) > 0 {
		shown = r.players[r.order[0]]
	}

	if shown != nil {
		for _, page := range r.bestiaryView(shown) {
			report.Beasts = append(report.Beasts, BeastInfo{
				Kind: page.Kind, Name: page.Name, Description: page.Description, Shape: string(page.Shape), Tier: page.Tier, Boss: page.Boss,
				HP: page.HP, Attack: page.Attack, Trait: page.Trait, Alive: alive[page.Name], Slain: r.slain[page.Name],
			})
		}
	}

	for index, item := range built.Items {
		report.Items = append(report.Items, ItemInfo{
			Index: index, ID: item.ID, Name: item.Name, Description: item.Description, Kind: string(item.Kind), Effect: string(item.Effect),
			Tier: item.Tier, Price: item.Price(), Power: Effect(item),
		})
	}

	for kind, mate := range built.Companions {
		info := MateInfo{Kind: kind, Name: mate.Name, Description: mate.Description, Role: roleName(mate.Role), Tier: mate.Tier, Legend: mate.Legend, HiredBy: r.hired[kind]}
		if len(built.Towns) > 0 && !built.Mode.Rooms() {
			info.Inn = r.placeAt(built.Towns[r.innOf(kind)]).Name
		}

		report.Companions = append(report.Companions, info)
	}

	for _, job := range r.quests {
		report.Quests = append(report.Quests, QuestInfo{
			Keeper: r.keeperName(job.keeper), Kind: string(job.kind), Subject: job.subject, Text: r.questText(job),
			Reward: job.reward, Taker: job.taker, Finished: job.finished, Paid: job.paid, Day: job.day,
		})
	}

	if r.heir != nil && r.heir.kind >= 0 && r.heir.kind < len(built.Monsters) {
		report.Heir = built.Monsters[r.heir.kind].Name
	}

	for flag, set := range r.story.flags {
		if set {
			report.Flags = append(report.Flags, flag)
		}
	}

	sort.Strings(report.Flags)
	report.Rejected = append([]RejectionInfo{}, r.rejections...)

	return report
}

// factInfo words a fact with its people.
func (r *Realm) factInfo(fact *Fact) FactInfo {
	info := FactInfo{ID: fact.ID, Subject: fact.Subject, Text: fact.Text, Source: fact.Source, Day: fact.Day, Known: []string{}, Heard: []string{}}

	for keeper := range fact.known {
		if name := r.keeperName(keeper); name != "" {
			info.Known = append(info.Known, name)
		}
	}

	for name := range fact.heard {
		info.Heard = append(info.Heard, name)
	}

	if len(fact.pointed) > 0 {
		info.Pointed = fact.pointed
	}

	sort.Strings(info.Known)
	sort.Strings(info.Heard)

	return info
}

// SetFact changes a fact, or adds one when id is 0: its subject, text and
// who knows it, by name. It returns the fact as it stands.
func (r *Realm) SetFact(id int, subject, text string, known []string) (FactInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	subject, text = strings.TrimSpace(subject), strings.TrimSpace(text)
	if subject == "" || text == "" {
		return FactInfo{}, false
	}

	var fact *Fact
	if id == 0 {
		fact = r.add(subject, text, "podgląd")
	} else {
		fact = r.fact(id)
	}

	if fact == nil {
		return FactInfo{}, false
	}

	fact.Subject, fact.Text = subject, text
	fact.known = map[int]bool{}

	for _, name := range known {
		if keeper := r.keeperByName(name); keeper >= 0 {
			fact.known[keeper] = true
		}
	}

	return r.factInfo(fact), true
}

// DropFact forgets a fact everywhere.
func (r *Realm) DropFact(id int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	for index, fact := range r.wisdom.facts {
		if fact.ID == id {
			r.wisdom.facts = append(r.wisdom.facts[:index], r.wisdom.facts[index+1:]...)

			return true
		}
	}

	return false
}

// fact finds a fact by ID.
func (r *Realm) fact(id int) *Fact {
	for _, fact := range r.wisdom.facts {
		if fact.ID == id {
			return fact
		}
	}

	return nil
}

// keeperByName finds a keeper, or -1.
func (r *Realm) keeperByName(name string) int {
	candidates := r.everyone()
	for kind := range r.world.Companions {
		candidates = append(candidates, len(r.world.Shops)+1+kind)
	}

	for _, index := range candidates {
		if index >= 0 && strings.EqualFold(r.keeperName(index), strings.TrimSpace(name)) {
			return index
		}
	}

	return -1
}

// Described is what the inspector may reword: the kinds of thing and the
// text each has.
const (
	DescribePlace     = "place"
	DescribeMonster   = "monster"
	DescribeBoss      = "boss"
	DescribeItem      = "item"
	DescribeCompanion = "companion"
	DescribeKeeper    = "keeper"
	DescribeFact      = "fact"
)

// Description returns the name and text of a thing the inspector may
// reword, by kind and index (a fact's ID for a fact).
func (r *Realm) Description(kind string, index int) (string, string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	built := r.world

	switch kind {
	case DescribePlace:
		if index >= 0 && index < len(r.places) {
			return r.places[index].Name, r.places[index].Description, true
		}
	case DescribeMonster:
		if index >= 0 && index < len(built.Monsters) {
			return built.Monsters[index].Name, built.Monsters[index].Description, true
		}
	case DescribeBoss:
		return built.Boss.Name, built.Boss.Description, true
	case DescribeItem:
		if index >= 0 && index < len(built.Items) {
			return built.Items[index].Name, built.Items[index].Description, true
		}
	case DescribeCompanion:
		if index >= 0 && index < len(built.Companions) {
			return built.Companions[index].Name, built.Companions[index].Description, true
		}
	case DescribeKeeper:
		if index >= 0 && index < len(built.Shops) {
			return built.Shops[index].Keeper, built.Shops[index].Persona, true
		}
	case DescribeFact:
		if fact := r.fact(index); fact != nil {
			return fact.Subject, fact.Text, true
		}
	}

	return "", "", false
}

// Describe replaces the text of a thing the inspector may reword.
func (r *Realm) Describe(kind string, index int, text string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	built := r.world

	switch kind {
	case DescribePlace:
		if index >= 0 && index < len(r.places) {
			r.places[index].Description = text
			if index < len(built.Places) {
				built.Places[index].Description = text
			}

			return true
		}
	case DescribeMonster:
		if index >= 0 && index < len(built.Monsters) {
			built.Monsters[index].Description = text

			return true
		}
	case DescribeBoss:
		built.Boss.Description = text

		return true
	case DescribeItem:
		if index >= 0 && index < len(built.Items) {
			built.Items[index].Description = text

			return true
		}
	case DescribeCompanion:
		if index >= 0 && index < len(built.Companions) {
			built.Companions[index].Description = text

			return true
		}
	case DescribeKeeper:
		if index >= 0 && index < len(built.Shops) {
			built.Shops[index].Persona = text

			return true
		}
	case DescribeFact:
		if fact := r.fact(index); fact != nil {
			fact.Text = text

			return true
		}
	}

	return false
}

// Adjust sets a hero's gold, health or mana from the inspector.
func (r *Realm) Adjust(name, field string, value int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	hero := r.players[name]
	if hero == nil || value < 0 {
		return false
	}

	switch field {
	case "gold":
		hero.gold = value
	case "hp":
		hero.hp = min(value, r.playerView(hero).MaxHP)
	case "mana":
		hero.mana = min(value, r.playerView(hero).MaxMana)
	default:
		return false
	}

	r.tell(hero, "Podgląd zmienia: "+field+" = "+strconv.Itoa(value)+".")

	return true
}

// RewriteRequest asks a model to word a description of the world again.
type RewriteRequest struct {
	// Kind is one of the Describe constants; Name what the text is about.
	Kind string
	Name string
	Text string
	// World and Style set the tone: the world's intro and its look.
	World string
	Style string
}

// Rewriter is a model able to reword a description.
type Rewriter interface {
	Rewrite(ctx context.Context, request RewriteRequest) (string, error)
}

// Tone returns what a rewrite should follow: the world's intro and style.
func (r *Realm) Tone() (string, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	style := ""
	if !r.world.Style.Empty() {
		style = r.world.Style.Look
	}

	return r.world.Intro, style
}

// names lists items by name.
func names(items []world.ItemSpec) []string {
	listed := make([]string, 0, len(items))
	for _, item := range items {
		listed = append(listed, item.Name)
	}

	return listed
}
