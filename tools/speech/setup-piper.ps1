# Prepares Piper, the light voice of the game: the program (MIT) and three
# Polish voices (gosia, darkman, mc_speech; CC0), about 200 MB in all, in
# the game's data directory. Piper runs on the processor and needs no
# graphics card. Afterwards pick it in the game: Esc -> Ustawienia -> Glos.
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$data = Join-Path $env:APPDATA "neuro-processor\piper"
$release = "https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_windows_amd64.zip"
$voices = "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/pl/pl_PL"

New-Item -ItemType Directory -Force (Join-Path $data "voices") | Out-Null

Write-Host "Pobieram piper..."
$zip = Join-Path $env:TEMP "piper_windows_amd64.zip"
Invoke-WebRequest $release -OutFile $zip
Expand-Archive $zip -DestinationPath $data -Force
Remove-Item $zip

foreach ($voice in "gosia", "darkman", "mc_speech") {
    Write-Host "Pobieram glos $voice..."
    foreach ($file in "pl_PL-$voice-medium.onnx", "pl_PL-$voice-medium.onnx.json") {
        Invoke-WebRequest "$voices/$voice/medium/$file" -OutFile (Join-Path $data "voices\$file")
    }
}

Write-Host "Gotowe. W grze: Esc -> Ustawienia -> Glos: Piper."
