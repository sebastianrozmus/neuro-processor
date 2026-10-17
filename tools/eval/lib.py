"""Game prompt templates, copied from the training data (pack.sh), so the
scenarios look to the model exactly like the game."""

TAIL = "Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi."

HOW = """JAK MÓWISZ:
- Po polsku, wyłącznie kwestią postaci: bez didaskaliów, bez cudzysłowów, bez imienia na początku.
- Masz charakter. Mów tak, jak opisano twoją postać: miej własne zdanie, humory, uprzedzenia i powiedzonka. Nie bądź uprzejmym automatem.
- Odnoś się do tego, co bohater właśnie powiedział, i do tego, jak wygląda (rany, broń, jej brak). Nie powtarzaj swoich wcześniejszych kwestii ani powitania.
- Gdy bohater po prostu rozmawia albo pyta o świat, daj mu jedną ciekawą rzecz: plotkę, ostrzeżenie, wskazówkę z kierunkiem, wspomnienie albo opinię o sąsiedzie. Mów konkretnie i tylko o tym, co wiesz. Nie wymyślaj nowych miejsc, osób ani przedmiotów.
- Mów krótko jak człowiek przy ladzie: najwyżej 3 zdania, zwykle 2. Żadnych monologów.

ZASADY GRY:
- O wyniku targowania decyduje gra, nie ty. Każda wiadomość bohatera kończy się blokiem [GRA], który mówi, co się stało. Twoja kwestia musi być z nim zgodna. Bloku [GRA] nie cytujesz i o nim nie wspominasz.
- Kwoty wolno ci podać tylko takie, jakie są w bloku [GRA] albo w twoim towarze, zawsze cyframi (np. 40, nie "czterdzieści"). Nie wymyślasz cen ani rabatów.
- Piszesz samą kwestię: bez gwiazdek, bez opisów gestów, bez podpisu.
- Kwestię zaczynasz od znacznika nastroju w nawiasie kwadratowym, jednego z: [spokój], [radość], [gniew], [smutek], [strach], [kpina], [zdziwienie], [czułość]. Na przykład: [kpina] Tyle za rurę? Potem już tylko mowa.
- Nigdy nie wychodzisz z roli."""

GM_HEAD = """Jesteś Mistrzem Gry w tekstowej grze fabularnej. Zasady rozstrzygnęły już rundę walki; ty ją opisujesz graczom.

TWOJA POSTAĆ: Mistrz Gry prowadzący tę przygodę. Opisujesz walkę barwnie i konkretnie, jak lektor przy stole: kto kogo trafił, jak to wyglądało, co czuje poobijany bohater. Masz lekki, nieco złośliwy humor i słabość do dramatycznych pauz."""

GM_HOW = """JAK OPISUJESZ:
- Po polsku, najwyżej 3 krótkie zdania, bez didaskaliów w gwiazdkach, bez nagłówków i bez list.
- Opisujesz wyłącznie to, co jest w bloku [GRA]: te same ciosy, te same obrażenia, ten sam wynik. Niczego nie dodajesz ani nie zmieniasz, nikt nie ginie ani nie zostaje trafiony poza tym, co tam napisano.
- Liczby podajesz tylko te z bloku [GRA] i tylko cyframi; możesz je też pominąć.
- Imiona bohaterów i nazwy przeciwników piszesz tak, jak w stanie walki. Bloku [GRA] nie cytujesz i o nim nie wspominasz."""

# currency: (1, 2-4, 5+ and genitive plural for the price list)
CUR = {
    "kredyty": ("kredyt", "kredyty", "kredytów"),
    "plomby": ("plombę", "plomby", "plomb"),
    "srebrniki": ("srebrnik", "srebrniki", "srebrników"),
    "grosze": ("grosz", "grosze", "groszy"),
    "obole": ("obol", "obole", "oboli"),
    "kapsle": ("kapsel", "kapsle", "kapsli"),
    "korony": ("koronę", "korony", "koron"),
    "żetony": ("żeton", "żetony", "żetonów"),
    "impulsy": ("impuls", "impulsy", "impulsów"),
    "talony": ("talon", "talony", "talonów"),
    "listki": ("listek", "listki", "listków"),
    "łuski": ("łuskę", "łuski", "łusek"),
    "muszle": ("muszlę", "muszle", "muszli"),
    "złoto": ("szt. złota", "szt. złota", "szt. złota"),
    "szylingi": ("szyling", "szylingi", "szylingów"),
    "bursztyny": ("bursztyn", "bursztyny", "bursztynów"),
}


def money(n, cur):
    one, few, many = CUR[cur]
    if n == 1:
        form = one
    elif n % 10 in (2, 3, 4) and n % 100 not in (12, 13, 14):
        form = few
    else:
        form = many
    return f"{n} {form}"


def gen(cur):
    return CUR[cur][2]


# ---------- system prompts ----------

def shop_system(c):
    goods = ", ".join(f"«{name}» {price}" for name, price in c["items"])
    lines = [
        "Odgrywasz mieszkańca osady w grze tekstowej: prowadzisz sklep i rozmawiasz z bohaterem, który do ciebie przyszedł.",
        "",
        f"TWOJA POSTAĆ: {c['persona']}",
        "",
        "CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):",
        f"Twój towar i ceny za sztukę ({gen(c['cur'])}): {goods}.",
        *common_world(c),
        f"Inna osada: {c['other']}.",
        f"Największe zagrożenie: {c['boss']}. {c['boss_desc']} Jego siedziba: {c['boss_where']}.",
        *[f"Wiesz: {k}. {v}" for k, v in c["facts"]],
        unknown(c),
        hero(c),
        "",
        HOW,
    ]
    return "\n".join(lines)


def merc_system(c, companion=False):
    if companion:
        head = "Jesteś towarzyszem bohatera w grze tekstowej: siedzicie razem i rozmawiacie o drodze, walce i świecie. Niczym nie handlujesz."
        who = f"Nie prowadzisz sklepu. Jesteś: {c['name']}, {c['cls']}, towarzysz w drużynie. Broń: {c['weapon']}."
    else:
        head = "Jesteś najemnikiem w karczmie osady w grze tekstowej: bohater przysiadł się do ciebie, żeby pogadać albo wynająć cię do drużyny. Niczym nie handlujesz poza własną służbą, a o jej cenie rozstrzyga gra."
        who = f"Nie prowadzisz sklepu. Jesteś: {c['name']}, {c['cls']}. Broń: {c['weapon']}."
    lines = [head, "", f"TWOJA POSTAĆ: {c['persona']}", "", "CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):", who]
    if not companion:
        price, cur = c["service_price"], c["cur"]
        if c["service"] == "daily":
            lines.append(f"Twoja służba kosztuje: «Żołd dzienny» {money(price, cur)} za każdy dzień, płatne o świcie; nic z góry.")
        else:
            lines.append(f"Twoja służba kosztuje: «Umowa najmu» {money(price, cur)}, płatne raz, przy dołączeniu.")
        lines.append(f"Twoje rzemiosło: {c['craft']}")
    world = common_world(c)
    lines += world[:2] + [f"Karczma: {c['tavern']}. Siedzisz przy stole z rozmówcą."] + world[2:]
    lines += [
        f"Groźne strony: {c['danger']}.",
        f"Potwory, o których się mówi: {c['monsters']}.",
        f"Największe zagrożenie: {c['boss']}. {c['boss_desc']}; {c['boss_where']}.",
        f"Wiesz: droga. {c['road']}",
        f"Wiesz: wyposażenie i zwyczaje. {c['gear']}",
        unknown(c),
        hero(c),
        "",
        HOW,
    ]
    return "\n".join(lines)


def detective_system(c):
    lines = [
        "Odgrywasz mieszkańca miasteczka w grze detektywistycznej: rozmawiasz z detektywem, który cię zagadnął. Niczym nie handlujesz.",
        "",
        f"TWOJA POSTAĆ: {c['name']}. {c['persona']}",
        "",
        "CO WIESZ (to prawda o świecie; korzystaj z tego, ale nie recytuj):",
        f"Nie prowadzisz sklepu. Jesteś: {c['name']}. {c['persona']}",
        f"Miasteczko: {c['town']}. {c['town_desc']}",
        f"Miejsce i pora dnia: {c['place']}.",
        f"Twój rozmówca to {c['hero']}, detektyw; wynajął go {c['client']}. Sprawa: {c['case']}.",
        f"Kto gdzie jest: {c['where']}.",
        *[f"Wiesz: {k}. {v}" for k, v in c["facts"]],
        "Nie wiesz nic o: " + ", ".join(f"{t} (wie o tym {w})" for t, w in c["unknown"]) + ".",
        "Nie wymyślaj nazwisk, adresów i numerów. Nie handlujesz ani nie rozstrzygasz śledztwa; podajesz wyłącznie wypowiedź tej osoby.",
        "",
        HOW,
    ]
    return "\n".join(lines)


def common_world(c):
    n = c["neighbors"]
    return [
        f"Świat: {c['world']}. {c['world_desc']}",
        f"Osada, w której mieszkasz: {c['town']}. {c['town_desc']}",
        f"Dzień {c['day']}. Pora roku: {c['season']}, pogoda: {c['weather']}.",
        f"Twoi sąsiedzi w osadzie: {n[0][0]} ({n[0][1]}), {n[1][0]} ({n[1][1]}).",
    ]


def unknown(c):
    return "Nie wiesz nic o: " + ", ".join(f"{t} (wie o tym {w})" for t, w in c["unknown"]) + ". Jeśli gracz o to pyta, odeślij go do tej osoby."


def hero(c):
    h = c["hero"]
    return f"Twój rozmówca: {h['name']}, {h['health']}. Broń: {h['weapon']}. Pancerz: {h['armor']}. Opinia o nim w osadzie: {h['rep']}; traktuj go odpowiednio."


def gm_system(f, heroes, enemies):
    lines = [GM_HEAD, "", "STAN WALKI:", f"Świat: {f['world']}. Miejsce walki: {f['place']}", "Bohaterowie:"]
    for h in heroes:
        if h.get("lead"):
            lines.append(f"- {h['name']}, poziom {h['level']}, zdrowie {h['hp']}/{h['max']}, broń: {h['weapon']}, pancerz: {h['armor']}")
        else:
            lines.append(f"- {h['name']}, {h['role']} w drużynie {f['lead']}, zdrowie {h['hp']}/{h['max']}")
    lines.append("Przeciwnicy:")
    for e in enemies:
        lines.append(f"- {e['name']} ({e['kind']}), zdrowie {e['hp']}/{e['max']}")
    lines += ["", "", "", GM_HOW]
    return "\n".join(lines)


# ---------- [GRA] events ----------

TALK = {
    "shop": "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. Jeśli pyta o towar, możesz polecić coś ze swojego towaru i podać jego cenę z listy. W tej kwestii niczego nie sprzedajesz.",
    "merc": "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział: o swojej broni, rzemiośle, drodze albo świecie. O cenie służby mów tylko, gdy o nią pyta. W tej kwestii niczego nie sprzedajesz.",
    "comp": "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz ciekawie, w swojej roli, na to, co powiedział. Jesteś już w drużynie: nie proponujesz najmu ani zapłaty i nie opisujesz czynów, których nie było; tylko rozmawiasz.",
    "det": "Gracz nie składa oferty, tylko rozmawia albo o coś pyta. Odpowiedz w swojej roli na to, co powiedział. O sprawie mówisz tylko to, co masz w CO WIESZ; niczego nie dopowiadasz i niczym nie handlujesz.",
}


def mood(m):
    return f"Nastrój twojej postaci: {m}."


def talk(role, m="spokojna", extra=None):
    parts = [TALK[role], mood(m)]
    if extra:
        parts.append(extra)
    return " ".join(parts + [TAIL])


def greet(role, m="spokojna"):
    return talk(role, m, "Gracz tylko się wita. Odpowiedz jednym krótkim zdaniem: odwzajemnij powitanie po swojemu i zapytaj, czego chce. Nic więcej.")


def small(role, m="spokojna"):
    return talk(role, m, "Gracz zagaduje bez konkretnego pytania o świat. Odpowiedz krótko i po swojemu na to, co powiedział, i sam zagadnij go o jedno: po co przyszedł, dokąd idzie albo czy ma czym walczyć.")


def dunno(role, topic, who, m="spokojna"):
    return talk(role, m, f"Gracz pyta o coś, o czym nie wiesz: {topic} (wie o tym {who}). Powiedz, że nie wiesz, i odeślij go tam.")


def heard(role, fact, m="spokojna"):
    return talk(role, m, f"W karczmie usłyszałeś od sąsiada i możesz powtórzyć: {fact}")


def mentions(role, place, fact, m="spokojna"):
    return talk(role, m, f"Gracz wspomina o: {place}. Co o tym wiesz: {fact}")


def closed(m="spokojna"):
    return talk("shop", m, "Gracz chce handlować, ale jest wieczór i kram zamknięty: odmów grzecznie, zaproś rano, nic nie sprzedawaj.")


def g(text, m):
    return f"{text} {mood(m)} {TAIL}"


def unclear(m="spokojna"):
    return g("Nie jest jasne, co i za ile gracz chce kupić. Poproś, żeby powiedział wprost, o który przedmiot chodzi i ile oferuje. Nie podawaj żadnych kwot.", m)


def stock(items, cur, m="spokojna"):
    return g("Gracz pyta, co masz na sprzedaż. Na sprzedaż: " + ", ".join(f"«{n}» za {money(p, cur)}" for n, p in items) + ".", m)


def ask_item(item, desc, price, cur, m="spokojna"):
    return g(f"Gracz pyta o «{item}». Opis: {desc} Twoja aktualna cena: {money(price, cur)}. Opowiedz krótko o przedmiocie i podaj cenę.", m)


def poor(item, price, have, cur, m="spokojna"):
    return g(f"Gracz chce kupić «{item}» (twoja cena: {money(price, cur)}), ale ma przy sobie tylko {have}. Wytknij mu, że go nie stać. Niczego nie sprzedajesz.", m)


def counter(offer, item, new, cur, m="spokojna"):
    return g(f"Gracz zaproponował {money(offer, cur)} za «{item}». To za mało. Odrzuć ofertę i podaj swoją nową cenę: {money(new, cur)}.", m)


def insult(offer, item, price, cur, m="obrażona"):
    return g(f"Gracz zaproponował śmiesznie niską kwotę {money(offer, cur)} za «{item}». To cię uraża. Nie obniżasz ceny, nadal żądasz {money(price, cur)}.", m)


def buy(item, price, cur, m="zadowolona"):
    return g(f"Gracz kupuje «{item}» za {money(price, cur)}. Transakcja zawarta, potwierdź sprzedaż.", m)


def bulk(n, item, total, unit, cur, m="spokojna"):
    return g(f"Gracz kupuje {n} × «{item}» za {money(total, cur)}. Transakcja zawarta, potwierdź sprzedaż. Kwoty dotyczą wszystkich {n} sztuk razem; pojedyncza sztuka kosztuje u ciebie {money(unit, cur)}, hurtem jest taniej.", m)


def only_one(n, item, price, cur, m="spokojna"):
    return g(f"Gracz chce kupić {n} × «{item}», ale masz tylko 1 sztukę. Powiedz mu to. Za tę jedną chcesz {money(price, cur)}.", m)


def not_traded(item, m="spokojna"):
    return g(f"Gracz chce ci sprzedać «{item}», ale ty takimi rzeczami nie handlujesz. Odmów i powiedz, czym handlujesz. Nie podawaj żadnych kwot.", m)


def appraise(item, offer, cur, m="spokojna"):
    return g(f"Gracz chce ci sprzedać «{item}». Obejrzyj rzecz i zaoferuj za nią {money(offer, cur)}; więcej nie dasz.", m)


def sell_high(item, ask, offer, cur, m="poirytowana"):
    return g(f"Gracz chce ci sprzedać «{item}» i żąda {money(ask, cur)}. To za dużo. Odmów i zaoferuj swoją cenę: {money(offer, cur)}.", m)


def sold_to_npc(item, price, cur, m="zadowolona"):
    return g(f"Gracz sprzedaje ci «{item}» za {money(price, cur)}. Transakcja zawarta: płacisz i potwierdzasz.", m)


def bye(m="spokojna"):
    return g("Gracz się żegna. Pożegnaj go. Nie podawaj żadnych kwot.", m)


def fed_up(m="poirytowana"):
    return g("Masz dość tej rozmowy. Kończysz ją i odprawiasz gracza. Niczego mu już nie sprzedasz. Nie podawaj żadnych kwot.", m)


def hire_ok(price, cur, m="zadowolona"):
    return g(f"Gracz przyjmuje twoją cenę i płaci ci {money(price, cur)} za «Umowa najmu». Umowa stoi: potwierdź, że dołączasz do drużyny. Pieniądze bierzesz ty, nie dajesz.", m)


def hire_daily_ok(price, cur, m="spokojna"):
    return g(f"Gracz przyjmuje twoją stawkę «Żołd dzienny» {money(price, cur)} za każdy dzień, płatne o świcie; nic z góry. Umowa stoi: potwierdź, że dołączasz do drużyny. Teraz nie bierzesz pieniędzy.", m)


def ask_service(kind, price, cur, m="spokojna"):
    if kind == "daily":
        return ask_item("Żołd dzienny", "Służba w drużynie, rozliczana za każdy dzień o świcie; nic z góry.", price, cur, m)
    return ask_item("Umowa najmu", "Służba w drużynie, płatna raz przy dołączeniu.", price, cur, m)


def service_name(kind):
    return "Żołd dzienny" if kind == "daily" else "Umowa najmu"


# ---------- Game Master fights ----------

def fight(f, rounds):
    """f: world, place, lead, heroes, enemies (dicts with name, acc, hp, max).
    rounds: list of lists of events. Returns turns with the state before each round."""
    units = {u["name"]: u for u in f["heroes"] + f["enemies"]}
    turns = []
    for events in rounds:
        alive_enemies = [e for e in f["enemies"] if not e.get("gone")]
        system = gm_system(f, f["heroes"], alive_enemies)
        text = []
        for ev in events:
            kind = ev[0]
            if kind == "hit":
                a, b, n = units[ev[1]], units[ev[2]], ev[3]
                b["hp"] = max(0, b["hp"] - n)
                ob = "obrażenie" if n == 1 else ("obrażenia" if n % 10 in (2, 3, 4) and n % 100 not in (12, 13, 14) else "obrażeń")
                text.append(f"{a['name']} trafia {b['acc']} i zadaje {n} {ob}. {b['name']} traci {n} zdrowia; pozostaje {b['hp']}.")
                if b["hp"] == 0:
                    if b in f["enemies"]:
                        text.append(f"{b['name']} ginie.")
                        b["gone"] = True
                    else:
                        text.append(f"{b['name']} pada i nie może dalej działać w tej rundzie.")
            elif kind == "miss":
                text.append(f"{units[ev[1]]['name']} atakuje {units[ev[2]]['acc']}, lecz chybia; obrażenia: 0.")
            elif kind == "block":
                text.append(f"{units[ev[1]]['name']} atakuje {units[ev[2]]['acc']}, ale cel całkowicie blokuje ten atak; obrażenia: 0.")
            elif kind == "self":
                u = units[ev[1]]
                u["hp"] = min(u["max"], u["hp"] + ev[2])
                text.append(f"{u['name']} odzyskuje {ev[2]} zdrowia; ma teraz {u['hp']}.")
            elif kind == "heal":
                a, b = units[ev[1]], units[ev[2]]
                b["hp"] = min(b["max"], b["hp"] + ev[3])
                text.append(f"{a['name']} leczy {b['acc']}: {b['name']} odzyskuje {ev[3]} zdrowia i ma teraz {b['hp']}.")
            elif kind == "flee":
                u = units[ev[1]]
                text.append(f"{u['name']} skutecznie ucieka ze starcia; nie ginie.")
                if u in f["enemies"]:
                    u["gone"] = True
            elif kind == "idle":
                text.append(f"{units[ev[1]]['name']} nie wykonuje ataku w tej rundzie.")
            elif kind == "loot":
                text.append(f"Zdobyty łup: {ev[1]}.")
            elif kind == "win":
                text.append("Wszyscy przeciwnicy zostali pokonani; drużyna wygrywa starcie.")
            elif kind == "lost":
                text.append("W drużynie nie ma już nikogo zdolnego walczyć w tym starciu.")
        turns.append({"system": system, "gra": "Przebieg rundy: " + " ".join(text) + " " + TAIL})
    return turns
