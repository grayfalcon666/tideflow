import sqlite3
import logging
import os

logger = logging.getLogger(__name__)

# In-memory cache for dictionary lookups
_dict_cache: dict = {}
_dict_loaded = False


def load_ecdict(db_path: str):
    """Load ECDICT SQLite database into in-memory cache."""
    global _dict_cache, _dict_loaded

    if _dict_loaded:
        return

    if not os.path.exists(db_path):
        logger.warning("ECDICT database not found at %s, dictionary lookup disabled", db_path)
        _dict_loaded = True
        return

    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        cursor.execute("SELECT word, phonetic, definition, translation, pos, exchange FROM stardict")
        for row in cursor.fetchall():
            word = row[0].lower()
            phonetic = row[1] or ""
            _dict_cache[word] = {
                "usphone": phonetic,
                "ukphone": phonetic,
                "definition": row[2] or "",
                "translation": row[3] or "",
                "pos": row[4] or "",
                "exchange": row[5] or "",
            }
        conn.close()
        logger.info("ECDICT loaded: %d words in cache", len(_dict_cache))
    except Exception as e:
        logger.error("Failed to load ECDICT: %s", e)
    finally:
        _dict_loaded = True


def lookup_word(word: str) -> dict:
    """
    Look up a word in the ECDICT dictionary.
    Returns dict with usphone, ukphone, definition, translation, pos, exchange.
    Returns empty dict if not found.
    """
    if not _dict_loaded:
        return {}

    result = _dict_cache.get(word.lower(), {})
    return result


def associate_captions(words: list, subtitles: list, max_per_word: int = 3) -> list:
    """
    Associate each word with up to max_per_word subtitle captions where the word appears.
    Returns list of word entries with associated captions.
    """
    result = []
    for entry in words:
        lemma = entry["lemma"]
        forms = entry.get("forms", [lemma])

        # Find captions containing any form of this word
        matched_captions = []
        for sub in subtitles:
            content = sub.get("content", "").lower()
            # Check if any form of the word appears in this caption
            for form in forms:
                if f" {form.lower()} " in f" {content} " or content.startswith(form.lower() + " ") or content.endswith(" " + form.lower()) or content == form.lower():
                    matched_captions.append({
                        "start": sub["start"],
                        "end": sub["end"],
                        "content": sub["content"],
                    })
                    break
            if len(matched_captions) >= max_per_word:
                break

        result.append({
            "value": lemma,
            "usphone": "",
            "ukphone": "",
            "definition": "",
            "translation": "",
            "pos": entry.get("pos", ""),
            "exchange": "",
            "captions": matched_captions,
        })

    return result


def enrich_wordbank(word_list: list) -> list:
    """
    Enrich word list with dictionary definitions and phonetics from ECDICT.
    Modifies the word list in place, filling in usphone, ukphone, definition, translation, exchange.
    """
    for entry in word_list:
        word = entry.get("value", "")
        if not word:
            continue

        result = lookup_word(word)
        if result:
            entry["usphone"] = result.get("usphone", "")
            entry["ukphone"] = result.get("ukphone", "")
            entry["definition"] = result.get("definition", "")
            entry["translation"] = result.get("translation", "")
            entry["exchange"] = result.get("exchange", "")
            if result.get("pos"):
                entry["pos"] = result["pos"]

    return word_list
