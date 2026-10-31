#!/usr/bin/env python3
"""Generates the small media files embedded into WhatsDown's Demo Mode.

Usage: python3 internal/app/demo_assets/generate.py internal/app/demo_assets
Requires Pillow, ffmpeg (libx264, libopus) and espeak-ng.
"""
import math
import os
import random
import subprocess
import sys
import tempfile

from PIL import Image, ImageDraw, ImageFilter

OUT = sys.argv[1]
os.makedirs(OUT, exist_ok=True)
random.seed(3)


def landscape(w, h, sky_top, sky_bottom, ridges, sun=None):
    img = Image.new("RGB", (w, h))
    d = ImageDraw.Draw(img)
    for y in range(h):
        t = y / h
        d.line([(0, y), (w, y)], fill=tuple(int(a + (b - a) * t) for a, b in zip(sky_top, sky_bottom)))
    if sun:
        sx, sy, r, col = sun
        glow = Image.new("L", (w, h), 0)
        ImageDraw.Draw(glow).ellipse([sx - r * 3, sy - r * 3, sx + r * 3, sy + r * 3], fill=90)
        glow = glow.filter(ImageFilter.GaussianBlur(r))
        img.paste(Image.new("RGB", (w, h), col), (0, 0), glow)
        ImageDraw.Draw(img).ellipse([sx - r, sy - r, sx + r, sy + r], fill=col)
    d = ImageDraw.Draw(img)
    for base, amp, colour, seed in ridges:
        rnd = random.Random(seed)
        waves = [(rnd.uniform(0.002, 0.012), rnd.uniform(0, 6.28), rnd.uniform(0.3, 1.0)) for _ in range(5)]
        pts = [(0, h)]
        for x in range(0, w + 4, 4):
            v = sum(a * math.sin(x * f + p) for f, p, a in waves) / 2.5
            pts.append((x, base * h - v * amp * h))
        pts.append((w, h))
        d.polygon(pts, fill=colour)
    return img


summit = landscape(1280, 853, (40, 70, 140), (250, 170, 120), [
    (0.55, 0.18, (88, 92, 130), 1),
    (0.66, 0.16, (60, 66, 104), 2),
    (0.78, 0.12, (38, 46, 76), 3),
    (0.90, 0.08, (22, 30, 50), 4),
], sun=(900, 380, 46, (255, 226, 170)))
summit.save(os.path.join(OUT, "summit.jpg"), quality=78, optimize=True, progressive=True)

trail = landscape(1920, 1080, (120, 180, 230), (220, 236, 245), [
    (0.50, 0.20, (150, 170, 190), 11),
    (0.62, 0.16, (90, 130, 110), 12),
    (0.76, 0.12, (60, 105, 70), 13),
    (0.90, 0.08, (36, 72, 44), 14),
], sun=(420, 230, 60, (255, 250, 220)))

with tempfile.TemporaryDirectory() as tmp:
    src = os.path.join(tmp, "trail.png")
    trail.save(src)
    subprocess.run([
        "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-loop", "1", "-i", src,
        "-vf", "zoompan=z='1+0.0016*on':x='iw/2-(iw/zoom/2)+on*1.2':y='ih/2-(ih/zoom/2)':d=150:s=854x480:fps=25,format=yuv420p",
        "-t", "6", "-c:v", "libx264", "-preset", "veryslow", "-crf", "30", "-movflags", "+faststart",
        os.path.join(OUT, "trail.mp4"),
    ], check=True)

    wav = os.path.join(tmp, "voice.wav")
    subprocess.run([
        "espeak-ng", "-v", "en-us+f3", "-s", "155", "-w", wav,
        "Hey! Quick voice note about the release. The new media viewer is great, and I love "
        "being able to play voice messages at double speed. Try seeking through this one, "
        "or switch to one point five x. Talk soon!",
    ], check=True)
    subprocess.run([
        "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", wav, "-ac", "1", "-ar", "48000",
        "-c:a", "libopus", "-b:a", "20k", "-application", "voip", os.path.join(OUT, "voice.ogg"),
    ], check=True)


def pdf(path, pages):
    """Writes a tiny vector PDF with Helvetica text, one list of (size, text) lines per page."""
    objs = ["<< /Type /Catalog /Pages 2 0 R >>", None,
            "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
            "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>"]
    kids = []
    for n, lines in enumerate(pages, 1):
        ops = ["0.106 0.533 0.384 rg 0 772 612 20 re f"]
        y = 730
        for i, (size, bold, text) in enumerate(lines):
            if bold and i > 0:
                y -= 12
            text = text.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")
            ops.append(f"BT /{'F2' if bold else 'F1'} {size} Tf 0.12 0.12 0.14 rg 56 {y} Td ({text}) Tj ET")
            if i == 0:
                ops.append(f"0.106 0.533 0.384 rg 56 {y - 12} 500 1.5 re f")
                y -= 14
            y -= size + (14 if size > 14 else 9)
        ops.append(f"BT /F1 9 Tf 0.45 0.45 0.5 rg 56 40 Td (WhatsDown Release 3 - page {n} of {len(pages)}) Tj ET")
        stream = "\n".join(ops).encode("latin-1")
        objs.append(f"<< /Length {len(stream)} >>\nstream\n".encode("latin-1") + stream + b"\nendstream")
        objs.append(f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents {len(objs)} 0 R >>")
        kids.append(len(objs))
    objs[1] = f"<< /Type /Pages /Kids [{' '.join(f'{k} 0 R' for k in kids)}] /Count {len(kids)} >>"
    out = bytearray(b"%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
    offsets = []
    for i, o in enumerate(objs, 1):
        offsets.append(len(out))
        body = o if isinstance(o, bytes) else o.encode("latin-1")
        out += f"{i} 0 obj\n".encode() + body + b"\nendobj\n"
    xref = len(out)
    out += f"xref\n0 {len(objs) + 1}\n0000000000 65535 f \n".encode()
    out += b"".join(f"{o:010d} 00000 n \n".encode() for o in offsets)
    out += f"trailer\n<< /Size {len(objs) + 1} /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n".encode()
    with open(path, "wb") as f:
        f.write(out)


pdf(os.path.join(OUT, "release-3-notes.pdf"), [
    [(28, True, "WhatsDown Release 3"), (16, False, "Functional - release notes"),
     (12, True, "Messaging"),
     (11, False, "- Select messages, forward them to several chats at once, or delete them."),
     (11, False, "- Message Info shows delivery and read times for every recipient."),
     (11, False, "- Links in messages are clickable and days are separated in the history."),
     (11, False, "- Group chat previews show who sent the last message."),
     (12, True, "Organisation"),
     (11, False, "- Coloured chat folders with avatar clusters and drag-and-drop."),
     (11, False, "- Add to Folder from the chat context menu, or select several chats."),
     (11, False, "- Read All marks every chat as read in one click.")],
    [(22, True, "Media"),
     (11, False, "- In-window photo, video and PDF viewers."),
     (11, False, "- Zoom, rotate and pan photos; page through PDF documents."),
     (11, False, "- Reply, forward, locate in chat, or open in another application."),
     (11, False, "- Voice messages support seeking, pause and resume."),
     (11, False, "- Pitch-preserving playback at 0.5x, 1x, 1.5x and 2x."),
     (22, True, "Notifications"),
     (11, False, "- Sender avatars for direct messages, group avatars for groups."),
     (11, False, "- Clicking a notification opens the relevant chat.")],
    [(22, True, "Keyboard shortcuts"),
     (11, False, "- Navigate between chats without leaving the keyboard."),
     (11, False, "- Format selected text as bold, italic, strikethrough or code."),
     (11, False, "- Press Up to edit the last message you sent."),
     (11, False, "- Send files straight from the composer."),
     (11, False, "- Open the shortcut overview from the keyboard or from Settings."),
     (22, True, "Thank you!"),
     (11, False, "Try all of this right here in Demo Mode - no WhatsApp account required.")],
])

for name in sorted(os.listdir(OUT)):
    print(f"{name}: {os.path.getsize(os.path.join(OUT, name))} bytes")
