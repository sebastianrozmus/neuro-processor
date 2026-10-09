"""Draw a picture with a small local image model and turn it into the
game's pixel-art format: a palette of a few dozen colours and one index per
pixel, stored in the PX2 layout that internal/pix reads.

Usage: python3 generate.py --prompt TEXT --out FILE.px [--big FILE.px] [--png FILE.png]
                           [--seed N] [--kind portrait|scene|backdrop]
                           [--cells WxH] [--cell WxH]
       python3 generate.py --jobs LIST.json [--cell WxH]

--kind picks the style of the picture: a hero portrait (the default), the
view of a place or a title picture for the menu. --cells is the size of the
picture in character cells, --cell how many pixels fill one cell (1x2 for
half blocks, 2x2 for quadrants, 2x3 for sextants). A portrait writes --out
in half the cells (the hero panel) and --big in the full size (the
character screen); the other kinds write --out alone. --jobs draws a whole
list of pictures ({"kind", "prompt", "out", "cells", "seed"} each) with the
model loaded once, which is how a world gets its scenes.

The first run downloads the model (stabilityai/sd-turbo, about 2.5 GB) into
the Hugging Face cache.
"""
import argparse
import json
import os
import struct
import sys

MODEL = "stabilityai/sd-turbo"
STEPS = 4

STYLES = {
    "portrait": ("retro video game character portrait of {prompt}, head and shoulders, centered, facing the viewer, "
                 "bold saturated colors, strong outlines, plain solid dark background, dramatic lighting, detailed face"),
    "scene": ("retro 16-bit pixel art video game background, {prompt}, wide landscape view, no people, no creatures, "
              "no text, bold colors, clean shapes"),
    "backdrop": ("retro 16-bit pixel art video game title screen key art, {prompt}, dramatic, wide view, no people, "
                 "no creatures, no text, bold colors, clean shapes"),
}
# Cells of each kind of picture and how many colours it is reduced to.
CELLS = {"portrait": (40, 20), "scene": (64, 16), "backdrop": (120, 40)}
COLOURS = {"portrait": 32, "scene": 48, "backdrop": 48}


def size(text):
    width, height = text.lower().split("x")
    return int(width), int(height)


def draw(prompt, seed):
    import torch
    from diffusers import AutoPipelineForText2Image

    device = "mps" if torch.backends.mps.is_available() else "cpu"
    pipe = AutoPipelineForText2Image.from_pretrained(MODEL, torch_dtype=torch.float16 if device == "mps" else torch.float32, variant="fp16")
    pipe = pipe.to(device)
    pipe.set_progress_bar_config(disable=True)
    generator = torch.Generator(device="cpu").manual_seed(seed)
    return pipe(prompt, num_inference_steps=STEPS, guidance_scale=0.0, width=512, height=512, generator=generator).images[0]


def pixelate(image, cells, cell, colours):
    """Crop the picture to the shape it will have on screen (a character
    cell is about twice as high as wide), shrink it to the pixel grid and
    reduce it to a handful of colours, so that the pixels read as drawn
    rather than as noise."""
    from PIL import Image, ImageEnhance

    width, height = cells[0] * cell[0], cells[1] * cell[1]
    aspect = cells[0] / (cells[1] * 2)
    band_width, band_height = image.width, image.height
    if aspect > band_width / band_height:
        band_height = round(band_width / aspect)
    else:
        band_width = round(band_height * aspect)
    left, top = (image.width - band_width) // 2, (image.height - band_height) // 2
    image = image.crop((left, top, left + band_width, top + band_height)).convert("RGB")
    image = ImageEnhance.Color(image).enhance(1.4)
    image = ImageEnhance.Contrast(image).enhance(1.2)
    image = image.resize((width, height), Image.LANCZOS)
    return image.quantize(colors=colours, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)


def write(small, cell, out, png):
    """Store a quantized picture in the PX2 layout. Index 0 is transparent
    in the game, so every palette entry moves up by one."""
    width, height = small.size
    flat = small.getpalette()
    entries = min(len(flat) // 3, 255 - 1)
    palette = b"\0\0\0" + bytes(flat[: entries * 3])
    pixels = bytes(min(index + 1, 255) for index in small.getdata())
    header = b"PX2" + struct.pack(">BBHHB", cell[0], cell[1], width, height, entries + 1)
    # The game picks files up as soon as they exist, so each one appears whole.
    with open(out + ".part", "wb") as handle:
        handle.write(header + palette + pixels)
    os.replace(out + ".part", out)
    if png:
        from PIL import Image
        small.convert("RGB").resize((width * 8, height * 8), Image.NEAREST).save(png)


def paint_jobs(path, cell):
    """Draw a list of pictures from a JSON file, loading the model once:
    [{"kind": ..., "prompt": ..., "out": ..., "cells": [w, h], "seed": n}, ...]."""
    with open(path, encoding="utf-8") as handle:
        jobs = json.load(handle)
    for job in jobs:
        kind = job.get("kind", "scene")
        cells = tuple(job.get("cells") or CELLS[kind])
        image = draw(STYLES[kind].format(prompt=job["prompt"]), int(job.get("seed", 0)))
        write(pixelate(image, cells, cell, COLOURS[kind]), cell, job["out"], None)
        print(job["out"], flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prompt")
    parser.add_argument("--out")
    parser.add_argument("--jobs", help="JSON list of pictures to draw instead of a single --prompt")
    parser.add_argument("--big")
    parser.add_argument("--png")
    parser.add_argument("--seed", type=int, default=0)
    parser.add_argument("--kind", choices=sorted(STYLES), default="portrait")
    parser.add_argument("--cells", type=size)
    parser.add_argument("--cell", type=size, default=(1, 2))
    args = parser.parse_args()

    if args.jobs:
        paint_jobs(args.jobs, args.cell)
        return
    if not args.prompt or not args.out:
        parser.error("--prompt and --out are required without --jobs")

    cells = args.cells or CELLS[args.kind]
    colours = COLOURS[args.kind]
    image = draw(STYLES[args.kind].format(prompt=args.prompt), args.seed)

    if args.kind == "portrait":
        write(pixelate(image, (cells[0] // 2, cells[1] // 2), args.cell, colours), args.cell, args.out, None)
        if args.big:
            write(pixelate(image, cells, args.cell, colours), args.cell, args.big, args.png)
    else:
        write(pixelate(image, cells, args.cell, colours), args.cell, args.out, args.png)
    print(args.out)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:  # noqa: BLE001 - the game shows this message to the player
        print(f"generator: {error}", file=sys.stderr)
        sys.exit(1)
