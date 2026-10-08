"""Draw a hero portrait with a small local image model and turn it into the
game's pixel-art format: 40x40 pixels of the 256-colour terminal palette,
written as 20 rows of half-block characters.

Usage: python3 generate.py --prompt TEXT --out FILE.txt [--big FILE.txt] [--png FILE.png] [--seed N]

--out gets the small portrait the hero panel shows (20x20 pixels), --big the
large one the character screen shows (40x40 pixels).

The first run downloads the model (stabilityai/sd-turbo, about 2.5 GB) into
the Hugging Face cache.
"""
import argparse
import colorsys
import sys

SIZE = 40
MODEL = "stabilityai/sd-turbo"
STYLE = ("retro video game character portrait of {prompt}, head and shoulders, centered, facing the viewer, "
         "bold saturated colors, strong outlines, plain solid dark background, dramatic lighting, detailed face")
STEPS = 4


def palette():
    """The xterm 256-colour palette as RGB triples."""
    base = [(0, 0, 0), (128, 0, 0), (0, 128, 0), (128, 128, 0), (0, 0, 128), (128, 0, 128), (0, 128, 128), (192, 192, 192),
            (128, 128, 128), (255, 0, 0), (0, 255, 0), (255, 255, 0), (0, 0, 255), (255, 0, 255), (0, 255, 255), (255, 255, 255)]
    steps = [0, 95, 135, 175, 215, 255]
    cube = [(steps[i // 36], steps[i // 6 % 6], steps[i % 6]) for i in range(216)]
    grey = [(8 + i * 10,) * 3 for i in range(24)]
    return base + cube + grey


PALETTE = palette()
CUBE, ALL = list(range(16, 232)), list(range(16, 256))
CACHE = {}


def nearest(rgb):
    """The closest palette index; saturated colours never fall onto the grey ramp."""
    if rgb not in CACHE:
        h, s, v = colorsys.rgb_to_hsv(*(c / 255 for c in rgb))
        pool = CUBE if s > 0.22 and v > 0.14 else ALL
        CACHE[rgb] = min(pool, key=lambda n: sum((a - b) ** 2 for a, b in zip(PALETTE[n], rgb)))
    return CACHE[rgb]


def draw(prompt, seed):
    import torch
    from diffusers import AutoPipelineForText2Image

    device = "mps" if torch.backends.mps.is_available() else "cpu"
    pipe = AutoPipelineForText2Image.from_pretrained(MODEL, torch_dtype=torch.float16 if device == "mps" else torch.float32, variant="fp16")
    pipe = pipe.to(device)
    pipe.set_progress_bar_config(disable=True)
    generator = torch.Generator(device="cpu").manual_seed(seed)
    return pipe(STYLE.format(prompt=prompt), num_inference_steps=STEPS, guidance_scale=0.0, width=512, height=512, generator=generator).images[0]


def pixelate(image, size):
    from PIL import Image, ImageEnhance

    side = min(image.size)
    left, top = (image.width - side) // 2, (image.height - side) // 2
    image = image.crop((left, top, left + side, top + side)).convert("RGB")
    image = ImageEnhance.Color(image).enhance(1.4)
    image = ImageEnhance.Contrast(image).enhance(1.2)
    return image.resize((size, size), Image.LANCZOS)


def write(small, out, png):
    size = small.width
    pixels = [[nearest(small.getpixel((x, y))) for x in range(size)] for y in range(size)]
    lines = []
    for y in range(0, size, 2):
        lines.append("".join(f"\x1b[38;5;{pixels[y][x]};48;5;{pixels[y + 1][x]}m▀\x1b[0m" for x in range(size)))
    with open(out, "w", encoding="utf-8") as handle:
        handle.write("\n".join(lines) + "\n")
    if png:
        from PIL import Image
        preview = Image.new("RGB", (size, size))
        for y in range(size):
            for x in range(size):
                preview.putpixel((x, y), PALETTE[pixels[y][x]])
        preview.resize((size * 8, size * 8), Image.NEAREST).save(png)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--prompt", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--big")
    parser.add_argument("--png")
    parser.add_argument("--seed", type=int, default=0)
    args = parser.parse_args()

    image = draw(args.prompt, args.seed)
    write(pixelate(image, SIZE // 2), args.out, None)
    if args.big:
        write(pixelate(image, SIZE), args.big, args.png)
    print(args.out)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:  # noqa: BLE001 - the game shows this message to the player
        print(f"generator: {error}", file=sys.stderr)
        sys.exit(1)
