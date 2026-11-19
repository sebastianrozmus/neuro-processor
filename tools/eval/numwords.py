import json, re, sys, glob, statistics
from check import NUM_WORDS, sentences
for pat in sys.argv[1:]:
    files = sorted(glob.glob(pat))
    rows = [json.loads(l) for f in files for l in open(f, encoding="utf-8") if l.strip()]
    rep = [r["messages"][-1]["content"] for r in rows if "messages" in r]
    nw = sum(1 for x in rep if re.search(NUM_WORDS, x.lower()))
    four = sum(1 for x in rep if len(sentences(re.sub(r"^\[\w+\]\s*", "", x))) > 3)
    print(f"{pat}: {len(rep)} odp., liczby słownie {nw} ({100*nw/max(len(rep),1):.1f}%), >3 zdania {four}, śr. dł. {statistics.mean(len(x) for x in rep):.0f}")
