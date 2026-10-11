"""Turn the frames of a recorded demo into a video.

Usage: python3 render.py FRAMES_DIR OUT.mp4

FRAMES_DIR holds frame-NNNN.txt files of ANSI-coloured text (or
frame-NNNN.png screenshots) and frames.txt with one "file seconds" line per
frame. Each frame becomes a PNG, and ffmpeg joins them into an H.264 video.
"""
import os
import re
import subprocess
import sys
import unicodedata

from PIL import Image, ImageDraw, ImageFont

CW, CH = 9, 18
FONT = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 15)
BOLD = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 15, index=1)
DEF_FG, DEF_BG = (220, 220, 220), (24, 24, 28)
FFMPEG = "/opt/anaconda3/bin/ffmpeg" if os.path.exists("/opt/anaconda3/bin/ffmpeg") else "ffmpeg"


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
        fg, bg, bold, cells = DEF_FG, DEF_BG, False, []
        for m in token.finditer(line):
            if m.group(1) is not None:
                params = [int(p) if p else 0 for p in m.group(1).split(";")]
                i = 0
                while i < len(params):
                    p = params[i]
                    if p == 0:
                        fg, bg, bold = DEF_FG, DEF_BG, False
                    elif p == 1:
                        bold = True
                    elif p == 22:
                        bold = False
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
                    elif 30 <= p <= 37:
                        fg = PALETTE[p - 30]
                    elif 90 <= p <= 97:
                        fg = PALETTE[p - 90 + 8]
                    elif 40 <= p <= 47:
                        bg = PALETTE[p - 40]
                    i += 1
            elif m.group(2) is not None and m.group(2) not in "\r":
                cells.append((m.group(2), fg, bg, bold))
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
            if char.strip():
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
        concat.append(f"file '{png}'\nduration {seconds}")
    concat.append(f"file '{png}'")
    with open(os.path.join(frames_dir, "concat.txt"), "w", encoding="utf-8") as handle:
        handle.write("\n".join(concat) + "\n")
    subprocess.run([FFMPEG, "-y", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", os.path.join(frames_dir, "concat.txt"),
                    "-vf", "fps=12,format=yuv420p", "-c:v", "libx264", "-preset", "medium", "-crf", "20", out], check=True)
    print(f"{len(entries)} klatek -> {out}")


if __name__ == "__main__":
    main()
