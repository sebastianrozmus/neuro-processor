package game

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

const (
	// BaseHP and BaseMana are the health and power of a hero without charms.
	BaseHP   = 40
	BaseMana = 12
	// DefaultRoundTimeout is how long a fight waits for slow players.
	DefaultRoundTimeout = 20 * time.Second

	// MaxLevel is the highest level a hero can reach.
	MaxLevel = 10

	baseAttack   = 3
	startGold    = 60
	hpPerLevel   = 6
	manaPerLevel = 2
	xpPerLevel   = 40
	bossXP       = 300
	manaPerStep  = 1
	// A day of travel is this many steps; wages fall due at its end.
	stepsPerDay     = 12
	fleePercent     = 60
	lootPercent     = 40
	fallLossPercent = 20
	revealRadius    = 1
	logLimit        = 80
	nameLimit       = 16
	defaultName     = "Wędrowiec"
)

var directionNames = map[world.Point]string{north: "północ", south: "południe", east: "wschód", west: "zachód"}

// Realm is one running world shared by every player in it. All game state
// lives here and changes only through its methods, so the rules are the same
// whether one person plays or several.
type Realm struct {
	mu    sync.Mutex
	world *world.World
	// places is the realm's own copy of the world's places: a story may
	// change them, and the screens draw these.
	places []world.PlaceSpec
	story  adventure
	// lore is what the world knows and who knows it; day the world's day,
	// markup the drift of prices and slain how many of each kind of
	// monster have fallen.
	wisdom       knowledge
	day          int
	markup       map[string]int
	slain        map[string]int
	rng          *rand.Rand
	roundTimeout time.Duration
	players      map[string]*player
	order        []string
	monsters     [][]*monster
	ground       [][]world.ItemSpec
	revealed     []bool
	fights       map[world.Point]*fight
	extras       []extra
	// hired maps a companion to the player they follow.
	hired map[int]string
	talks int
	won   bool
	lost  bool
	// blows are the latest things that happened to anyone's health, for
	// the screens to show; serial numbers them. minted counts monsters.
	blows  []BlowView
	serial int
	minted int
	// weather is the sky of the day; caravan the merchant on the road;
	// quests the jobs keepers posted; heir the threat growing after the
	// boss fell.
	weather Weather
	caravan *caravan
	quests  []*quest
	heir    *heir
	// rejections are the model's latest lines the rules refused.
	rejections []RejectionInfo

	// master narrates rounds of combat when the world has one.
	master Narrator
	record func(NarrateRequest, string)
}

type player struct {
	world     *world.World
	name      string
	avatar    string
	connected bool
	pos       world.Point
	prev      world.Point
	// home is the settlement the hero wakes up in after falling.
	home      int
	abilities Abilities
	// day counts the hero's days in the world; steps the hours of the current.
	day   int
	steps int
	// level starts at 1; xp counts towards the next one.
	level  int
	xp     int
	hp     int
	mana   int
	gold   int
	bag    []world.ItemSpec
	weapon *world.ItemSpec
	armor  *world.ItemSpec
	charm  *world.ItemSpec
	spells []world.ItemSpec
	log    []string
	talks  map[int]*talk
	shop   int
	party  []*companion
	// autoFight lets every companion of the player act on their own.
	autoFight bool
	// poison is how many more rounds venom works in the hero.
	poison int
	// bestiary maps the kinds of monster the hero has met to how many of
	// each fell in their fights.
	bestiary map[int]int
	// standing is what each settlement thinks of the hero; inn whether
	// the current talk is at the inn after hours; sky the last weather
	// the hero remarked on.
	standing map[int]int
	inn      bool
	sky      Weather
	// following names the person of a town the hero shadows.
	following string
	updates   chan struct{}
}

// talk is a hero's conversation with one shopkeeper.
type talk struct {
	session *Session
	serial  int
	// lead is the fact the keeper last sent the player elsewhere about,
	// so that "a co słyszałeś?" stays on it; leadKeeper who knows it.
	lead       *Fact
	leadKeeper int
}

type monster struct {
	// id tells monsters of one kind apart; kind indexes the world's
	// monsters, the boss having -1.
	id     int
	kind   int
	name   string
	shape  world.Shape
	tier   int
	boss   bool
	hp     int
	maxHP  int
	attack int
}

// NewRealm starts a world. The seed drives combat rolls and loot.
func NewRealm(built *world.World, seed int64) *Realm {
	realm := &Realm{
		world:        built,
		places:       append([]world.PlaceSpec(nil), built.Places...),
		story:        adventure{flags: map[string]bool{}, done: map[int]bool{}},
		day:          1,
		markup:       map[string]int{},
		slain:        map[string]int{},
		rng:          rand.New(rand.NewSource(seed)),
		roundTimeout: DefaultRoundTimeout,
		players:      map[string]*player{},
		monsters:     make([][]*monster, len(built.Tiles)),
		ground:       make([][]world.ItemSpec, len(built.Tiles)),
		revealed:     make([]bool, len(built.Tiles)),
		fights:       map[world.Point]*fight{},
		hired:        map[int]string{},
	}

	for index, tile := range built.Tiles {
		for _, kind := range tile.Monsters {
			realm.monsters[index] = append(realm.monsters[index], realm.newMonster(kind))
		}

		if tile.Boss {
			realm.monsters[index] = append(realm.monsters[index], realm.newMonster(-1))
		}

		for _, item := range tile.Loot {
			realm.ground[index] = append(realm.ground[index], built.Items[item])
		}
	}

	realm.weather = WeatherClear
	realm.newCaravan()
	realm.seedKnowledge()

	return realm
}

func (r *Realm) newMonster(kind int) *monster {
	r.minted++

	if kind < 0 {
		stats := world.StatsFor(0, true)
		boss := r.world.Boss

		return &monster{id: r.minted, kind: -1, name: boss.Name, shape: boss.Shape, tier: world.MaxTier, boss: true, hp: stats.HP, maxHP: stats.HP, attack: stats.Attack}
	}

	spec := r.world.Monsters[kind]
	stats := world.StatsFor(spec.Tier, false)

	return &monster{id: r.minted, kind: kind, name: spec.Name, shape: spec.Shape, tier: spec.Tier, hp: stats.HP, maxHP: stats.HP, attack: stats.Attack}
}

// World returns the world the realm runs. It never changes.
func (r *Realm) World() *world.World {
	return r.world
}

// SetRoundTimeout changes how long a fight waits for slow players.
func (r *Realm) SetRoundTimeout(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.roundTimeout = timeout
}

// Join adds a player to the world and returns the name they play under.
// Joining under the name of a player who left resumes that character.
func (r *Realm) Join(name, avatar string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	name = cleanName(name)

	if existing, ok := r.players[name]; ok && !existing.connected {
		existing.connected = true
		r.tell(existing, "Wracasz do gry.")
		r.tellOthers(existing, existing.name+" wraca do gry.")

		if r.alive(existing.pos) > 0 {
			r.engage(existing)
		}

		return name
	}

	base := name
	for suffix := 2; r.players[name] != nil; suffix++ {
		name = fmt.Sprintf("%s %d", base, suffix)
	}

	joined := &player{
		world:     r.world,
		name:      name,
		avatar:    avatar,
		connected: true,
		pos:       r.world.Start(),
		prev:      r.world.Start(),
		abilities: heroAbilities(r.rng, avatar),
		day:       1,
		level:     1,
		gold:      startGold,
		shop:      -1,
		updates:   make(chan struct{}, 1),
	}
	joined.hp, joined.mana = joined.maxHP(), joined.maxMana()

	r.players[name] = joined
	r.order = append(r.order, name)
	r.reveal(joined.pos)
	r.tell(joined, r.world.Intro)

	if r.adventure() {
		r.tell(joined, "Cel: "+r.world.Goal+" Wpisz «pomoc», żeby zobaczyć polecenia.")
	} else {
		r.tell(joined, fmt.Sprintf("Cel: odnaleźć i pokonać przeciwnika, który nazywa się %s. Wpisz «pomoc», żeby zobaczyć polecenia.", r.world.Boss.Name))
	}

	r.arrive(joined, -1)
	r.tellOthers(joined, name+" dołącza do świata.")

	return name
}

// Leave disconnects a player. Their character stays in the world.
func (r *Realm) Leave(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	leaving := r.active(name)
	if leaving == nil {
		return
	}

	leaving.connected = false
	leaving.shop, leaving.talks = -1, nil
	r.tellOthers(leaving, name+" opuszcza świat.")

	if current := r.fightOf(leaving); current != nil {
		current.stopTimer()
		r.leaveFight(current, leaving)
		r.proceed(current)
	}
}

// Updates returns a channel that receives a value whenever the world changes.
func (r *Realm) Updates(name string) <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	if found := r.players[name]; found != nil {
		return found.updates
	}

	return nil
}

// Announce writes a message into every player's log.
func (r *Realm) Announce(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	for _, name := range r.order {
		r.tell(r.players[name], text)
	}
}

// Tell writes a message into one player's log.
func (r *Realm) Tell(name, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if listener := r.players[name]; listener != nil {
		r.tell(listener, text)
	}
}

// Move walks a player one step. Stepping onto monsters starts a fight or
// joins the one already going on there.
func (r *Realm) Move(name string, dx, dy int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer r.notify()

	if mover := r.active(name); mover != nil {
		r.walk(mover, world.Point{X: dx, Y: dy})
	}
}

func (r *Realm) walk(mover *player, step world.Point) {
	target := world.Point{X: mover.pos.X + step.X, Y: mover.pos.Y + step.Y}

	switch {
	case directionNames[step] == "":
		r.tell(mover, "Dokąd? Podaj kierunek: północ, południe, wschód albo zachód.")
	case r.fightOf(mover) != nil && r.dnd():
		if current, who := r.turn(mover); current != nil {
			r.step(current, who, step)
		}
	case r.fightOf(mover) != nil:
		r.tell(mover, "Najpierw walka albo ucieczka.")
	case !r.world.In(target):
		r.tell(mover, "Dalej nie ma przejścia.")
	case r.adventure() && !r.passage(mover, step):
	default:
		before := r.world.At(mover.pos).Place
		mover.prev = mover.pos
		if r.weather != WeatherHeat {
			mover.mana = min(mover.maxMana(), mover.mana+manaPerStep)
		}

		r.tell(mover, "Idziesz na "+directionNames[step]+".")
		r.place(mover, target, before)
		r.hour(mover)

		if r.slow() {
			r.hour(mover)
		}
	}
}

// hour passes a step's worth of time for a hero; a full day of steps ends
// the day.
func (r *Realm) hour(hero *player) {
	before := r.hourName(hero)

	hero.steps++
	if hero.steps >= stepsPerDay {
		r.newDay(hero)
	}

	if r.investigation() {
		r.passHour(hero, before)
	} else {
		r.deadline(hero)
	}
}

// newDay turns the hero's day over and settles the wages of their legends.
func (r *Realm) newDay(hero *player) {
	hero.day++
	hero.steps = 0
	r.worldDay(hero.day)
	r.tell(hero, fmt.Sprintf("Zaczyna się dzień %d (%s). Pogoda: %s.", hero.day, r.season(), r.weather))
	r.payWages(hero)
	r.fire(hero, onDay, "")
}

// place puts a player on a tile and deals with what waits there. The place
// they came from decides how much of the new one is described.
func (r *Realm) place(mover *player, target world.Point, before int) {
	mover.pos = target
	mover.shop, mover.talks, mover.inn = -1, nil, false
	r.reveal(target)
	r.arrive(mover, before)
}

// arrive tells a player what they find where they stand.
func (r *Realm) arrive(visitor *player, before int) {
	tile := r.world.At(visitor.pos)
	place := r.placeAt(visitor.pos)

	if tile.Place == before {
		r.tell(visitor, place.Name+".")
	} else {
		r.tell(visitor, place.Name+". "+place.Description)
	}

	if r.adventure() {
		r.arriveRoom(visitor)

		return
	}

	if tile.Town >= 0 {
		visitor.home = tile.Town
		r.tell(visitor, "Możesz tu odwiedzić: "+join(r.buildings())+". Odpoczynek jest za darmo.")

		if len(r.world.Companions) > 0 {
			r.tell(visitor, r.innLine(tile.Town))
		}
	}

	if r.night(visitor) && tile.Place != before {
		r.tell(visitor, "Zapada noc.")
	}

	if note := r.weatherNote(); note != "" && visitor.sky != r.weather && tile.Town < 0 {
		visitor.sky = r.weather
		r.tell(visitor, note)
	}

	if r.caravanHere(visitor.pos) {
		r.tell(visitor, caravanLine())
	}

	r.reached(visitor)
	r.nerve(visitor, place)

	r.fire(visitor, onEnter, place.Name)
	if tile.Site >= 0 {
		r.fire(visitor, onEnter, r.world.Sites[tile.Site].Name)
	}

	r.fire(visitor, onArrive, "")
	r.meeting(visitor)

	if tile.Site >= 0 {
		site := r.world.Sites[tile.Site]
		r.tell(visitor, site.Name+". "+site.Description)

		if site.Kind == world.SiteShrine {
			visitor.hp, visitor.mana = visitor.maxHP(), visitor.maxMana()
			r.cure(visitor)
			r.restParty(visitor)
			r.tell(visitor, "Wracają ci siły: zdrowie i moc są pełne.")
		}
	}

	if r.alive(visitor.pos) > 0 {
		r.engage(visitor)

		return
	}

	if lying := r.ground[r.world.Index(visitor.pos)]; len(lying) > 0 {
		r.tell(visitor, "Na ziemi leży: "+join(itemNames(lying))+".")
	}
}

// buildings lists what can be visited in a settlement.
func (r *Realm) buildings() []string {
	names := make([]string, 0, len(r.world.Shops))
	for _, shop := range r.world.Shops {
		names = append(names, fmt.Sprintf("%s (%s)", shop.Name, shop.Keeper))
	}

	return names
}

// placeAt returns the place at a point as the story has left it.
func (r *Realm) placeAt(pos world.Point) world.PlaceSpec {
	if place := r.world.At(pos).Place; place >= 0 && place < len(r.places) {
		return r.places[place]
	}

	return world.PlaceSpec{}
}

// reveal uncovers the map around a point; the rooms of an adventure show
// only once entered.
func (r *Realm) reveal(center world.Point) {
	radius := revealRadius
	if r.adventure() || r.blind() {
		radius = 0
	}

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if point := (world.Point{X: center.X + dx, Y: center.Y + dy}); r.world.In(point) {
				r.revealed[r.world.Index(point)] = true
			}
		}
	}
}

// foes returns the living monsters on a tile.
func (r *Realm) foes(pos world.Point) []*monster {
	var living []*monster

	for _, foe := range r.monsters[r.world.Index(pos)] {
		if foe.hp > 0 {
			living = append(living, foe)
		}
	}

	return living
}

func (r *Realm) alive(pos world.Point) int {
	return len(r.foes(pos))
}

// active returns a connected player.
func (r *Realm) active(name string) *player {
	if found := r.players[name]; found != nil && found.connected {
		return found
	}

	return nil
}

func (r *Realm) tell(listener *player, text string) {
	listener.log = append(listener.log, text)
	if len(listener.log) > logLimit {
		listener.log = listener.log[len(listener.log)-logLimit:]
	}
}

func (r *Realm) tellOthers(speaker *player, text string) {
	for _, name := range r.order {
		if listener := r.players[name]; listener != speaker {
			r.tell(listener, text)
		}
	}
}

// notify wakes every player's interface. It never blocks.
func (r *Realm) notify() {
	for _, listener := range r.players {
		select {
		case listener.updates <- struct{}{}:
		default:
		}
	}
}

func (p *player) maxHP() int {
	return max(1, BaseHP+(p.level-1)*hpPerLevel+Modifier(p.abilities[Constitution])*p.level+p.bonus(world.EffectVigor))
}

func (p *player) maxMana() int {
	return max(0, BaseMana+(p.level-1)*manaPerLevel+Modifier(p.abilities[Intelligence])*2+p.bonus(world.EffectFocus))
}

// nextLevel is the experience needed to finish the current level.
func (p *player) nextLevel() int {
	return p.level * xpPerLevel
}

func (p *player) attack() int {
	attack := baseAttack + (p.level - 1) + Modifier(p.abilities[Strength]) + p.bonus(world.EffectPower)
	if p.weapon != nil {
		attack += p.weapon.Power()
	}

	return attack
}

func (p *player) defense() int {
	defense := p.bonus(world.EffectWard)
	if p.armor != nil {
		defense += p.armor.Power()
	}

	return defense
}

// bonus is what the worn charm adds, if it has the given effect.
func (p *player) bonus(effect world.Effect) int {
	if p.charm != nil && p.charm.Effect == effect {
		return p.charm.Power()
	}

	return 0
}

func (p *player) knows(spell world.ItemSpec) bool {
	for _, known := range p.spells {
		if known.ID == spell.ID {
			return true
		}
	}

	return false
}

// xpFor is the experience a beaten monster is worth.
var xpFor = [world.MaxTier]int{10, 25, 60}

// reward gives a hero experience and raises their level when it is due. A
// new level restores health and power.
func (r *Realm) reward(hero *player, xp int) {
	if hero.level >= MaxLevel {
		return
	}

	hero.xp += xp

	for hero.level < MaxLevel && hero.xp >= hero.nextLevel() {
		hero.xp -= hero.nextLevel()
		hero.level++
		hero.hp, hero.mana = hero.maxHP(), hero.maxMana()
		r.tell(hero, fmt.Sprintf("Awans! Poziom %d: zdrowie +%d, moc +%d, atak +1. Siły wracają do pełna.", hero.level, hpPerLevel, manaPerLevel))
	}

	if hero.level == MaxLevel {
		hero.xp = 0
	}
}

func itemNames(items []world.ItemSpec) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}

	return names
}

func cleanName(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if runes := []rune(name); len(runes) > nameLimit {
		name = string(runes[:nameLimit])
	}

	if name == "" {
		return defaultName
	}

	return name
}
