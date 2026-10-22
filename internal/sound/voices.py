# Makes the intro's voice lines in internal/sound/clips: what the agent says
# as he winks, his own line for some agents, and the safe-for-work one.
#
# The voice is Kokoro-82M (Apache-2.0), run locally with kokoro-onnx; see
# packaging/claude-code/sounds/gen.py for setting it up. `make sounds` runs it.
import subprocess, sys, tempfile
from pathlib import Path

import soundfile as sf
from kokoro_onnx import Kokoro
from kokoro_onnx.config import EspeakConfig

out = Path(sys.argv[1])
models = Path(sys.argv[2]) if len(sys.argv) > 2 else Path('.')
tmp = Path(tempfile.mkdtemp())
k = Kokoro(str(models / 'kokoro-v1.0.onnx'), str(models / 'voices-v1.0.bin'),
           espeak_config=EspeakConfig(lib_path='/usr/lib64/libespeak-ng.so.1', data_path='/usr/share/espeak-ng-data'))
him = 0.7 * k.get_voice_style('am_puck') + 0.3 * k.get_voice_style('am_michael')

TRIM = ('silenceremove=start_periods=1:start_threshold=-48dB,areverse,'
        'silenceremove=start_periods=1:start_threshold=-48dB,areverse')
for name, text in [
    ('voice', 'Who got the good dee? Who got the good dee?'),
    ('voice_codex', 'Who got the good diff? Who got the good diff?'),
    ('voice_gemini', 'Who got the good dee? ...Twice.'),
    ('voice_cursor', 'Who got the good tab? Who got the good tab?'),
    ('voice_copilot', 'Who got the good autocomplete?'),
    ('voice_aider', 'Who got the good commit? Who got the good commit?'),
    ('voice_goose', 'Honk honk! Who got the good dee?'),
    ('voice_sfw', 'Who got the good code? Who got the good code?'),
]:
    samples, sr = k.create(text, voice=him, speed=0.9, lang='en-us')
    sf.write(tmp / f'{name}.wav', samples, sr)
    subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', str(tmp / f'{name}.wav'),
                    '-af', f'{TRIM},aecho=0.8:0.3:18:0.08,loudnorm=I=-16:TP=-1.5',
                    '-ac', '1', '-ar', '22050', '-c:a', 'pcm_s16le', str(out / f'{name}.wav')], check=True)
