import logging

logger = logging.getLogger(__name__)


def tokenize_and_lemmatize(text: str) -> list:
    """
    Tokenize text, perform POS tagging, and lemmatize each word.
    Returns list of dicts: [{"word": "abandon", "lemma": "abandon", "pos": "VB"}]
    """
    try:
        import nltk
        from nltk.tokenize import word_tokenize
        from nltk.corpus import wordnet
        from nltk.stem import WordNetLemmatizer
    except LookupError:
        logger.warning("NLTK data not found, downloading...")
        nltk.download("punkt_tab", quiet=True)
        nltk.download("averaged_perceptron_tagger_eng", quiet=True)
        nltk.download("wordnet", quiet=True)
        from nltk.tokenize import word_tokenize
        from nltk.corpus import wordnet
        from nltk.stem import WordNetLemmatizer

    lemmatizer = WordNetLemmatizer()

    # Tokenize
    tokens = word_tokenize(text)

    # POS tag
    tagged = nltk.pos_tag(tokens)

    results = []
    seen = set()

    for word, tag in tagged:
        # Skip non-word tokens
        if not word.isalpha():
            continue
        # Skip short words
        if len(word) < 2:
            continue

        word_lower = word.lower()
        if word_lower in seen:
            continue
        seen.add(word_lower)

        # Convert Penn Treebank tag to WordNet POS
        wn_pos = _get_wordnet_pos(tag)

        if wn_pos:
            lemma = lemmatizer.lemmatize(word_lower, pos=wn_pos)
        else:
            lemma = lemmatizer.lemmatize(word_lower)

        results.append({
            "word": word_lower,
            "lemma": lemma,
            "pos": _simplify_pos(tag),
        })

    return results


def build_word_list(subtitles: list) -> list:
    """
    From subtitle JSON array, tokenize each caption, lemmatize,
    deduplicate by lemma, and return list of unique word entries.
    Returns list of dicts: [{"lemma": "abandon", "pos": "v.", "forms": ["abandon", "abandons"]}]
    """
    all_tokens = []
    for sub in subtitles:
        tokens = tokenize_and_lemmatize(sub.get("content", ""))
        all_tokens.extend(tokens)

    # Merge by lemma (keep the most common form)
    merged = {}
    for t in all_tokens:
        lemma = t["lemma"]
        word = t["word"]
        pos = t["pos"]
        if lemma not in merged:
            merged[lemma] = {
                "lemma": lemma,
                "pos": pos,
                "forms": {word},
            }
        else:
            merged[lemma]["forms"].add(word)

    result = []
    for lemma, data in merged.items():
        result.append({
            "lemma": data["lemma"],
            "pos": data["pos"],
            "forms": sorted(data["forms"]),
        })

    return result


def _get_wordnet_pos(treebank_tag: str) -> str:
    """Map Penn Treebank POS tag to WordNet POS tag."""
    if treebank_tag.startswith("J"):
        return "a"  # ADJ
    elif treebank_tag.startswith("V"):
        return "v"  # VERB
    elif treebank_tag.startswith("N"):
        return "n"  # NOUN
    elif treebank_tag.startswith("R"):
        return "r"  # ADV
    return ""


def _simplify_pos(treebank_tag: str) -> str:
    """Simplify Penn Treebank POS tag to a readable form."""
    mapping = {
        "CC": "conj.", "CD": "num.", "DT": "det.", "EX": "pron.",
        "FW": "foreign", "IN": "prep.", "JJ": "adj.", "JJR": "adj.",
        "JJS": "adj.", "MD": "modal", "NN": "n.", "NNS": "n.",
        "NNP": "n.", "NNPS": "n.", "PDT": "det.", "PRP": "pron.",
        "PRP$": "pron.", "RB": "adv.", "RBR": "adv.", "RBS": "adv.",
        "RP": "particle", "TO": "to", "UH": "interj.",
        "VB": "v.", "VBD": "v.", "VBG": "v.", "VBN": "v.",
        "VBP": "v.", "VBZ": "v.", "WDT": "det.", "WP": "pron.",
        "WP$": "pron.", "WRB": "adv.",
    }
    return mapping.get(treebank_tag, "other")
