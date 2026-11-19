"""Turn the frames of a recorded demo into a video.

Usage: python3 render.py FRAMES_DIR OUT.mp4|OUT.gif

FRAMES_DIR holds frame-NNNN.txt files of ANSI-coloured text (or
frame-NNNN.png screenshots) and frames.txt with one "file seconds" line per
frame. Each frame becomes a PNG (kept as NNNN.png, usable as a screenshot),
and ffmpeg joins them into an H.264 video or a GIF with its own palette.

Block characters (half blocks of the pixel art, eighths, quadrants, shades)
are drawn as exact rectangles, not with the font: a font's glyph never fills
the cell exactly, which left dark stripes between the rows of every picture.

NEURO_DEMO_FONT and NEURO_DEMO_FONT_BOLD pick the font; FFMPEG the ffmpeg binary.
"""
import os
import re
import subprocess
import sys
import unicodedata

from PIL import Image, ImageDraw, ImageFont

CW, CH = 9, 18
DEF_FG, DEF_BG = (220, 220, 220), (24, 24, 28)
FONTS = [  # (regular, bold, bold index in the same file)
    ("/System/Library/Fonts/Menlo.ttc", "/System/Library/Fonts/Menlo.ttc", 1),
    ("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", "/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf", 0),
    ("C:/Windows/Fonts/consola.ttf", "C:/Windows/Fonts/consolab.ttf", 0),
]


def fonts():
    if os.environ.get("NEURO_DEMO_FONT"):
        regular = os.environ["NEURO_DEMO_FONT"]
        return ImageFont.truetype(regular, 15), ImageFont.truetype(os.environ.get("NEURO_DEMO_FONT_BOLD", regular), 15)
    for regular, bold, index in FONTS:
        if os.path.exists(regular) and os.path.exists(bold):
            return ImageFont.truetype(regular, 15), ImageFont.truetype(bold, 15, index=index)
    sys.exit("brak fontu: ustaw NEURO_DEMO_FONT")


FONT, BOLD = fonts()
FFMPEG = os.environ.get("FFMPEG") or ("/opt/anaconda3/bin/ffmpeg" if os.path.exists("/opt/anaconda3/bin/ffmpeg") else "ffmpeg")

# Block elements U+2580..U+259F as filled parts of the cell: (x0, y0, x1, y1) in eighths.
BLOCKS = {"▀": [(0, 0, 8, 4)], "▔": [(0, 0, 8, 1)], "▕": [(7, 0, 8, 8)], "▐": [(4, 0, 8, 8)]}
BLOCKS.update({chr(0x2581 + i): [(0, 7 - i, 8, 8)] for i in range(8)})  # ▁..█ lower eighths
BLOCKS.update({chr(0x2589 + i): [(0, 0, 7 - i, 8)] for i in range(7)})  # ▉..▏ left eighths
QUADS = {"▖": "3", "▗": "4", "▘": "1", "▙": "134", "▚": "14", "▛": "123", "▜": "124", "▝": "2", "▞": "23", "▟": "234"}
QUAD = {"1": (0, 0, 4, 4), "2": (4, 0, 8, 4), "3": (0, 4, 4, 8), "4": (4, 4, 8, 8)}
BLOCKS.update({char: [QUAD[q] for q in parts] for char, parts in QUADS.items()})
SHADES = {"░": 0.25, "▒": 0.5, "▓": 0.75}


def block(draw, char, x, y, fg, bg):
    """Draws a block character exactly; returns False for any other character."""
    left, top = x * CW, y * CH
    if char in SHADES:
        a = SHADES[char]
        mixed = tuple(round(f * a + b * (1 - a)) for f, b in zip(fg, bg))
        draw.rectangle([left, top, left + CW - 1, top + CH - 1], fill=mixed)
        return True
    if char not in BLOCKS:
        return False
    for x0, y0, x1, y1 in BLOCKS[char]:
        # Rounded edges of neighbouring parts meet without a gap (half of 18 px is exactly 9).
        px0, px1 = left + round(x0 * CW / 8), left + round(x1 * CW / 8) - 1
        py0, py1 = top + round(y0 * CH / 8), top + round(y1 * CH / 8) - 1
        if px1 >= px0 and py1 >= py0:
            draw.rectangle([px0, py0, px1, py1], fill=fg)
    return True


def palette():
    base = [(0, 0, 0), (205, 0, 0), (0, 205, 0), (205, 205, 0), (0, 0, 238), (205, 0, 205), (0, 205, 205), (229, 229, 229),
            (127, 127, 127), (255, 0, 0), (0, 255, 0), (255, 255, 0), (92, 92, 255), (255, 0, 255), (0, 255, 255), (255, 255, 255)]
    levels = [0, 95, 135, 175, 215, 255]
    cube = [(levels[r], levels[g], levels[b]) for r in range(6) for g in range(6) for b in range(6)]
    greys = [(8 + 10 * i,) * 3 for i in range(24)]
    return base + cube + greys


PALETTE = palette()


def width(char):
    return 2 if unicodedata.east_asian_width(char) in ("W", "F") else 1


def render(src, dst):
    lines = open(src, encoding="utf-8").read().split("\n")
    token = re.compile(r"\x1b\[([0-9;]*)m|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|(.)", re.S)
    rows = []
    for line in lines:
        fg, bg, bold, reverse, faint, cells = DEF_FG, DEF_BG, False, False, False, []
        for m in token.finditer(line):
            if m.group(1) is not None:
                params = [int(p) if p else 0 for p in m.group(1).split(";")]
                i = 0
                while i < len(params):
                    p = params[i]
                    if p == 0:
                        fg, bg, bold, reverse, faint = DEF_FG, DEF_BG, False, False, False
                    elif p == 1:
                        bold = True
                    elif p == 2:
                        faint = True
                    elif p == 22:
                        bold = faint = False
                    elif p == 7:
                        reverse = True
                    elif p == 27:
                        reverse = False
                    elif 100 <= p <= 107:
                        bg = PALETTE[p - 100 + 8]
                    elif p == 39:
                        fg = DEF_FG
                    elif p == 49:
                        bg = DEF_BG
                    elif p in (38, 48) and i + 2 < len(params) and params[i + 1] == 5:
                        colour = PALETTE[params[i + 2]]
                        if p == 38:
                            fg = colour
                        else:
                            bg = colour
                        i += 2
                    elif p in (38, 48) and i + 4 < len(params) and params[i + 1] == 2:
                        colour = tuple(params[i + 2:i + 5])
                        if p == 38:
                            fg = colour
                        else:
                            bg = colour
                        i += 4
                    elif 30 <= p <= 37:
                        fg = PALETTE[p - 30]
                    elif 90 <= p <= 97:
                        fg = PALETTE[p - 90 + 8]
                    elif 40 <= p <= 47:
                        bg = PALETTE[p - 40]
                    i += 1
            elif m.group(2) is not None and m.group(2) not in "\r":
                f, b = (bg, fg) if reverse else (fg, bg)
                if faint:
                    f = tuple(round(c * 0.6 + d * 0.4) for c, d in zip(f, b))
                cells.append((m.group(2), f, b, bold))
        rows.append(cells)

    cols = max((sum(width(c[0]) for c in row) for row in rows), default=1)
    img = Image.new("RGB", (max(cols, 140) * CW, max(len(rows), 44) * CH), DEF_BG)
    draw = ImageDraw.Draw(img)
    for y, row in enumerate(rows):
        x = 0
        for char, fg, bg, bold in row:
            w = width(char)
            if bg != DEF_BG:
                draw.rectangle([x * CW, y * CH, (x + w) * CW - 1, (y + 1) * CH - 1], fill=bg)
            if char.strip() and not block(draw, char, x, y, fg, bg):
                draw.text((x * CW, y * CH), char, fill=fg, font=BOLD if bold else FONT)
            x += w
    img.save(dst)


CANVAS = (140 * CW, 44 * CH)


def picture(src, dst):
    """Fits a ready picture (a browser screenshot) onto the frame canvas."""
    img = Image.open(src).convert("RGB")
    if img.size != CANVAS:
        img.thumbnail(CANVAS)
        canvas = Image.new("RGB", CANVAS, DEF_BG)
        canvas.paste(img, ((CANVAS[0] - img.width) // 2, (CANVAS[1] - img.height) // 2))
        img = canvas
    img.save(dst)


def main():
    frames_dir, out = sys.argv[1], sys.argv[2]
    entries = [line.split() for line in open(os.path.join(frames_dir, "frames.txt"), encoding="utf-8") if line.strip()]
    concat = []
    for index, (name, seconds) in enumerate(entries):
        png = os.path.join(frames_dir, f"{index:04d}.png")
        if name.endswith(".png"):
            picture(os.path.join(frames_dir, name), png)
        else:
            render(os.path.join(frames_dir, name), png)
        # Names relative to concat.txt (ffmpeg resolves them from its folder), so any ffmpeg can read the list.
        concat.append(f"file '{index:04d}.png'\nduration {seconds}")
    concat.append(f"file '{index:04d}.png'")
    with open(os.path.join(frames_dir, "concat.txt"), "w", encoding="utf-8") as handle:
        handle.write("\n".join(concat) + "\n")
    source = [FFMPEG, "-y", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", os.path.join(frames_dir, "concat.txt")]
    if out.endswith(".gif"):
        # One palette for the whole demo, no dithering: flat terminal colours stay flat.
        graph = "fps=10,split[a][b];[a]palettegen=max_colors=256:stats_mode=full[p];[b][p]paletteuse=dither=none"
        subprocess.run(source + ["-filter_complex", graph, "-loop", "0", out], check=True)
    else:
        subprocess.run(source + ["-vf", "fps=12,format=yuv420p", "-c:v", "libx264", "-preset", "medium", "-crf", "20", out], check=True)
    print(f"{len(entries)} klatek -> {out}")


if __name__ == "__main__":
    main()
