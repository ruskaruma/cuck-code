# Regenerates the mod's voice lines and bed sounds.
#
# The voices come from Kokoro-82M (Apache-2.0), run locally with kokoro-onnx;
# the creaks and the thump are synthesised. Needs espeak-ng and ffmpeg too.
#
#   uv venv -p 3.12 .venv && uv pip install -p .venv/bin/python kokoro-onnx soundfile
#   curl -LO https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/kokoro-v1.0.onnx
#   curl -LO https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0/voices-v1.0.bin
#   .venv/bin/python packaging/claude-code/sounds/gen.py packaging/claude-code/sounds
import subprocess, sys, tempfile
from pathlib import Path

import numpy as np
import soundfile as sf
from kokoro_onnx import Kokoro
from kokoro_onnx.config import EspeakConfig

out = Path(sys.argv[1])
tmp = Path(tempfile.mkdtemp())
SR = 24000
k = Kokoro('kokoro-v1.0.onnx', 'voices-v1.0.bin',
           espeak_config=EspeakConfig(lib_path='/usr/lib64/libespeak-ng.so.1', data_path='/usr/share/espeak-ng-data'))

TRIM = ('silenceremove=start_periods=1:start_threshold=-48dB,areverse,'
        'silenceremove=start_periods=1:start_threshold=-48dB,areverse,apad=pad_dur=0.05')
DOOR = 'lowpass=f=1500,lowpass=f=2200,aecho=0.85:0.45:28|57:0.22|0.12'  # heard through a door


def encode(src, name, filters):
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', str(src), '-af', filters,
                    '-ac', '1', '-ar', '22050', '-c:a', 'pcm_s16le', str(out / f'{name}.wav')], check=True)


def say(name, text, style, lang, speed, filters):
    samples, sr = k.create(text, voice=style, speed=speed, lang=lang)
    sf.write(tmp / f'{name}.wav', samples, sr)
    encode(tmp / f'{name}.wav', name, filters)


her = 0.6 * k.get_voice_style('af_bella') + 0.4 * k.get_voice_style('af_nicole')
dora = k.get_voice_style('ef_dora')
puck = k.get_voice_style('am_puck')
him = 0.7 * puck + 0.3 * k.get_voice_style('am_michael')
behind_door = f'{TRIM},{DOOR},loudnorm=I=-19:TP=-2'
for name, text, style, lang, speed in [
    ('her_yeah', 'Ohh... yeah. Oh, yeahh!', her, 'en-us', 0.82),
    ('her_dontstop', "Don't stop! Don't stop!", her, 'en-us', 0.9),
    ('her_rightthere', 'Right there... right there!', her, 'en-us', 0.85),
    ('her_omg', 'Oh my god... Claude!', her, 'en-us', 0.85),
    ('her_yes', 'Yes! Yes! Yes!', her, 'en-us', 0.95),
    ('her_papi', '¡Sí, papi! ¡Sí, papi!', dora, 'es', 0.9),
    ('her_aypapi', '¡Ay, papi!', dora, 'es', 0.85),
    ('her_mas', '¡Más, papi, más!', dora, 'es', 0.9),
    ('him_gooddee_door', 'Who got the good dee?', puck, 'en-us', 0.95),
    ('him_daddy', "Who's your daddy?", puck, 'en-us', 0.95),
]:
    say(name, text, style, lang, speed, behind_door)
# Said to you, outside the door, slowly.
say('him_gooddee', 'Who got... the good... dee?', him, 'en-us', 0.68, f'{TRIM},aecho=0.8:0.3:18:0.08,loudnorm=I=-16:TP=-1.5')

rng = np.random.default_rng(7)


def resonate(x, f, q):
    # Two-pole resonator (biquad band-pass): one wooden mode of the bed frame.
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
    sf.write(tmp / f'{name}.wav', y / np.max(np.abs(y)) * 0.9, SR)
    encode(tmp / f'{name}.wav', name, 'lowpass=f=2400,aecho=0.85:0.4:25:0.15,loudnorm=I=-27:TP=-4')


creak('creak1', 0.42, 150, 95, [(380, 9, 1.0), (760, 7, 0.6), (1450, 6, 0.3)])
creak('creak2', 0.36, 110, 165, [(410, 9, 1.0), (830, 7, 0.5), (1600, 6, 0.25)])

# The headboard against the wall: a low thud, wood on plaster, a little rattle.
n = int(0.45 * SR)
t = np.arange(n) / SR
thud = np.sin(2 * np.pi * (70 - 25 * t / 0.45) * t) * np.exp(-t / 0.07)
knock = resonate(rng.standard_normal(n) * np.exp(-t / 0.006), 220, 3) * 3
rattle = resonate(rng.standard_normal(n) * np.exp(-((t - 0.03) ** 2) / 0.0004) * (t > 0.02), 1900, 12) * 0.25
y = thud + knock + rattle
sf.write(tmp / 'thump.wav', y / np.max(np.abs(y)) * 0.9, SR)
encode(tmp / 'thump.wav', 'thump', 'lowpass=f=1000,loudnorm=I=-24:TP=-3')
