import json
import os
import subprocess
import tempfile
import logging

logger = logging.getLogger(__name__)


def check_embedded_subtitles(video_path: str) -> bool:
    """
    Use ffprobe to check if the video has embedded English subtitle streams.
    Returns True if English subtitles are found.
    """
    try:
        result = subprocess.run(
            [
                "ffprobe",
                "-v", "quiet",
                "-show_entries", "stream=index:stream_tags=language",
                "-select_streams", "s",
                "-of", "json",
                video_path,
            ],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            return False

        data = json.loads(result.stdout)
        for stream in data.get("streams", []):
            tags = stream.get("tags", {})
            lang = tags.get("language", "").lower()
            if lang in ("eng", "en", "english"):
                return True
        return False
    except Exception as e:
        logger.warning(f"ffprobe subtitle check failed: {e}")
        return False


def extract_embedded_subtitles(video_path: str) -> list:
    """
    Extract embedded English subtitles using ffmpeg and parse to standard JSON array.
    Returns list of dicts: [{"start": "00:00:01,000", "end": "00:00:03,500", "content": "Hello"}]
    """
    with tempfile.NamedTemporaryFile(suffix=".srt", delete=False) as tmp:
        srt_path = tmp.name

    try:
        # Extract subtitles to SRT
        result = subprocess.run(
            [
                "ffmpeg",
                "-y",
                "-v", "quiet",
                "-i", video_path,
                "-map", "0:s:m:language:eng?",
                "-c:s", "srt",
                srt_path,
            ],
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode != 0 or not os.path.exists(srt_path):
            return []

        return _parse_srt(srt_path)
    finally:
        if os.path.exists(srt_path):
            os.unlink(srt_path)


def extract_audio(video_path: str) -> str:
    """
    Extract audio from video as 16kHz mono WAV using ffmpeg.
    Returns path to the temporary WAV file.
    """
    wav_path = tempfile.mktemp(suffix=".wav")
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-v", "quiet",
            "-i", video_path,
            "-ac", "1",
            "-ar", "16000",
            "-sample_fmt", "s16",
            wav_path,
        ],
        check=True,
        timeout=120,
    )
    return wav_path


def transcribe_audio(
    audio_path: str,
    model_name: str = "small",
    device: str = "cpu",
    compute_type: str = "int8",
) -> list:
    """
    Transcribe audio using faster-whisper and return subtitle JSON array.
    Returns list of dicts: [{"start": "00:00:01,000", "end": "00:00:03,500", "content": "Hello"}]
    """
    from faster_whisper import WhisperModel

    model = WhisperModel(model_name, device=device, compute_type=compute_type)
    segments, info = model.transcribe(audio_path, beam_size=5, language="en")

    logger.info(
        "Whisper transcription started: language=%s duration=%.1fs model=%s device=%s",
        info.language, info.duration, model_name, device,
    )

    subtitles = []
    for segment in segments:
        subtitles.append({
            "start": _format_timestamp(segment.start),
            "end": _format_timestamp(segment.end),
            "content": segment.text.strip(),
        })

    logger.info("Whisper transcription completed: %d segments", len(subtitles))
    return subtitles


def _format_timestamp(seconds: float) -> str:
    """Convert seconds to SRT timestamp format HH:MM:SS,mmm."""
    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)
    secs = int(seconds % 60)
    millis = int((seconds - int(seconds)) * 1000)
    return f"{hours:02d}:{minutes:02d}:{secs:02d},{millis:03d}"


def _parse_timestamp(ts: str) -> float:
    """Parse SRT timestamp HH:MM:SS,mmm to float seconds."""
    parts = ts.replace(",", ":").split(":")
    h, m, s, ms = int(parts[0]), int(parts[1]), int(parts[2]), int(parts[3])
    return h * 3600 + m * 60 + s + ms / 1000.0


def _parse_srt(srt_path: str) -> list:
    """Parse SRT file to standard JSON subtitle array."""
    subtitles = []
    with open(srt_path, "r", encoding="utf-8", errors="replace") as f:
        content = f.read()

    blocks = content.strip().split("\n\n")
    for block in blocks:
        lines = block.strip().split("\n")
        if len(lines) < 3:
            continue
        # Skip index line, parse timestamp line
        time_line = lines[1]
        if "-->" not in time_line:
            continue
        start_str, end_str = time_line.split(" --> ")
        # Remove any trailing metadata after the timestamp
        end_str = end_str.split(" ")[0]
        text = "\n".join(lines[2:]).strip()
        if text:
            subtitles.append({
                "start": start_str.replace(".", ","),
                "end": end_str.replace(".", ","),
                "content": text,
            })

    return subtitles
