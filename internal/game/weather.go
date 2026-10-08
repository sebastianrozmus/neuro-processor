package game

import "github.com/sebastianrozmus/neuro-processor/internal/world"

// The sky over the world: four seasons of daysPerSeason each, and a
// weather rolled every morning that changes how far one sees, how fast
// one walks and what the roads hold.

// Weather is the sky of a day.
type Weather string

const (
	WeatherClear Weather = "pogodnie"
	WeatherFog   Weather = "mgła"
	WeatherRain  Weather = "deszcz"
	WeatherHeat  Weather = "upał"
	WeatherSnow  Weather = "śnieżyca"
)

const daysPerSeason = 10

// Seasons names the seasons in order.
var Seasons = []string{"wiosna", "lato", "jesień", "zima"}

// weatherOdds are the chances of each weather in each season, in percent.
var weatherOdds = [][]struct {
	weather Weather
	percent int
}{
	{{WeatherClear, 50}, {WeatherFog, 25}, {WeatherRain, 25}},
	{{WeatherClear, 50}, {WeatherHeat, 35}, {WeatherRain, 15}},
	{{WeatherClear, 35}, {WeatherFog, 35}, {WeatherRain, 30}},
	{{WeatherClear, 40}, {WeatherSnow, 40}, {WeatherFog, 20}},
}

// winterMarkup is where prices settle in winter: everything costs more
// when the roads are shut.
const winterMarkup = 115

// seasonOf is the season a day falls in.
func seasonOf(day int) int {
	return (max(day, 1) - 1) / daysPerSeason % len(Seasons)
}

// season names the current season.
func (r *Realm) season() string {
	return Seasons[seasonOf(r.day)]
}

// winter reports whether it is winter.
func (r *Realm) winter() bool {
	return seasonOf(r.day) == len(Seasons)-1
}

// rollWeather picks the day's weather for the season.
func (r *Realm) rollWeather() {
	roll := r.rng.Intn(100)

	for _, odds := range weatherOdds[seasonOf(r.day)] {
		if roll < odds.percent {
			r.weather = odds.weather

			return
		}

		roll -= odds.percent
	}

	r.weather = WeatherClear
}

// sky words the season and the weather.
func (r *Realm) sky() string {
	return "Pora roku: " + r.season() + ", pogoda: " + string(r.weather) + "."
}

// slow reports whether the weather makes every step take two hours.
func (r *Realm) slow() bool {
	return r.weather == WeatherRain || r.weather == WeatherSnow
}

// blind reports whether the weather hides the land beyond one's own tile.
func (r *Realm) blind() bool {
	return r.weather == WeatherFog
}

// weatherNote is what a traveller notices about the weather on the road,
// once a day.
func (r *Realm) weatherNote() string {
	switch r.weather {
	case WeatherFog:
		return "Mgła: widać tylko to, na czym stoisz, a z bieli może wyjść wszystko."
	case WeatherRain:
		return "Deszcz: każdy krok zajmuje dwie godziny."
	case WeatherHeat:
		return "Upał: moc nie wraca w marszu."
	case WeatherSnow:
		return "Śnieżyca: każdy krok zajmuje dwie godziny, a nocą drogi są puste i groźne."
	}

	return ""
}

// seasonal dresses places for the season: in winter the open country
// lies under snow.
func (r *Realm) seasonal(places []world.PlaceSpec) []world.PlaceSpec {
	if !r.winter() {
		return places
	}

	dressed := append([]world.PlaceSpec(nil), places...)
	for index := range dressed {
		switch dressed[index].Terrain {
		case "plains", "forest", "swamp":
			dressed[index].Terrain = "snow"
		}
	}

	return dressed
}
