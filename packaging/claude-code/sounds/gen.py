# Regenerates the mod's voice lines and sounds.
#
# The voices come from Kokoro-82M (Apache-2.0), run locally with kokoro-onnx;
# the creaks, knocks and clacks are synthesised. Needs espeak-ng and ffmpeg too.
#
#   uv venv -p 3.12 .venv && uv pip install -p .venv/bin/python kokoro-onnx soundfile
#   curl -LO https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/kokoro-v1.0.onnx
#   curl -LO https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/voices-v1.0.bin
#   .venv/bin/python packaging/claude-code/sounds/gen.py packaging/claude-code/sounds [name ...]
#
# With names, only those clips are made; the rest are left as they are.
import subprocess, sys, tempfile
from pathlib import Path

import numpy as np
import soundfile as sf
from kokoro_onnx import Kokoro
from kokoro_onnx.config import EspeakConfig

out = Path(sys.argv[1])
only = set(sys.argv[2:])
tmp = Path(tempfile.mkdtemp())
SR = 24000
k = Kokoro('kokoro-v1.0.onnx', 'voices-v1.0.bin',
           espeak_config=EspeakConfig(lib_path='/usr/lib64/libespeak-ng.so.1', data_path='/usr/share/espeak-ng-data'))

TRIM = ('silenceremove=start_periods=1:start_threshold=-48dB,areverse,'
        'silenceremove=start_periods=1:start_threshold=-48dB,areverse,apad=pad_dur=0.05')
DOOR = 'lowpass=f=1500,lowpass=f=2200,aecho=0.85:0.45:28|57:0.22|0.12'  # heard through a door
BEHIND_DOOR = f'{TRIM},{DOOR},loudnorm=I=-19:TP=-2'
TO_YOU = f'{TRIM},aecho=0.8:0.3:18:0.08,loudnorm=I=-16:TP=-1.5'  # said to you, outside the door
VAMPIRE = f'asetrate={SR}*0.93,aresample={SR},{TO_YOU}'  # a little deeper, a little slower


def wanted(name):
    return not only or name in only


def encode(src, name, filters):
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', str(src), '-af', filters,
                    '-ac', '1', '-ar', '22050', '-c:a', 'pcm_s16le', str(out / f'{name}.wav')], check=True)


def say(name, text, style, lang, speed, filters):
    if not wanted(name):
        return
    samples, sr = k.create(text, voice=style, speed=speed, lang=lang)
    sf.write(tmp / f'{name}.wav', samples, sr)
    encode(tmp / f'{name}.wav', name, filters)


def write(name, y, filters):
    if not wanted(name):
        return
    sf.write(tmp / f'{name}.wav', y / np.max(np.abs(y)) * 0.9, SR)
    encode(tmp / f'{name}.wav', name, filters)


her = 0.6 * k.get_voice_style('af_bella') + 0.4 * k.get_voice_style('af_nicole')
dora = k.get_voice_style('ef_dora')
puck = k.get_voice_style('am_puck')
him = 0.7 * puck + 0.3 * k.get_voice_style('am_michael')
for name, text, style, lang, speed, filters in [
    # Through the door, while Claude works.
    ('her_yeah', 'Ohh... yeah. Oh, yeahh!', her, 'en-us', 0.82, BEHIND_DOOR),
    ('her_dontstop', "Don't stop! Don't stop!", her, 'en-us', 0.9, BEHIND_DOOR),
    ('her_rightthere', 'Right there... right there!', her, 'en-us', 0.85, BEHIND_DOOR),
    ('her_omg', 'Oh my god... Claude!', her, 'en-us', 0.85, BEHIND_DOOR),
    ('her_yes', 'Yes! Yes! Yes!', her, 'en-us', 0.95, BEHIND_DOOR),
    ('her_papi', '¡Sí, papi! ¡Sí, papi!', dora, 'es', 0.9, BEHIND_DOOR),
    ('her_aypapi', '¡Ay, papi!', dora, 'es', 0.85, BEHIND_DOOR),
    ('her_mas', '¡Más, papi, más!', dora, 'es', 0.9, BEHIND_DOOR),
    ('him_gooddee_door', 'Who got the good dee?', puck, 'en-us', 0.95, BEHIND_DOOR),
    ('him_daddy', "Who's your daddy?", puck, 'en-us', 0.95, BEHIND_DOOR),
    # Halloween.
    ('her_count', 'Oh, Count Claude!', her, 'en-us', 0.85, BEHIND_DOOR),
    ('him_goodbite_door', 'Who got the good bite?', puck, 'en-us', 0.95, BEHIND_DOOR),
    # Safe for work.
    ('sfw_shipit', 'Ship it! Ship it!', her, 'en-us', 0.95, BEHIND_DOOR),
    ('sfw_lgtm', 'Looks good to me!', her, 'en-us', 0.9, BEHIND_DOOR),
    ('sfw_green', 'Yes! All green!', her, 'en-us', 0.9, BEHIND_DOOR),
    ('sfw_merge', 'Merge it!', her, 'en-us', 0.9, BEHIND_DOOR),
    ('him_senior', "Trust me, I'm a senior engineer.", puck, 'en-us', 0.95, BEHIND_DOOR),
    # To you: as he winks, slowly, and leaning out of the door for a yes.
    ('him_gooddee', 'Who got... the good... dee?', him, 'en-us', 0.68, TO_YOU),
    ('him_goodcode', 'Who got... the good... code?', him, 'en-us', 0.68, TO_YOU),
    ('him_goodbite', 'Who got... the good... bite?', him, 'en-us', 0.68, VAMPIRE),
    ('him_yespapi', 'Can I get a yes, papi?', him, 'en-us', 0.85, TO_YOU),
    ('him_canigetayes', 'Can I get a yes?', him, 'en-us', 0.85, TO_YOU),
]:
    say(name, text, style, lang, speed, filters)

rng = np.random.default_rng(7)


def resonate(x, f, q):
    # Two-pole resonator (biquad band-pass): one wooden mode.
    w = 2 * np.pi * f / SR
    a = np.sin(w) / (2 * q)
    b0, b2, a0, a1, a2 = a, -a, 1 + a, -2 * np.cos(w), 1 - a
    y = np.zeros_like(x)
    x1 = x2 = y1 = y2 = 0.0
    for i, v in enumerate(x):
        o = (b0 * v + b2 * x2 - a1 * y1 - a2 * y2) / a0
        x2, x1, y2, y1 = x1, v, y1, o
        y[i] = o
    return y


def creak(name, dur, f0, f1, modes):
    # Stick-slip friction: a train of tiny clicks whose rate glides as the
    # joint loads, each ringing the frame's wooden modes.
    n = int(dur * SR)
    t = np.arange(n) / SR
    rate = f0 + (f1 - f0) * (t / dur) ** 0.7
    phase = np.cumsum(rate * (1 + 0.12 * rng.standard_normal(n)) / SR)
    clicks = np.zeros(n)
    idx = np.nonzero(np.diff(np.floor(phase)) > 0)[0]
    clicks[idx] = 0.6 + 0.4 * rng.random(len(idx))
    m = int(0.004 * SR)
    burst = np.exp(-np.arange(m) / (0.0008 * SR)) * rng.standard_normal(m)
    exc = np.convolve(clicks, burst)[:n]
    y = sum(g * resonate(exc, f, q) for f, q, g in modes)
    y *= np.minimum(1, t / 0.03) * np.minimum(1, (dur - t) / 0.08) * (0.7 + 0.3 * np.sin(np.pi * t / dur))
    write(name, y, 'lowpass=f=2400,aecho=0.85:0.4:25:0.15,loudnorm=I=-27:TP=-4')


creak('creak1', 0.42, 150, 95, [(380, 9, 1.0), (760, 7, 0.6), (1450, 6, 0.3)])
creak('creak2', 0.36, 110, 165, [(410, 9, 1.0), (830, 7, 0.5), (1600, 6, 0.25)])

# The headboard against the wall: a low thud, wood on plaster, a little rattle.
n = int(0.45 * SR)
t = np.arange(n) / SR
thud = np.sin(2 * np.pi * (70 - 25 * t / 0.45) * t) * np.exp(-t / 0.07)
knock = resonate(rng.standard_normal(n) * np.exp(-t / 0.006), 220, 3) * 3
rattle = resonate(rng.standard_normal(n) * np.exp(-((t - 0.03) ** 2) / 0.0004) * (t > 0.02), 1900, 12) * 0.25
write('thump', thud + knock + rattle, 'lowpass=f=1000,loudnorm=I=-24:TP=-3')

# Our own rng from here on, so the clips above come out the same as before.
rng2 = np.random.default_rng(11)


def knuckle(n, at, strength):
    # One knuckle on a wooden door: a sharp tap that rings the panel.
    t = np.arange(n) / SR - at
    tap = rng2.standard_normal(n) * np.exp(-np.maximum(t, 0) / 0.0025) * (t >= 0)
    return strength * (resonate(tap, 180, 4) * 2.2 + resonate(tap, 420, 6) + resonate(tap, 1100, 8) * 0.35)


# Knock knock (knock): three on the door, the last a little softer.
n = int(0.75 * SR)
write('knock', knuckle(n, 0.02, 1.0) + knuckle(n, 0.21, 0.9) + knuckle(n, 0.40, 0.75),
      'aecho=0.8:0.35:22:0.12,loudnorm=I=-18:TP=-2')

# The door slammed: a heavy thud, the frame rattling, the room ringing after.
n = int(0.9 * SR)
t = np.arange(n) / SR
boom = np.sin(2 * np.pi * (58 - 20 * t) * t) * np.exp(-t / 0.09) * 1.4
crack = resonate(rng2.standard_normal(n) * np.exp(-t / 0.004), 260, 2.5) * 4
frame_rattle = resonate(rng2.standard_normal(n) * np.exp(-((t - 0.05) ** 2) / 0.002) * (t > 0.03), 2300, 10) * 0.35
write('slam', boom + crack + frame_rattle, 'aecho=0.8:0.5:40|83:0.3|0.15,loudnorm=I=-16:TP=-1.5')


def clacks(name, count, dur):
    # Someone typing fast through the door: a burst of keys, each a click as
    # it bottoms out and a softer thock as it comes back.
    n = int(dur * SR)
    y = np.zeros(n)
    at = 0.02
    for _ in range(count):
        t = np.arange(n) / SR - at
        on = t >= 0
        click = rng2.standard_normal(n) * np.exp(-np.maximum(t, 0) / 0.0012) * on
        thock = np.sin(2 * np.pi * 310 * t) * np.exp(-np.maximum(t, 0) / 0.012) * on
        y += (0.7 + 0.3 * rng2.random()) * (resonate(click, 3200, 3) * 1.5 + resonate(click, 1800, 4) + thock * 0.4)
        at += 0.045 + rng2.random() * 0.05
    write(name, y, f'{DOOR},loudnorm=I=-28:TP=-4')


clacks('clack1', 6, 0.42)
clacks('clack2', 5, 0.38)
