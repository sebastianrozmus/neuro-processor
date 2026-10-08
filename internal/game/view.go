package game

import "github.com/sebastianrozmus/neuro-processor/internal/world"

// PlayerView is what everybody may know about a hero.
type PlayerView struct {
	Name      string
	Avatar    string
	Pos       world.Point
	Abilities Abilities
	Day       int
	Level     int
	XP        int
	// NextXP is the experience that completes the current level.
	NextXP  int
	HP      int
	MaxHP   int
	Mana    int
	MaxMana int
	Gold    int
	Attack  int
	Defense int
	Speed   int
	// AC is the armour class of the D&D mode.
	AC       int
	Weapon   string
	Armor    string
	Charm    string
	Bag      []world.ItemSpec
	Spells   []world.ItemSpec
	Party    []CompanionView
	Fighting bool
	// AutoFight reports that the player's companions act on their own.
	AutoFight bool
	// Poison is how many rounds of venom are left in the hero.
	Poison int
	// Bestiary has a page for every kind of monster of the world.
	Bestiary []BeastView
	// Standing is what the settlements think of the hero.
	Standing []StandingView
}

// CompanionView is a hero for hire following a player, with everything the
// hero's own card shows.
type CompanionView struct {
	Name        string
	Role        string
	Description string
	Avatar      string
	Abilities   Abilities
	Legend      bool
	// Wage is what a legend costs per day; zero for the rest.
	Wage    int
	Level   int
	XP      int
	NextXP  int
	HP      int
	MaxHP   int
	Mana    int
	MaxMana int
	Attack  int
	Defense int
	Speed   int
	AC      int
	Weapon  string
	Armor   string
	Charm   string
	Bag     []world.ItemSpec
	Spells  []world.ItemSpec
	Auto    bool
	Poison  int
}

// MonsterView is a living monster in a fight.
type MonsterView struct {
	ID    int
	Name  string
	Shape world.Shape
	HP    int
	MaxHP int
	Boss  bool
	Trait string
	// AC is the armour class of the D&D mode.
	AC int
}

// MemberView is a hero taking part in a fight.
type MemberView struct {
	Name  string
	HP    int
	MaxHP int
	Acted bool
}

// TurnView is one place in the order of a round.
type TurnView struct {
	Name    string
	Foe     bool
	Current bool
}

// ActorView is the unit whose turn it is, when it belongs to the viewer.
type ActorView struct {
	Name string
	// Companion is the position of the unit in the viewer's party, or -1
	// for the hero.
	Companion int
	Mana      int
	Powers    []world.ItemSpec
	Bag       []world.ItemSpec
}

// FightView is the fight the viewing player is in.
type FightView struct {
	Round    int
	Monsters []MonsterView
	Members  []MemberView
	Order    []TurnView
	// Actor is set when it is the viewer's turn: the unit to decide for.
	Actor *ActorView
	// Waiting names whose turn it is when it is not the viewer's.
	Waiting string
	// Board is the battlefield of the D&D mode.
	Board *BoardView
}

// ShopView is the viewer's ongoing conversation with a shopkeeper.
type ShopView struct {
	// Serial changes whenever a new conversation starts.
	Serial      int
	Building    string
	Kind        world.ShopKind
	Keeper      string
	Greeting    string
	Patience    int
	MaxPatience int
	Wares       []Ware
	Focus       string
	Over        bool
}

// Mark says what is known to stand on a revealed tile.
type Mark struct {
	Monsters int
	Boss     bool
	// Caravan marks the wandering merchant.
	Caravan bool
}

// View is a snapshot of the world as one player sees it. It shares nothing
// mutable with the realm, so an interface can keep and read it freely.
type View struct {
	World *world.World
	// Places are the world's places as the story has left them; the map
	// and the scenes draw these, not the spec's.
	Places   []world.PlaceSpec
	Me       PlayerView
	Others   []PlayerView
	Revealed []bool
	Marks    []Mark
	// Tile and Place describe where the viewer stands.
	Tile   world.Tile
	Place  world.PlaceSpec
	Ground []world.ItemSpec
	Fight  *FightView
	Shop   *ShopView
	Log    []string
	Won    bool
	// Blows are the latest blows struck where the viewer stands.
	Blows []BlowView
	// People are who stands in the viewer's room of an adventure.
	People []string
	// Night is whether it is dark where the viewer is; Day the world's day.
	Night bool
	Day   int
	// Season and Weather are the sky of the day.
	Season  string
	Weather Weather
	// Quests are the viewer's jobs and the ones on offer where they stand.
	Quests []QuestView
}

// PlaceAt returns the place at a point as the story has left it.
func (v View) PlaceAt(pos world.Point) world.PlaceSpec {
	if place := v.World.At(pos).Place; place >= 0 && place < len(v.Places) {
		return v.Places[place]
	}

	return world.PlaceSpec{}
}

// Adventure reports whether the world is a text adventure.
func (v View) Adventure() bool {
	return v.World.Mode == world.ModeAdventure
}

// DnD reports whether the world is played by the tabletop rules.
func (v View) DnD() bool {
	return v.World.Mode == world.ModeDnD
}

// View returns the world as the named player sees it.
func (r *Realm) View(name string) View {
	r.mu.Lock()
	defer r.mu.Unlock()

	view := View{
		World:    r.world,
		Places:   r.seasonal(append([]world.PlaceSpec(nil), r.places...)),
		Revealed: append([]bool(nil), r.revealed...),
		Marks:    make([]Mark, len(r.revealed)),
		Won:      r.won,
		Season:   r.season(),
		Weather:  r.weather,
	}

	if r.caravan != nil && r.revealed[r.world.Index(r.caravan.pos)] {
		view.Marks[r.world.Index(r.caravan.pos)].Caravan = true
	}

	for index, group := range r.monsters {
		if !r.revealed[index] {
			continue
		}

		for _, foe := range group {
			if foe.hp > 0 {
				view.Marks[index].Monsters++
				view.Marks[index].Boss = view.Marks[index].Boss || foe.boss
			}
		}
	}

	viewer := r.players[name]
	if viewer == nil {
		return view
	}

	view.Me = r.playerView(viewer)
	view.Night, view.Day = r.night(viewer), r.day
	view.Tile = r.world.At(viewer.pos)
	view.Place = view.PlaceAt(viewer.pos)
	view.Quests = r.questViews(viewer)
	view.People = personNames(r.peopleAt(viewer.pos))
	view.Ground = append([]world.ItemSpec(nil), r.ground[r.world.Index(viewer.pos)]...)
	view.Log = append([]string(nil), viewer.log...)

	for _, struck := range r.blows {
		if struck.Pos == viewer.pos {
			view.Blows = append(view.Blows, struck)
		}
	}

	for _, other := range r.order {
		if found := r.players[other]; found != viewer && found.connected {
			view.Others = append(view.Others, r.playerView(found))
		}
	}

	if current := r.fightOf(viewer); current != nil {
		view.Fight = r.fightView(current, name)
	}

	if viewer.shop >= 0 && viewer.talks[viewer.shop] != nil {
		view.Shop = r.shopView(viewer)
	}

	return view
}

func (r *Realm) playerView(shown *player) PlayerView {
	return PlayerView{
		Name:      shown.name,
		Avatar:    shown.avatar,
		Pos:       shown.pos,
		Abilities: shown.abilities,
		Day:       shown.day,
		Level:     shown.level,
		XP:        shown.xp,
		NextXP:    shown.nextLevel(),
		HP:        shown.hp,
		MaxHP:     shown.maxHP(),
		Mana:      shown.mana,
		MaxMana:   shown.maxMana(),
		Gold:      shown.gold,
		Attack:    shown.attack(),
		Defense:   shown.defense(),
		Speed:     shown.pace(),
		AC:        armourClass(shown),
		Weapon:    nameOf(shown.weapon),
		Armor:     nameOf(shown.armor),
		Charm:     nameOf(shown.charm),
		Bag:       append([]world.ItemSpec(nil), shown.bag...),
		Spells:    append([]world.ItemSpec(nil), shown.spells...),
		Party:     r.partyView(shown),
		Fighting:  r.fightOf(shown) != nil,
		AutoFight: shown.autoFight,
		Poison:    shown.poison,
		Bestiary:  r.bestiaryPages(shown),
		Standing:  r.standingView(shown),
	}
}

// bestiaryPages is the bestiary, which an adventure has no use for.
func (r *Realm) bestiaryPages(shown *player) []BeastView {
	if r.adventure() {
		return nil
	}

	return r.bestiaryView(shown)
}

func (r *Realm) partyView(leader *player) []CompanionView {
	party := make([]CompanionView, 0, len(leader.party))

	for _, member := range leader.party {
		spec := member.spec()
		party = append(party, CompanionView{
			Name: spec.Name, Role: roleName(spec.Role), Description: spec.Description, Avatar: member.avatar(),
			Abilities: member.abilities, Legend: spec.Legend, Wage: spec.Wage(),
			Level: member.level, XP: member.xp, NextXP: member.nextLevel(),
			HP: member.hp, MaxHP: member.maxHealth(), Mana: member.mana, MaxMana: member.maxEnergy(),
			Attack: member.might(), Defense: member.guard(), Speed: member.pace(), AC: armourClass(member),
			Weapon: nameOf(member.weapon), Armor: nameOf(member.armor), Charm: nameOf(member.charm),
			Bag: append([]world.ItemSpec(nil), member.bag...), Spells: member.powers(), Auto: member.auto, Poison: member.poison,
		})
	}

	return party
}

func nameOf(item *world.ItemSpec) string {
	if item == nil {
		return ""
	}

	return item.Name
}

func (r *Realm) fightView(current *fight, viewer string) *FightView {
	view := &FightView{Round: current.round}

	for _, foe := range r.foes(current.pos) {
		view.Monsters = append(view.Monsters, MonsterView{ID: foe.id, Name: foe.name, Shape: foe.shape, HP: foe.hp, MaxHP: foe.maxHP, Boss: foe.boss, Trait: foe.traitNames(), AC: monsterClass(foe)})
	}

	view.Board = r.boardView(current, viewer)

	now := r.current(current)

	for index, who := range current.queue {
		switch {
		case who.foe != nil && who.foe.hp > 0:
			view.Order = append(view.Order, TurnView{Name: who.foe.name, Foe: true, Current: index == current.turn})
		case who.foe == nil && who.unit.health() > 0 && current.has(who.unit.leader().name):
			view.Order = append(view.Order, TurnView{Name: who.unit.label(), Current: index == current.turn})
		}
	}

	for _, name := range current.members {
		member := r.players[name]
		view.Members = append(view.Members, MemberView{Name: name, HP: member.hp, MaxHP: member.maxHP(), Acted: now != nil && now.unit == unit(member)})

		for _, ally := range member.party {
			view.Members = append(view.Members, MemberView{Name: ally.spec().Name, HP: ally.hp, MaxHP: ally.maxHealth(), Acted: now != nil && now.unit == unit(ally)})
		}
	}

	if now == nil || now.unit == nil {
		return view
	}

	if now.unit.leader().name != viewer {
		view.Waiting = now.unit.label()

		return view
	}

	view.Actor = &ActorView{Name: now.unit.label(), Companion: -1, Mana: now.unit.energy(), Powers: now.unit.powers(), Bag: append([]world.ItemSpec(nil), *now.unit.pack()...)}

	for index, ally := range r.players[viewer].party {
		if unit(ally) == now.unit {
			view.Actor.Companion = index
		}
	}

	return view
}

func (r *Realm) shopView(customer *player) *ShopView {
	shop, current := r.shopSpec(customer.shop), customer.talks[customer.shop]

	return &ShopView{
		Serial:      current.serial,
		Building:    shop.Name,
		Kind:        shop.Kind,
		Keeper:      shop.Keeper,
		Greeting:    shop.Greeting,
		Patience:    current.session.Patience(),
		MaxPatience: current.session.Merchant.Patience,
		Wares:       current.session.Wares(),
		Focus:       current.session.Focus(),
		Over:        current.session.Over,
	}
}
