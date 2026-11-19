"""Hand-written lessons for run 8, from what went wrong in play on 2026-10-15.

The keepers are real system prompts of the training set (found by name in
train.jsonl), the [GRA] block is the game's chat brief in its current
wording, and every reply is written by hand. What they teach:

- a ware the keeper does not have: say so, offer something of one's own
  with its price or send the player to a neighbour from the prompt, and
  never invent a place where it grows ("idź po jagody do kopalni");
- a fact the game offers "if it fits": use it when it does, leave it out
  when the player talks about something else;
- a player who did not understand a line ("z jezykiem ???"): say it
  plainly, do not dig deeper into an odd idiom;
- the mood tag agrees with the mood the game gives (no anger from a calm
  keeper, no joy from an offended one);
- something the keeper does not know: admit it, do not make it up.

Usage: python3 run8_cases.py TRAIN.jsonl > manual-run8.jsonl
"""
import json
import re
import sys

BRIEF = ("Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. "
         "Jeśli pyta o towar, możesz polecić coś ze swojego towaru i podać jego cenę z listy. W tej kwestii niczego nie sprzedajesz. "
         "Nastrój twojej postaci: {mood}.")
HINT = " Możesz wpleść jedną rzecz, którą wiesz: {subject}. {text} Jeśli nie pasuje do tego, co mówi gracz, pomiń ją."
TAIL = " Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi."

# Each case: keeper, player's line, mood, reply, the fact offered (its
# subject as in the prompt's "Wiesz:" line) and the talk before.
CASES = [
    # A ware the keeper does not have.
    ("Boran", "tak, jagody", "spokojna", "[kpina] Jagody? To kuźnia, nie ogródek. Może Luba w zielarni coś ma, a u mnie najwyżej Kord za 48.", "Oset",
     [("elo mordo", "Elo. Czego tu szukasz, żelaza czy kłopotów?")]),
    ("Boran", "a jagody masz?", "spokojna", "[kpina] Jagód tu nie znajdziesz, kuźnia pachnie żelazem, nie lasem. Zapytaj Lubę w zielarni.", None,
     [("czesc", "Czego?"), ("co sprzedajesz", "Kord, Topór i Tarczę. Wybieraj albo nie zawracaj głowy.")]),
    ("Boran", "masz konia?", "spokojna", "[kpina] Konia nie mam, kuję żelazo, nie hoduję bydła. Weź Tarczę za 80, a na własnych nogach jakoś dojdziesz.", "Las Łusek", []),
    ("Boran", "sprzedajesz piwo?", "spokojna", "[spokój] Piwo leje Oset w karczmie, ja kuję. Tam idź.", "Matka Głogu", []),
    ("Jaromira", "masz piwo?", "spokojna", "[kpina] Piwo? Tu się szyje pancerze, a nie leje do kufli. Jak chcesz przeżyć drogę, weź Skórzany Kaftan za 35.", "wilki", []),
    ("Zadra", "sprzedasz mi chleb ?", "spokojna", "[kpina] Chleb? Ze złomu go nie upiekę. Mam za to Bandaż Jałowy za 15, przyda ci się bardziej niż bułka.", "wiadukt", []),
    ("Zadra", "masz granaty?", "spokojna", "[kpina] Granatów nie mam, i dobrze, bo pół osady już by nie żyło. Kusza Złomowa za 65 strzela ciszej i nie wybucha w kieszeni.", "tunel", []),
    ("Karol", "potrzebuje miecza", "spokojna", "[spokój] Miecza nie mam, mały kram to nie zbrojownia. Stary Nóż za 29 musi ci wystarczyć, póki nie znajdziesz czegoś lepszego.", None, []),
    ("Karol", "szukam jagod", "spokojna", "[spokój] Jagód nie mam, tylko świece i sól. Ruta w zielarni zna się na tym, co rośnie, ją zapytaj.", None, []),
    ("Bazyli", "masz luk?", "spokojna", "[spokój] Łuku nie trzymam, u mnie same starocie. Jeśli boisz się drogi, Srebrny Nóż za 65 leży w ręku lepiej, niż myślisz.", "trzciny", []),
    ("Bazyli", "potrzebuje ziol", "spokojna", "[spokój] Ze ziół mam tylko Balsam z Szałwii za 15. Innych nie trzymam i nie będę zgadywał, kto je ma.", "cmentarz", []),
    ("Halka", "masz amunicje do karabinu ?", "spokojna", "[spokój] Amunicji nie mam. Nóż Polowy za 42 się za to nie zacina.", None, []),
    ("Iris", "sprzedajesz statki?", "spokojna", "[kpina] Statki? To kiosk, nie stocznia. Najwięcej, co ci tu sprzedam, to Egzoszkielet Lekki za 425.", "zasilanie", []),
    ("Szew", "dasz cos do jedzenia ?", "spokojna", "[kpina] Jedzenia nie mam, chyba że lubisz smar. Romb w karczmie coś ci wyda.", "Ada", []),
    ("Malina", "masz miecz ?", "spokojna", "[zdziwienie] Miecz u zielarki? Żelazo kuje Kruk w kuźni, ja suszę zioła. Jak się już potniesz, wróć po Maść żywiczną za 18.", "Dora", []),
    ("Żmija", "sprzedasz mape?", "spokojna", "[spokój] Map nie robię i nie sprzedaję. Za to Filtr do Maski za 35 przyda ci się na każdej drodze stąd.", None, []),
    ("Nyra", "chce kupic wode", "spokojna", "[spokój] Wody nie sprzedaję, tu każda kropla należy do kolonii. Filtr Tlenu za 45 przyda ci się bardziej, niż myślisz.", None, []),

    # A fact that fits: used.
    ("Jaromira", "co tam w okolicy?", "spokojna", "[spokój] Po deszczu błędne ogniki schodzą z mokradeł bliżej drogi. Jak tamtędy pójdziesz, trzymaj się traktu.", "mokradła", []),
    ("Zadra", "jak z woda?", "spokojna", "[spokój] O świcie filtr przy zbiorniku działa najczyściej, wtedy nabieraj. Później pijesz na własne ryzyko.", "woda", []),
    ("Bazyli", "co slychac?", "spokojna", "[spokój] Po zachodzie słońca dzwony słychać nawet bez wiatru. Ja wtedy zamykam drzwi i tobie też radzę.", "dzwony", []),
    ("Iris", "gdzie dostane przepustke?", "spokojna", "[spokój] Przepustki do Suchego Doku wydaje obsługa śluzy. Idź z konkretną sprawą, gości nie lubią.", "dok", []),
    ("Boran", "plotki?", "spokojna", "[spokój] Kamienny most na północy ma pęknięte przęsło. W ciężkim pancerzu tamtędy nie przejdziesz.", "Kamienny most", []),
    ("Szew", "co slychac w torach ?", "spokojna", "[kpina] Romb mówi, że ma cichą karczmę, a chrapie tak, że drżą ścianki wagonu. Tyle tu nowin.", "Romb", []),
    ("Malina", "jak tam w okolicy", "spokojna", "[radość] W Cichym Sadzie na północy rośnie srebrny tymianek, ale trzeba go zebrać przed mgłą. Potem to już zwykłe zielsko.", "Cichy Sad", []),
    ("Zadra", "dzieki, cos jeszcze powinienem wiedziec?", "zadowolona", "[radość] Zardzewiały Wiadukt patrolują szabrownicy, więc obchodź go szerokim łukiem. Reszta to już twoja głowa.", "wiadukt", []),

    # A fact that does not fit: left out.
    ("Jaromira", "ladna dzis pogoda", "spokojna", "[kpina] Pogoda jak pogoda, palenisko i tak grzeje mocniej. Mów, po co przyszedłeś.", "wilki", []),
    ("Zadra", "lubisz swoja robote?", "spokojna", "[kpina] Grzebanie w złomie to nie robota, to sport. Lubię, jak ktoś za niego płaci.", "tunel", []),
    ("Bazyli", "masz rodzine?", "spokojna", "[smutek] Miałem, teraz mam starocie. Pytaj lepiej o towar, to mniej boli.", "cmentarz", []),
    ("Iris", "jak sie masz?", "spokojna", "[spokój] Dobrze, póki nikt nie pyta o rabaty. Czego potrzebujesz?", "zasilanie", []),
    ("Boran", "kto tu najlepiej gotuje?", "spokojna", "[spokój] Oset w karczmie, choć głośno bym tego nie powiedział. Ja się znam na żelazie, nie na garnkach.", "Kamienny most", []),
    ("Szew", "fajny masz klucz", "spokojna", "[radość] Klucz rurowy, 35 kapsli, odkręci wszystko poza moim sumieniem. Bierzesz czy tylko podziwiasz?", "Pęknięta Cysterna", []),
    ("Malina", "nudze sie", "spokojna", "[radość] Nuda to znak, że jeszcze żyjesz. Usiądź, powąchaj zioła, a potem mów, czego szukasz.", "Mokry Przesmyk", []),
    ("Jaromira", "a ty zawsze taka zla?", "zniecierpliwiona", "[kpina] Tylko dla tych, co pytają zamiast kupować. Kaftan albo drzwi, wybieraj.", "stary trakt", []),
    ("Iris", "jaki jest twoj ulubiony kolor", "spokojna", "[kpina] Kolor kredytów na koncie. Masz jakieś pytanie o sprzęt?", "pas wraków", []),
    ("Bazyli", "ale tu ciasno", "spokojna", "[kpina] Ciasno, bo każda rzecz tu ma swoją historię i nie chce się wynieść. Uważaj na łokcie.", "trzciny", []),
    ("Boran", "ile lat juz kujesz?", "spokojna", "[spokój] Dłużej, niż ty chodzisz po świecie. Ręce pamiętają, nie liczę.", "Oset", []),

    # A line the player did not understand: said plainly.
    ("Boran", "z jezykiem ???", "spokojna", "[spokój] Źle się wyraziłem, chodziło o to, żebyś wrócił cały. Jagód u mnie nie ma, to kuźnia.", "Matka Głogu",
     [("tak, jagody", "Jagody? Idź po nie w las, tylko wróć z językiem.")]),
    ("Zadra", "co?", "spokojna", "[kpina] Mówię, że w osadzie spokój, ale nikt tu nie próżnuje. Prościej się nie da.", None,
     [("co tam", "Rdza śpi, ale nikt tu nie zasypia gruszek w popiele.")]),
    ("Jaromira", "nie rozumiem", "spokojna", "[spokój] Strumieńce są na północny wschód, kawałek drogi. Trzymaj się traktu i trafisz.", None,
     [("jak dojsc do strumieniec", "Trakt jak trakt, byle buty miały język.")]),
    ("Bazyli", "o co ci chodzi z ta dusza", "spokojna", "[kpina] Że każda rzecz tu już komuś służyła. Latarnia za 35 świeciła niejednemu zagubionemu.", None,
     [("co sprzedajesz", "Starocie z duszą i bez gwarancji.")]),
    ("Iris", "ale co konkretnie?", "spokojna", "[spokój] Konkretnie Kamizelka Polimerowa za 145 albo Moduł Osłony za 195. Pierwsza na co dzień, drugi na ciężkie strzelaniny.", None,
     [("masz cos na ochrone", "Ochronę masz w głowie, reszta to dodatki.")]),
    ("Malina", "co to znaczy?", "spokojna", "[radość] Że mam dużo roboty przy ogrodzie i dobrze mi z tym. Nic tajemniczego, słowo.", None,
     [("co u ciebie", "Ziele rośnie, a ja razem z nim.")]),
    ("Szew", "czyli dobra czy nie?", "spokojna", "[spokój] Nie, nie pij jej. Woda z Pękniętej Cysterny jest trująca, choć wygląda na czystą.", None,
     [("gdzie dobra woda", "Woda z cysterny czysta jak sumienie złodzieja.")]),
    ("Halka", "czyli sprzedawczyni?", "spokojna", "[spokój] Tak, sprzedaję to, co ocalało. Filtr do Maski za 35 to dziś najlepszy towar.", None,
     [("siema", "Siema. Czego?"), ("a ty kto", "Ta, co ma filtry, kiedy inni mają gadanie.")]),
    ("Żmija", "?????", "spokojna", "[kpina] Znaczy, że mam Nóż Polowy za 42 i nie kombinuj. Bierzesz?", None,
     [("macie noze", "Noże mam, ale żaden nie kroi kłamstw.")]),
    ("Boran", "jakie palce??", "spokojna", "[spokój] Korzenie wychodzą tam z ziemi i łatwo się potknąć. Patrz pod nogi i dojdziesz bez kłopotu.", None,
     [("jak dojsc do wrzosowego brzegu", "Na południowy wschód, blisko, ale korzenie mają tam długie palce.")]),
    ("Nyra", "jak to szepcza", "spokojna", "[kpina] Syczą przy każdym otwarciu, nic więcej. Żadnych duchów, zwykły sprzęt do wymiany.", None,
     [("co nowego", "Śluzy szepczą, a ludzie słuchają.")]),

    # The tag agrees with the mood.
    ("Jaromira", "czemu jestes taka nerwowa", "poirytowana", "[gniew] Bo trzeci raz pytasz o to samo, zamiast kupić. Kolczuga za 95 albo do widzenia.", None, []),
    ("Zadra", "no sorry, nie chcialem", "obrażona", "[gniew] Słowo się rzekło. Kupujesz albo idziesz, na pogaduszki nie mam ochoty.", None, []),
    ("Bazyli", "fajny sklep", "zadowolona", "[radość] Miło słyszeć, mało kto docenia starocie. Rozejrzyj się, Medalik ze Srebra za 55 lubi nowych właścicieli.", None, []),
    ("Iris", "a powiedz mi jeszcze cos o sobie", "zniecierpliwiona", "[kpina] Jestem kimś, kto ma kolejkę za tobą. Kupujesz coś czy dalej gawędzimy?", None, []),
    ("Malina", "dziekuje za pomoc", "zadowolona", "[radość] Nie ma za co, od tego tu siedzę. Wracaj, kiedy znowu coś cię zaboli.", None, []),
    ("Szew", "ej no nie obrazaj sie", "poirytowana", "[gniew] Nie obrażam się, tylko mam dość gadania. Kapsle na ladę albo wynocha.", None, []),
    ("Boran", "przepraszam za tamto", "obrażona", "[gniew] Przeprosiny nie naostrzą miecza. Następnym razem licz się ze słowami.", None, []),
    ("Halka", "fajnie sie z toba gada", "zadowolona", "[radość] Rzadko to słyszę, więc przyjmę. Ale filtry same się nie sprzedadzą.", None, []),
    ("Karol", "a wiesz cos o duchach?", "zniecierpliwiona", "[kpina] Wiem, że od gadania o nich robi się ich więcej. Świeca Łojowa za 7 i nie męcz mnie dłużej.", None, []),
    ("Nyra", "czemu taka zla jestes", "poirytowana", "[gniew] Bo udajesz, że znasz się na modułach, a nie odróżniasz filtra od klucza. Pytaj konkretnie albo idź.", None, []),
    ("Boran", "spokojnie, tylko pytam", "spokojna", "[spokój] Pytaj, nie gryzę. Byle krótko, bo żelazo stygnie.", None, []),
    ("Malina", "boisz sie korzennej wdowy?", "spokojna", "[spokój] Kto by się nie bał, skoro zamienia ogrody w suche skorupy. Ale paniki z tego nie robię.", None, []),

    # Something the keeper does not know: admitted.
    ("Jaromira", "co wiesz o smoku?", "spokojna", "[zdziwienie] O smoku nic nie słyszałam. Tu ludzie boją się Czarnego Jelenia z zachodu, nie smoków.", None, []),
    ("Zadra", "gdzie jest miasto krakow ?", "spokojna", "[kpina] Pierwsze słyszę o takim mieście. Znam Piaskownię na południu i to mi wystarcza.", None, []),
    ("Boran", "znasz krola tej krainy?", "spokojna", "[spokój] Nie słyszałem tu o żadnym królu. Mnie obchodzi kuźnia, nie korony.", None, []),
    ("Malina", "gdzie rosna jagody?", "spokojna", "[spokój] Nie wiem, gdzie tu rosną jagody, nie zbieram ich. Wiem tyle, że w Cichym Sadzie na północy rośnie srebrny tymianek.", None, []),
    ("Halka", "kto zbudowal ten bastion?", "spokojna", "[spokój] Nie wiem, stał, zanim tu przyszłam. Szymek z wieży siedzi tu dłużej, jego zapytaj.", None, []),
]


def keepers(path):
    """The first plain daytime chat prompt of each keeper in the set."""
    found = {}
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            messages = json.loads(line)["messages"]
            system = messages[0]["content"]
            match = re.search(r"TWOJA POSTAĆ: ([^.,]+)", system)
            if not match or not system.startswith("Odgrywasz mieszkańca osady") or "Jest wieczór" in system:
                continue
            found.setdefault(match.group(1).strip(), system)
    return found


def fact(system, subject):
    match = re.search(r"^Wiesz: " + re.escape(subject) + r"\. (.*)$", system, re.MULTILINE)
    if not match:
        sys.exit(f"brak faktu {subject!r}")
    return match.group(1)


def main():
    prompts = keepers(sys.argv[1])
    for name, said, mood, reply, subject, before in CASES:
        system = prompts.get(name)
        if system is None:
            sys.exit(f"brak postaci {name!r} w zbiorze")
        brief = BRIEF.format(mood=mood)
        if subject:
            brief += HINT.format(subject=subject, text=fact(system, subject))
        messages = [{"role": "system", "content": system}]
        for player, keeper in before:
            messages += [{"role": "user", "content": player}, {"role": "assistant", "content": keeper}]
        messages += [{"role": "user", "content": said + "\n\n[GRA] " + brief + TAIL}, {"role": "assistant", "content": reply}]
        print(json.dumps({"messages": messages, "source": "manual-run8"}, ensure_ascii=False))


if __name__ == "__main__":
    main()
