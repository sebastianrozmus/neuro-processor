# Wydania: jak gra trafia do ludzi

Gra jest czystym Go, bez cgo, więc jedna maszyna buduje ją na wszystkie systemy. Robi to [GoReleaser](https://goreleaser.com) w GitHub Actions.

## Jak wydać nową wersję

```sh
git tag v0.1.0
git push origin v0.1.0
```

Tag `v*` uruchamia workflow `.github/workflows/release.yml`. W ok. 2 minuty:

1. przechodzą testy (`go test ./...`);
2. GoReleaser buduje `neuro` dla macOS (Intel i Apple), Linuksa (amd64, arm64) i Windows (amd64), z wersją wpisaną w program (`neuro -version`);
3. pakuje każdy build z README, licencją i skryptami `tools/avatar/setup.sh` i `tools/speech/setup-tts.sh` (`.tar.gz`, na Windows `.zip`), liczy `checksums.txt`;
4. zakłada **szkic** wydania na GitHubie z listą zmian od poprzedniego tagu.

Szkic przeglądasz na stronie Releases, dopisujesz kilka zdań, co nowego, i klikasz „Publish”. Numeracja: `v0.x.y` do czasu, aż gra będzie stabilna; `y` dla poprawek, `x` dla nowych rzeczy.

Próba na sucho, lokalnie, bez publikacji (`brew install goreleaser`):

```sh
goreleaser release --snapshot --clean   # paczki w dist/
```

## Model

Model rozmów nie jest częścią wydania gry, bo waży 7,9 GB. Leży na Hugging Face i Ollama pobiera go stamtąd (`ollama pull hf.co/…`). Nowy model publikuje `tools/hf/upload.sh` (opis w `tools/hf/README.md` to strona modelu). Gdy zmienia się format promptów gry, nowy model i nowe wydanie gry idą razem, a w opisie wydania jest napisane, którego modelu wymaga.

## Następne kroki (gdy przyjdą pierwsi użytkownicy)

- **Homebrew** (`brew install sebastianrozmus/tap/neuro`): repo `homebrew-tap` na GitHubie i sekcja `brews` w `.goreleaser.yaml` z tokenem w sekretach. Brew omija też problem z Gatekeeperem.
- **Scoop** na Windows: analogicznie sekcja `scoops`.
- **Podpis i notaryzacja na macOS**: pobrany przez przeglądarkę plik macOS zablokuje („nie można zweryfikować dewelopera”). Do tego czasu: `xattr -d com.apple.quarantine neuro`. Notaryzacja wymaga konta Apple Developer (99 USD rocznie); GoReleaser ją obsługuje (sekcja `notarize`).
- **Obraz Dockera** z serwerem SSH (`neuro -listen :2222`) do grania ze znajomymi na serwerze.
