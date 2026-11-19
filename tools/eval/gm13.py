# Dwie nowe walki dla Mistrza Gry, w formacie gry (prompt.go masterInstructions, master.go sceneLore).
import json, sys
PERSONA = ("Mistrz Gry prowadzący tę przygodę. Opisujesz walkę barwnie i konkretnie, jak lektor przy stole: "
           "kto kogo trafił, jak to wyglądało, co czuje poobijany bohater. Masz lekki, nieco złośliwy humor i słabość do dramatycznych pauz.")
TEMPLATE = """Jesteś Mistrzem Gry w tekstowej grze fabularnej. Zasady rozstrzygnęły już rundę walki; ty ją opisujesz graczom.

TWOJA POSTAĆ: %s

STAN WALKI:
%s

JAK OPISUJESZ:
- Po polsku, najwyżej 3 krótkie zdania, bez didaskaliów w gwiazdkach, bez nagłówków i bez list.
- Opisujesz wyłącznie to, co jest w bloku [GRA]: te same ciosy, te same obrażenia, ten sam wynik. Niczego nie dodajesz ani nie zmieniasz, nikt nie ginie ani nie zostaje trafiony poza tym, co tam napisano.
- Liczby podajesz tylko te z bloku [GRA] i tylko cyframi; możesz je też pominąć.
- Imiona bohaterów i nazwy przeciwników piszesz tak, jak w stanie walki. Bloku [GRA] nie cytujesz i o nim nie wspominasz."""
TAIL = " Odpowiedz najwyżej trzema krótkimi zdaniami. Liczby pisz cyframi."

def fight(fid, note, place, desc, hero, mates, foes, rounds):
    turns = []
    for events in rounds:
        lines = []
        for e in events:
            k = e[0]
            if k == "hit":     # kto, cel, obrażenia, uwaga
                _, a, t, d, *n = e; n = n[0] if n else ""
                if t in foes:
                    crit = " (trafienie krytyczne!)" if n == "crit" else ""
                    lines.append(f"{a} atakuje{crit}: {t} traci {d} zdrowia{' (' + n + ')' if n not in ('', 'crit') else ''}.")
                else:
                    lines.append(f"{a} atakuje: {t} traci {d} zdrowia.{' ' + n if n else ''}")
                hp(hero, mates, foes, t, -d)
            elif k == "miss": lines.append(f"{e[1]} atakuje: {e[2]}. Pudło.")
            elif k == "bolt": lines.append(f"{e[1]} rzuca: {e[2]}. {e[3]} traci {e[4]} zdrowia."); hp(hero, mates, foes, e[3], -e[4])
            elif k == "blast":
                lines.append(f"{e[1]} rzuca: Podmuch.")
                for t, d, saved in e[2]:
                    lines.append(("Rzut obronny udany, " if saved else "") + f"{t} traci {d} zdrowia."); hp(hero, mates, foes, t, -d)
            elif k == "heal":
                lines.append(f"{e[1]} rzuca: Opatrunek. Drużyna odzyskuje do {e[2]} zdrowia.")
                for who in [hero["name"]] + list(mates):
                    hp(hero, mates, foes, who, e[2])
            elif k == "potion": lines.append(f"{e[1]} używa: {e[2]} (+{e[3]} zdrowia)."); hp(hero, mates, foes, e[1], e[3])
            elif k == "venom": lines.append(f"Jad działa: {e[1]} traci {e[2]} zdrowia."); hp(hero, mates, foes, e[1], -e[2])
            elif k == "regen": lines.append(f"{e[1]} zrasta się: odzyskuje {e[2]} zdrowia."); hp(hero, mates, foes, e[1], e[2])
            elif k == "text": lines.append(e[1])
            elif k == "down": lines.append(f"{e[1]} pada nieprzytomny. Odpoczynek w osadzie postawi go na nogi.")
            elif k == "fall": lines.append(f"{e[1]} pada.")
        state = [f"Świat: Smocza Góra. Miejsce walki: {place}. {desc}", "Bohaterowie:",
                 f"- {hero['name']}, poziom {hero['level']}, zdrowie {max(hero['hp'],0)}/{hero['max']}, broń: {hero['weapon']}, pancerz: {hero['armor']}"]
        for name, m in mates.items():
            state.append(f"- {name}, {m['role']} w drużynie {hero['name']}, zdrowie {max(m['hp'],0)}/{m['max']}")
        state.append("Przeciwnicy:")
        for name, f in foes.items():
            state.append(f"- {name} ({f['shape']}), " + (f"zdrowie {f['hp']}/{f['max']}" if f['hp'] > 0 else "pokonany"))
        turns.append({"system": TEMPLATE % (PERSONA, "\n".join(state) + "\n"), "gra": "Przebieg rundy: " + " ".join(lines) + TAIL})
    return {"id": fid, "kind": "gm", "gm": True, "note": note, "turns": turns}

def hp(hero, mates, foes, who, delta):
    for pool in (mates, foes):
        if who in pool:
            pool[who]["hp"] = min(pool[who]["max"], pool[who]["hp"] + delta) if delta > 0 else pool[who]["hp"] + delta; return
    if who == hero["name"]:
        hero["hp"] = min(hero["max"], hero["hp"] + delta) if delta > 0 else hero["hp"] + delta

R, B, E, M, A = "Radomir", "Borin Kamienna Pięść", "Elira Cieniokrąg", "Siostra Mira", "Sir Aldric Smokobójca"
W, T = "Ogniste Widmo", "Troll Skalny"
f1 = fight("gm-kopalnia", "dwóch wrogów, Podmuch na obu, tarcza i widmo, towarzysz pada, zwycięstwo z łupem",
    "Kopalnia Durgara", "Opuszczone krasnoludzkie sztolnie, z których bije czerwona łuna i zapach siarki.",
    {"name": R, "level": 4, "hp": 50, "max": 58, "weapon": "Krasnoludzki Topór", "armor": "Kolczuga"},
    {B: {"role": "wojownik", "hp": 44, "max": 48}, E: {"role": "mag", "hp": 30, "max": 36}},
    {W: {"shape": "duch", "hp": 40, "max": 40}, T: {"shape": "humanoid", "hp": 55, "max": 55}},
    [[("text", f"{T} zbliża się."), ("hit", R, W, 4, "widmo"), ("blast", E, [(W, 10, False), (T, 5, True)]), ("hit", W, E, 9)],
     [("hit", B, T, 7, "tarcza"), ("miss", R, W), ("hit", T, B, 14, "Trafienie krytyczne!"), ("hit", W, E, 12)],
     [("blast", E, [(W, 10, False), (T, 10, False)]), ("hit", T, B, 18), ("miss", B, T)],
     [("hit", R, T, 9), ("hit", T, B, 15), ("down", B), ("miss", W, R)],
     [("blast", E, [(W, 10, False), (T, 10, False)]), ("hit", R, W, 6, "widmo"), ("fall", W), ("hit", T, R, 11)],
     [("hit", R, T, 14, "crit"), ("fall", T), ("text", "Zwycięstwo! Łup na głowę: 45 szt. złota, doświadczenie: 60."), ("text", "Na ziemi leży: Eliksir Jasności, Zwój Błyskawicy.")]])
Z, S, K = "Żmija Bagienna", "Olbrzymi Szczur", "Szkielet Wojownika"
f2 = fight("gm-mokradla", "trzech wrogów, jad, leczenie, nieudana ucieczka, krytyk, mikstura, uzdrowicielka pada",
    "Mokradła Czarnej Wody", "Bagna, w których tonie światło. Kładki zapadają się pod stopami.",
    {"name": R, "level": 2, "hp": 30, "max": 44, "weapon": "Krótki Miecz", "armor": "Skórzany Kaftan"},
    {M: {"role": "uzdrowiciel", "hp": 32, "max": 32}, A: {"role": "wojownik", "hp": 70, "max": 70}},
    {Z: {"shape": "wąż", "hp": 34, "max": 34}, S: {"shape": "bestia", "hp": 18, "max": 18}, K: {"shape": "nieumarły", "hp": 34, "max": 34}},
    [[("text", f"Czas na decyzję minął: {R} atakuje."), ("hit", R, S, 5), ("hit", Z, R, 7, "Jad działa!"), ("hit", A, K, 12), ("miss", K, A)],
     [("venom", R, 3), ("regen", K, 4), ("heal", M, 12), ("hit", S, M, 6), ("miss", R, Z)],
     [("text", f"{R} próbuje uciec, bez skutku."), ("text", f"{S} wyprowadza atak okazyjny na uciekającego."), ("hit", Z, R, 9), ("hit", A, S, 16, "crit"), ("fall", S)],
     [("potion", R, "Mikstura Leczenia", 15), ("hit", A, Z, 11), ("hit", K, M, 10), ("miss", Z, A)],
     [("hit", R, Z, 6), ("miss", M, K), ("hit", K, M, 9), ("hit", A, Z, 17), ("fall", Z)],
     [("regen", K, 4), ("hit", R, K, 7), ("hit", A, K, 14), ("hit", K, M, 8), ("down", M)],
     [("hit", R, K, 9), ("fall", K), ("text", "Zwycięstwo! Łup na głowę: 22 szt. złota, doświadczenie: 40."), ("text", "Na ziemi leży: Mikstura Leczenia.")]])
with open(sys.argv[1], "w", encoding="utf-8") as out:
    for f in (f1, f2):
        out.write(json.dumps(f, ensure_ascii=False) + "\n")
for f in (f1, f2):
    for t in f["turns"]:
        print(t["gra"][:300]); print("   ", t["system"].split("STAN WALKI:\n")[1].split("\n\nJAK")[0].replace("\n", " | ")[-260:])
