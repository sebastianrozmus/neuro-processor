package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// The adventure mode: rooms joined by exits, things to find and use on one
// another, people to talk to and scripted steps that move the story on.
// Everything here is decided by the spec's steps, never by a model.

// adventure is the realm's state of an adventure.
type adventure struct {
	flags map[string]bool
	// done marks the once-only steps that have happened.
	done map[int]bool
}

func (r *Realm) adventure() bool {
	return r.world.Mode == world.ModeAdventure
}

// roomOf returns the room a player stands in.
func (r *Realm) roomOf(who *player) world.RoomSpec {
	room, _ := r.world.RoomAt(who.pos)

	return room
}

// passage reports whether a player can walk from where they stand in a
// direction, telling them what stops them otherwise.
func (r *Realm) passage(mover *player, step world.Point) bool {
	target := world.Point{X: mover.pos.X + step.X, Y: mover.pos.Y + step.Y}

	if _, ok := r.world.RoomAt(target); !ok {
		r.tell(mover, "Dalej nie ma przejścia.")

		return false
	}

	if door, shut := r.roomOf(mover).Door(dirCode(step)); shut && !r.story.flags[door.Flag] {
		r.tell(mover, door.Text)

		return false
	}

	return true
}

// dirCode is the one-letter name of a direction used by doors.
func dirCode(step world.Point) string {
	switch step {
	case north:
		return "n"
	case south:
		return "s"
	case east:
		return "e"
	}

	return "w"
}

// peopleAt lists who stands in a room.
func (r *Realm) peopleAt(pos world.Point) []world.PersonSpec {
	room, ok := r.world.RoomAt(pos)
	if !ok {
		return nil
	}

	var here []world.PersonSpec
	for _, person := range r.world.People {
		if person.Room == room.Name {
			here = append(here, person)
		}
	}

	return here
}

// personAt finds somebody in the player's room by name.
func (r *Realm) personAt(who *player, name string) (world.PersonSpec, bool) {
	for _, person := range r.peopleAt(who.pos) {
		if Matches(person.Name, name) {
			return person, true
		}
	}

	return world.PersonSpec{}, false
}

// arriveRoom tells a player what they find in a room and runs the steps
// of entering it.
func (r *Realm) arriveRoom(visitor *player) {
	if people := r.peopleAt(visitor.pos); len(people) > 0 {
		names := make([]string, 0, len(people))
		for _, person := range people {
			names = append(names, person.Name)
		}

		r.tell(visitor, "Jest tu: "+join(names)+".")
	}

	if lying := r.ground[r.world.Index(visitor.pos)]; len(lying) > 0 {
		r.tell(visitor, "Widzisz tu: "+join(itemNames(lying))+".")
	}

	r.act(visitor, world.StepEnter, "", r.roomOf(visitor).Name)
}

// exits lists where a room's open exits lead.
func (r *Realm) exits(viewer *player) []string {
	var exits []string

	for _, step := range []world.Point{north, east, south, west} {
		next := world.Point{X: viewer.pos.X + step.X, Y: viewer.pos.Y + step.Y}
		if !r.world.In(next) {
			continue
		}

		room, ok := r.world.RoomAt(next)
		if !ok {
			continue
		}

		exit := directionNames[step] + ": " + room.Name
		if door, shut := r.roomOf(viewer).Door(dirCode(step)); shut && !r.story.flags[door.Flag] {
			exit = directionNames[step] + ": zamknięte"
		}

		exits = append(exits, exit)
	}

	return exits
}

// examine describes a thing, a person or the room, running the examine
// steps first.
func (r *Realm) examine(viewer *player, what string) {
	if what == "" {
		r.look(viewer)

		return
	}

	if r.act(viewer, world.StepExamine, "", what) {
		return
	}

	if index := find(viewer.bag, what); index >= 0 {
		r.tell(viewer, viewer.bag[index].Name+". "+viewer.bag[index].Description)

		return
	}

	if lying := r.ground[r.world.Index(viewer.pos)]; find(lying, what) >= 0 {
		item := lying[find(lying, what)]
		r.tell(viewer, item.Name+". "+item.Description)

		return
	}

	if person, ok := r.personAt(viewer, what); ok {
		r.tell(viewer, person.Name+". "+person.Description)

		return
	}

	r.tell(viewer, "Nie widzisz tu niczego takiego.")
}

// handle is "użyj X", "użyj X na Y" or "otwórz Y" in an adventure.
func (r *Realm) handle(user *player, verb, what string) {
	thing, on := what, ""
	if before, after, found := strings.Cut(" "+what+" ", " na "); found {
		thing, on = strings.TrimSpace(before), strings.TrimSpace(after)
	}

	if verb == world.StepOpen {
		thing, on = "", what
	}

	if thing != "" && find(user.bag, thing) < 0 {
		if on == "" && r.act(user, verb, "", thing) {
			return
		}

		r.tell(user, "Nie masz niczego takiego.")

		return
	}

	if r.act(user, verb, thing, on) {
		return
	}

	if on == "" && thing != "" {
		r.tell(user, "Nic z tego nie wynika. Spróbuj «użyj "+thing+" na …».")

		return
	}

	r.tell(user, "Nic się nie dzieje.")
}

// hand is "daj X Y" to a person of an adventure.
func (r *Realm) hand(giver *player, what string) {
	for _, person := range r.peopleAt(giver.pos) {
		for _, word := range strings.Fields(Fold(person.Name)) {
			if before, _, found := strings.Cut(" "+Fold(what)+" ", " "+Stem(word)); found && strings.TrimSpace(before) != "" {
				thing := strings.TrimSpace(before)
				if find(giver.bag, thing) < 0 {
					r.tell(giver, "Nie masz niczego takiego.")

					return
				}

				if !r.act(giver, world.StepGive, thing, person.Name) {
					r.tell(giver, person.Name+" nie chce tego.")
				}

				return
			}
		}
	}

	r.tell(giver, "Komu? Napisz np. «daj kawę "+firstPersonHere(r, giver)+"».")
}

func firstPersonHere(r *Realm, who *player) string {
	if people := r.peopleAt(who.pos); len(people) > 0 {
		return strings.Fields(people[0].Name)[0]
	}

	return "imię"
}

// chat is "rozmawiaj z Y" in an adventure: a step of the story, or what
// the person always says.
func (r *Realm) chat(speaker *player, who string) {
	person, ok := r.personAt(speaker, who)
	if !ok {
		if people := r.peopleAt(speaker.pos); len(people) == 0 {
			r.tell(speaker, "Nie ma tu nikogo, z kim dałoby się porozmawiać.")
		} else {
			r.tell(speaker, "Z kim? Jest tu: "+join(personNames(people))+".")
		}

		return
	}

	if !r.act(speaker, world.StepTalk, "", person.Name) {
		r.tell(speaker, person.Name+": «"+person.Talk+"»")
	}
}

func personNames(people []world.PersonSpec) []string {
	names := make([]string, 0, len(people))
	for _, person := range people {
		names = append(names, person.Name)
	}

	return names
}

// act runs the first step of the story that fits what the player does
// and reports whether there was one.
func (r *Realm) act(actor *player, verb, thing, on string) bool {
	room := r.roomOf(actor).Name

	for index, step := range r.world.Steps {
		switch {
		case step.Verb != verb, on == "" || !Matches(step.On, on):
			continue
		case step.Thing != "" && (thing == "" || !Matches(step.Thing, thing)):
			continue
		case step.Thing == "" && thing != "" && verb == world.StepUse:
			continue
		case step.Room != "" && step.Room != room:
			continue
		case step.Needs != "" && !r.story.flags[step.Needs]:
			continue
		case step.Unless != "" && r.story.flags[step.Unless]:
			continue
		case step.Once && r.story.done[index]:
			continue
		}

		r.story.done[index] = true
		r.follow(actor, step)

		return true
	}

	return false
}

// follow applies a step: its words and everything it changes.
func (r *Realm) follow(actor *player, step world.StepSpec) {
	if step.Text != "" {
		r.tell(actor, step.Text)
	}

	if step.Flag != "" {
		r.story.flags[step.Flag] = true
	}

	if step.Takes && step.Thing != "" {
		if index := find(actor.bag, step.Thing); index >= 0 {
			actor.bag = remove(actor.bag, index)
		}
	}

	if item, ok := r.thing(step.Gives); ok {
		actor.bag = append(actor.bag, item)
		r.tell(actor, "Masz teraz: "+item.Name+".")
	}

	if item, ok := r.thing(step.Reveals); ok {
		index := r.world.Index(actor.pos)
		r.ground[index] = append(r.ground[index], item)
		r.tell(actor, "Widzisz tu: "+item.Name+".")
	}

	if step.Changes != nil {
		r.change(*step.Changes)
	}

	if step.Win {
		r.won = true

		for _, name := range r.order {
			r.tell(r.players[name], fmt.Sprintf("Przygoda «%s» ukończona!", r.world.Name))
		}
	}
}

// thing finds an item of the world by its exact name.
func (r *Realm) thing(name string) (world.ItemSpec, bool) {
	if name == "" {
		return world.ItemSpec{}, false
	}

	for _, item := range r.world.Items {
		if item.Name == name {
			return item, true
		}
	}

	return world.ItemSpec{}, false
}

// change turns a room into what a step makes of it: the map and the scene
// follow, since both draw the realm's places.
func (r *Realm) change(into world.RoomChange) {
	for index, place := range r.places {
		if place.Name != into.Room {
			continue
		}

		if into.Description != "" {
			place.Description = into.Description
		}

		if into.Terrain != "" {
			place.Terrain = into.Terrain
		}

		r.places[index] = place

		for _, name := range r.order {
			if r.world.PlaceAt(r.players[name].pos).Name == into.Room {
				r.tell(r.players[name], place.Description)
			}
		}
	}
}

// AdventureHelp lists the commands of an adventure.
const AdventureHelp = "Polecenia: idź na północ / południe / wschód / zachód (albo n, s, e, w) · rozejrzyj się · zbadaj X · podnieś X · upuść X · " +
	"użyj X · użyj X na Y · otwórz X · daj X Y · rozmawiaj z Y · plecak · zapisz · pomoc · wyjdź. " +
	"Strzałki też chodzą, Tab otwiera plecak, a pusty Enter powtarza ostatnie polecenie."
