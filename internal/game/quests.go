package game

import (
	"fmt"
	"strings"

	"github.com/sebastianrozmus/neuro-processor/internal/world"
)

// Quests grow out of the knowledge graph: a fact a keeper has sat on for
// a couple of days turns into a job for whoever walks in. A fact about
// monsters asks for a hunt; one about a place or a site asks for a visit.
// The keeper pays in gold from the till, sometimes with a thing from the
// shelf, and the settlement remembers who did it.

type questKind string

const (
	questHunt  questKind = "hunt"
	questVisit questKind = "visit"
)

const (
	// questAge is how many days a fact waits before it becomes a job;
	// questPercent the chance per day that a keeper with none posts one.
	questAge     = 2
	questPercent = 50
	questItemAt  = 40
)

// quest is one job.
type quest struct {
	id      int
	keeper  int
	kind    questKind
	subject string
	// monster is the kind hunted; count how many; done how many so far.
	monster int
	count   int
	done    int
	reward  int
	item    string
	// taker is who took the job; finished and paid its progress; day
	// when it was posted.
	taker    string
	finished bool
	paid     bool
	day      int
}

// QuestView is a job as the journal shows it.
type QuestView struct {
	Keeper   string
	Text     string
	Progress string
	Taken    bool
	Finished bool
}

// postQuests lets keepers turn aged facts into jobs, one open job per
// keeper at a time.
func (r *Realm) postQuests() {
	for _, keeper := range r.everyone() {
		if (r.caravan != nil && keeper == r.caravanShop()) || r.openQuest(keeper) != nil || r.rng.Intn(100) >= questPercent {
			continue
		}

		var aged []*Fact
		for _, fact := range r.knownBy(keeper) {
			if r.day-fact.Day >= questAge && !r.quested(fact.Subject) {
				aged = append(aged, fact)
			}
		}

		r.rng.Shuffle(len(aged), func(a, b int) { aged[a], aged[b] = aged[b], aged[a] })

		for _, fact := range aged {
			if posted := r.questFrom(keeper, fact); posted != nil {
				r.quests = append(r.quests, posted)

				break
			}
		}
	}
}

// openQuest is a keeper's job that is still posted or being done.
func (r *Realm) openQuest(keeper int) *quest {
	for _, job := range r.quests {
		if job.keeper == keeper && !job.paid {
			return job
		}
	}

	return nil
}

// quested reports whether a subject has been a job already.
func (r *Realm) quested(subject string) bool {
	for _, job := range r.quests {
		if strings.EqualFold(job.subject, subject) {
			return true
		}
	}

	return false
}

// questFrom makes a job of a fact, if the fact is about something a job
// can be about.
func (r *Realm) questFrom(keeper int, fact *Fact) *quest {
	job := &quest{id: len(r.quests) + 1, keeper: keeper, subject: fact.Subject, day: r.day}

	for kind, monster := range r.world.Monsters {
		if strings.EqualFold(monster.Name, fact.Subject) {
			job.kind, job.monster, job.count = questHunt, kind, 2+r.rng.Intn(2)
			job.reward = monster.Tier * 15 * job.count

			return r.withItem(job)
		}
	}

	for _, site := range r.world.Sites {
		if strings.EqualFold(site.Name, fact.Subject) {
			job.kind, job.reward = questVisit, 45

			return r.withItem(job)
		}
	}

	for _, place := range r.places {
		if place.Danger > 0 && strings.EqualFold(place.Name, fact.Subject) {
			job.kind, job.reward = questVisit, 20*place.Danger

			return r.withItem(job)
		}
	}

	return nil
}

// withItem sometimes adds a thing from the keeper's shelf to the reward.
func (r *Realm) withItem(job *quest) *quest {
	if r.rng.Intn(100) >= questItemAt {
		return job
	}

	var shelf []world.ItemSpec
	for _, item := range r.world.Items {
		if r.world.ShopFor(item.Kind) == job.keeper && item.Tier <= 2 {
			shelf = append(shelf, item)
		}
	}

	if len(shelf) > 0 {
		job.item = shelf[r.rng.Intn(len(shelf))].ID
	}

	return job
}

// questText words a job.
func (r *Realm) questText(job *quest) string {
	var text string

	switch job.kind {
	case questHunt:
		text = fmt.Sprintf("ubij %d sztuk: %s", job.count, job.subject)
	default:
		text = "dotrzyj do miejsca: " + job.subject + ", i wróć"
	}

	reward := fmt.Sprintf("%d %s", job.reward, r.world.Currency)
	if item, ok := r.item(job.item); ok {
		reward += " i " + item.Name
	}

	return fmt.Sprintf("%s. Nagroda: %s", text, reward)
}

// offerQuest tells a customer about the keeper's job, if one is posted.
func (r *Realm) offerQuest(customer *player, keeper int) {
	job := r.openQuest(keeper)
	if job == nil {
		return
	}

	switch {
	case job.taker == "":
		r.tell(customer, fmt.Sprintf("Zlecenie od %s: %s. Podejmiesz się poleceniem «przyjmij».", r.shopSpec(keeper).Keeper, r.questText(job)))
	case job.taker == customer.name && job.finished:
		r.payQuest(customer, job)
	case job.taker == customer.name:
		r.tell(customer, "Twoje zlecenie stąd: "+r.questText(job)+". "+r.progress(job))
	}
}

// acceptQuest takes the job posted where the hero stands: by the keeper
// they talk to, or the one open job of the settlement.
func (r *Realm) acceptQuest(hero *player) {
	var offered []*quest

	for _, job := range r.quests {
		if job.taker == "" && (hero.shop == job.keeper || (hero.shop < 0 && r.townOf(hero) >= 0)) {
			offered = append(offered, job)
		}
	}

	switch {
	case len(offered) == 0:
		r.tell(hero, "Nikt tu teraz nie ma dla ciebie zlecenia. Pytaj po sklepach, wracaj za dzień lub dwa.")
	case len(offered) > 1 && hero.shop < 0:
		r.tell(hero, "Zleceń jest kilka: wejdź do kramu tego, kogo chcesz wysłuchać.")
	default:
		job := offered[0]
		job.taker, job.done = hero.name, 0
		r.tell(hero, fmt.Sprintf("Przyjmujesz zlecenie od %s: %s. Po wykonaniu wróć do niego po nagrodę.", r.shopSpec(job.keeper).Keeper, r.questText(job)))
	}
}

// progress words how far a job has come.
func (r *Realm) progress(job *quest) string {
	switch {
	case job.finished:
		return "Wykonane: wróć po nagrodę."
	case job.kind == questHunt:
		return fmt.Sprintf("Ubite: %d z %d.", job.done, job.count)
	}

	return "Jeszcze tam nie dotarłeś."
}

// hunted counts a kill towards the hunts of everyone in the fight.
func (r *Realm) hunted(current *fight, foe *monster) {
	for _, name := range current.members {
		for _, job := range r.quests {
			if job.taker != name || job.kind != questHunt || job.finished || job.monster != foe.kind {
				continue
			}

			job.done++
			if job.done >= job.count {
				r.finishQuest(r.players[name], job)
			} else {
				r.tell(r.players[name], fmt.Sprintf("Zlecenie: %d z %d.", job.done, job.count))
			}
		}
	}
}

// reached checks whether where a hero arrived is what a visit asked for.
func (r *Realm) reached(hero *player) {
	tile := r.world.At(hero.pos)
	names := []string{r.placeAt(hero.pos).Name}
	if tile.Site >= 0 {
		names = append(names, r.world.Sites[tile.Site].Name)
	}

	for _, job := range r.quests {
		if job.taker != hero.name || job.kind != questVisit || job.finished {
			continue
		}

		for _, name := range names {
			if strings.EqualFold(name, job.subject) {
				r.finishQuest(hero, job)
			}
		}
	}
}

func (r *Realm) finishQuest(hero *player, job *quest) {
	job.finished = true
	r.tell(hero, fmt.Sprintf("Zlecenie wykonane. Wróć do %s po nagrodę.", r.shopSpec(job.keeper).Keeper))
}

// payQuest settles a finished job: gold, the promised thing, standing in
// the settlement and a fact everyone will tell.
func (r *Realm) payQuest(hero *player, job *quest) {
	job.paid = true
	hero.gold += job.reward
	keeper := r.shopSpec(job.keeper).Keeper
	r.tell(hero, fmt.Sprintf("%s płaci za zlecenie: %d %s.", keeper, job.reward, r.world.Currency))

	if item, ok := r.item(job.item); ok {
		hero.bag = append(hero.bag, item)
		r.tell(hero, "Do tego dostajesz: "+item.Name+".")
	}

	r.esteem(hero, r.townOf(hero), questStanding, "Wieść o wykonanej robocie się niesie.")

	if fact := r.add(job.subject, fmt.Sprintf("%s wykonał zlecenie %s: %s.", hero.name, keeper, r.questText(job)), keeper, r.everyone()...); fact != nil {
		r.learn(hero, fact)
	}
}

// questLore tells a keeper about their job and the customer's part in it.
func (r *Realm) questLore(customer *player, keeper int) string {
	job := r.openQuest(keeper)
	if job == nil {
		return ""
	}

	switch {
	case job.taker == "":
		return "Masz zlecenie do oddania: " + r.questText(job) + ". Wspomnij o nim, jeśli rozmowa na to pozwala."
	case job.taker == customer.name:
		return "Gracz wykonuje twoje zlecenie (" + r.questText(job) + "). " + r.progress(job)
	}

	return "Twoje zlecenie wziął już ktoś inny: " + job.taker + "."
}

// Quests lists a player's jobs and the ones on offer where they stand.
func (r *Realm) Quests(name string) []QuestView {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hero := r.players[name]; hero != nil {
		return r.questViews(hero)
	}

	return nil
}

func (r *Realm) questViews(hero *player) []QuestView {
	var view []QuestView

	for _, job := range r.quests {
		switch {
		case job.paid:
			continue
		case job.taker == hero.name:
			view = append(view, QuestView{Keeper: r.shopSpec(job.keeper).Keeper, Text: r.questText(job), Progress: r.progress(job), Taken: true, Finished: job.finished})
		case job.taker == "" && r.townOf(hero) >= 0:
			view = append(view, QuestView{Keeper: r.shopSpec(job.keeper).Keeper, Text: r.questText(job), Progress: "Do wzięcia tu: «przyjmij»."})
		}
	}

	return view
}

// QuestLines words a player's jobs for the log.
func QuestLines(jobs []QuestView) []string {
	if len(jobs) == 0 {
		return []string{"Nie masz zleceń. Rozmawiaj z kupcami: z tego, co wiedzą, rodzą się zadania."}
	}

	lines := []string{fmt.Sprintf("Zlecenia (%d):", len(jobs))}
	for _, job := range jobs {
		lines = append(lines, fmt.Sprintf("• %s: %s. %s", job.Keeper, job.Text, job.Progress))
	}

	return lines
}
