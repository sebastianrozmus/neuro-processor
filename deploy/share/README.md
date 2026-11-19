# neuro-share: zbieranie rozmów do ulepszania modelu

Mały serwer, do którego gra wysyła rozmowy, ale tylko gdy gracz się na to zgodził. Działa jako jeden proces bez bazy danych: każdy dzień to jeden plik JSON Lines w `/data`. Przed serwerem stoi Caddy, który obsługuje HTTPS z automatycznym certyfikatem.

## Uruchomienie

```sh
cd deploy/share
cp .env.example .env
# w .env: SHARE_DOMAIN (rekord A/AAAA na ten serwer)
docker compose up -d --build
curl https://$SHARE_DOMAIN/healthz         # ok
```

W grze trzeba ustawić `NEURO_SHARE_URL=https://twoja-domena` (w `.env.local` albo w zmiennych środowiskowych). Bez tej zmiennej pozycja „Udostępnij rozmowy” w ogóle się nie pojawia.

Wymagania: otwarte porty 80 i 443. Serwer zużywa poniżej 20 MB RAM i ma limit 128 MB oraz pół rdzenia. Caddy ma 256 MB, bo buforuje ciała zapytań. Na jednym katalogu danych może działać tylko jeden serwer, bo drugi proces nie dostanie blokady i nie wystartuje.

## API

| | |
|---|---|
| `GET /v1/challenge` | jednorazowe zadanie `{challenge, difficulty}`, ważne 10 minut |
| `POST /v1/submit` | koperta `{key, challenge, nonce, payload, sig}`; zwraca `201 {"ok":true,"stored":true}`, przy powtórce z ostatnich dwóch dni `409` (klient traktuje to jak sukces), a dla klucza, który wycofał zgodę, `403` |
| `POST /v1/forget` | od razu odrzuca klucz i zapisuje to na dysku, a jego wpisy usuwa w ciągu kilku sekund; zwraca `{"ok":true,"forgotten":true}` (koperta jak wyżej, `payload` = `{}`) |
| `GET /healthz` | `ok` |

Protokół i schemat danych opisuje `internal/share/share.go`. Klientem jest `share.Client`.

## Rzetelność danych

- **Podpis Ed25519.** Gra przy pierwszej wysyłce tworzy klucz instalacji w `share.key` w katalogu danych gry. Każda wysyłka jest nim podpisana. Dzięki temu serwer wie, że dane nie zmieniły się po drodze i z której instalacji przyszły, i potrafi usunąć wszystko z danej instalacji na jej prośbę. Słabe klucze (punkty małego rzędu i niekanoniczne zapisy), pod którymi podpis przechodzi bez żadnego sekretu, są odrzucane.
- **Jednorazowe zadanie (HMAC).** Zadanie podpisane kluczem serwera można zużyć tylko raz, więc przechwyconej wysyłki nie da się powtórzyć. Klucz jest losowany przy każdym starcie, dlatego po restarcie nie przejdzie nic, co było podpisane wcześniej.
- **Postać kanoniczna.** Serwer przyjmuje dane tylko wtedy, gdy są bajt w bajt tym, co zapisuje `json.Marshal`. Odpadają zduplikowane pola, inna wielkość liter, inna kolejność, brakujące pola i niepoprawne UTF-8. Dzięki temu podpisane bajty znaczą to samo dla każdego czytnika JSON, także dla Pythona przy treningu. Base64 też musi mieć jedną, kanoniczną postać.
- **Zgoda wycofana raz na zawsze.** Po `forget` klucz jest odrzucany na stałe, także po restarcie i także dla wysyłek podpisanych przed usunięciem. Wycofanie trafia najpierw na dysk (`forget.log`), a wpisy usuwa osobny proces sprzątający, który blokuje naraz tylko jeden plik. Przy każdym starcie sprzątacz robi pełny przebieg, więc usuwanie przerwane awarią zawsze się dokończy. Gra po wycofaniu zgody tworzy nowy klucz.
- **Dowód pracy.** Klient musi policzyć skrót SHA-256 z odpowiednią liczbą zer na początku (domyślnie 20 bitów, około 1 s na laptopie). Pojedynczy gracz tego nie zauważy, ale tysiąc fałszywych wysyłek kosztuje tysiąc sekund obliczeń.
- **Walidacja.** Serwer sprawdza wersję, zgodę, rodzaj, mówców, długości, poprawność UTF-8 i brak znaków sterujących, a nieznanych pól nie przyjmuje. Słabe wpisy dostają flagi, ale nie są odrzucane: `krotka`, `bez-gracza`, `bez-modelu`, `powtorzenia`, `nieznany-model`.
- **Łańcuch skrótów.** Każda linia pliku zawiera skrót linii poprzedniej, a także całą kopertę z podpisem. Polecenie `docker compose exec share /neuro-share verify` sprawdza podpisy, klucze, dowody pracy, łańcuchy, schemat i to, czy nie zostały wpisy wycofanych kluczy, więc pokazuje każdą zmianę albo usunięcie linii w środku pliku. Wypisuje też „głowy”, czyli liczbę wpisów i skrót ostatniej linii każdego pliku. Zapisane osobno, na przykład razem z kopią zapasową, pozwalają później wykryć obcięcie końca pliku albo zniknięcie całego dnia.
- **Zapis odporny na awarie.** Każda linia jest od razu zapisywana na dysk (fsync). Gdy zapis się przerwie, plik jest cofany do poprzedniej długości, a niedokończona linia po awarii jest obcinana przy starcie. Gdy serwer przy starcie znajdzie przerwany łańcuch, przestaje przyjmować zapisy, dopóki ktoś nie sprawdzi danych przez `verify`. Sprzątacz nie przepisze takiego pliku, więc ślad zostaje.

Czego to nie daje: dowodu, że rozmowę prowadził człowiek. Klient jest otwarty, a klucz należy do gracza. Dane są więc nienaruszone, przypisane do instalacji, możliwe do usunięcia i drogie do masowego podrabiania. Przed treningiem i tak trzeba je przejrzeć: flagi, duplikaty, liczba wpisów na klucz. Nie mogą też trafić do zestawów testowych t18, t20, t23 ani t25.

## Ochrona przed zalewaniem

- limit zapytań na adres IP (pule tokenów): 10 zadań i 5 zapisów na minutę, adresy IPv6 liczone po /64;
- limit na całą sieć (IPv4 /24, IPv6 /48), ośmiokrotnie wyższy, żeby właściciel wielu adresów nie mógł ich po kolei wykorzystywać;
- gdy tablica adresów się zapełni, nowe adresy dzielą jedną wspólną pulę, zamiast być wszystkie odrzucane;
- osobne pule równoległych zapytań dla zadań i zapisów (po 32); nadmiar dostaje od razu `503` zamiast czekać w kolejce;
- Caddy wczytuje całe ciało zapytania, zanim przekaże je dalej, więc wolny nadawca blokuje tani wątek Caddy, a nie miejsce na serwerze;
- limity dzienne: 200 wysyłek na klucz, 20 000 wysyłek i 200 MB łącznie, 5000 wycofań zgody, do tego 5 GB na dysku w sumie; po przekroczeniu serwer odpowiada „zajęty”; usunięcie zwalnia miejsce na dysku, ale nie dzienny limit;
- ciało zapytania do 128 KB (tak samo w Caddy), nagłówki do 8 KB, krótkie limity czasu na odczyt i zapis (chroni przed atakiem slowloris); Caddy przepuszcza tylko cztery ścieżki API;
- ograniczona pamięć: bez indeksu całego archiwum, liczniki tylko z bieżącego dnia, powtórki sprawdzane z ostatnich dwóch dni po 16-bajtowych skrótach;
- nagłówek `X-Forwarded-For` jest brany pod uwagę tylko od Caddy, pod stałym adresem w sieci Dockera (`SHARE_TRUSTED_PROXIES`); serwer nie wystawia żadnego portu na zewnątrz.

To wystarcza na spam i małe ataki. Atak wolumetryczny zapycha łącze, zanim ruch dotrze do jakiegokolwiek procesu, więc przed nim chroni tylko dostawca serwera (firewall lub ochrona DDoS) albo CDN. Przy Cloudflare z proxy trzeba w Caddy ustawić `trusted_proxies` na adresy Cloudflare i przekazywać `CF-Connecting-IP`, inaczej wszyscy gracze wyglądają jak kilka adresów Cloudflare.

## Świadome ograniczenia

Serwer ma zostać mały. Te rzeczy recenzja wskazała, ale wymagałyby bazy danych albo dodatkowej infrastruktury:
- `verify` nie porównuje stanu z uwierzytelnionym manifestem trzymanym poza wolumenem. Obcięcie końca pliku albo zniknięcie całego dnia widać tylko po porównaniu z „głowami” zapisanymi wcześniej. Pola `received` i `flags` oraz kolejność wpisów nie są podpisane przez serwer.
- Caddy nie ma globalnego limitu równoległych połączeń. Przed bardzo dużym zalewem chroni dopiero dostawca serwera albo CDN.
- Rejestr wycofań (`forget.log`) rośnie najwyżej o 5000 wpisów dziennie i jest w całości wczytywany do pamięci (16 bajtów na klucz).
- Dzienne limity po restarcie są liczone od nowa z zapisanych plików, więc wpisy usunięte tego dnia nie wchodzą do licznika.
- Cofnięcie zegara systemowego w trakcie działania może na chwilę przywrócić ważność zużytego zadania. Wysłanych danych to nie dubluje, bo serwer i tak odrzuca powtórki.

## Gra

W menu pauzy (Esc) jest pozycja „Udostępnij rozmowy”:
1. Za pierwszym razem gra pokazuje tekst zgody z przyciskami „Zgadzam się” i „Nie teraz”. Tekst wymienia każde wysyłane pole i mówi wprost, że modelu już wytrenowanego na rozmowach nie da się z nich wyczyścić. Nic się wtedy nie wysyła.
2. Po zgodzie są przyciski „Wyślij nowe rozmowy z tej gry (N kwestii)”, „Podejrzyj, co zostanie wysłane”, „Wycofaj zgodę i usuń moje rozmowy z serwera” i „Wróć”. Podgląd pokazuje całą paczkę, stronami (←/→). Wysyła się dokładnie to, co widać w podglądzie, i tylko po kliknięciu. Duża rozmowa idzie w kilku paczkach, od najstarszych kwestii.
3. Gdy wysyłka skończy się bez odpowiedzi serwera, następna próba wyśle tę samą paczkę bajt w bajt. Serwer rozpozna ją jako powtórkę, więc nic się nie zdubluje.
4. Wysyłka działa w tle (`tea.Cmd`), poza wątkiem interfejsu. Cała operacja ma limit 20 s, a każde zapytanie 8 s. Gra się nie zatrzymuje, można od razu wrócić do gry. Gdy serwer nie odpowiada albo jest zajęty, w dzienniku pojawia się jedna linia z `share.Message(err)`.
5. Klient nie podąża za przekierowaniami i wymaga HTTPS (poza localhost). Sukces uznaje tylko wtedy, gdy serwer go wprost potwierdzi w treści odpowiedzi. Tekst od serwera jest oczyszczany ze znaków sterujących i przycinany do 120 znaków.

Wycofanie zgody działa od razu, a potem gra prosi serwer o usunięcie danych (`Forget`). Do czasu potwierdzenia w `settings.json` zostaje znacznik `forget_pending`, a menu proponuje „Ponów usuwanie moich rozmów z serwera”, bez ponownej zgody na wysyłkę. Gdy serwer potwierdzi, gra kasuje klucz instalacji.

## Dane

- `/data/RRRR-MM-DD.jsonl` to wpisy: `received`, `key`, `challenge`, `nonce`, `sig`, `sha256`, `prev`, `flags`, `payload`. Adresów IP serwer nie zapisuje.
- `/data/forget.log` to prośby o usunięcie, zapisane tylko jako skrót SHA-256 klucza publicznego. Jest potrzebny, żeby wycofany klucz był odrzucany także po restarcie, i nie wolno go usuwać.
- `/data/.lock` to blokada katalogu działającego serwera.

Kopia zapasowa: `docker run --rm -v neuro-share_share-data:/d -v $PWD:/b alpine tar czf /b/share.tgz -C /d .`
