// Wspólne zachowanie stron przewodnika: dymki przy terminach, spis treści ze wskaźnikiem, podświetlenie terminu w słowniku.
(function () {
  const TERMS = window.TERMS || [];
  const byId = Object.fromEntries(TERMS.map(t => [t.id, t]));

  // --- dymki: <a class="t" href="slownik.html#sft">SFT</a>
  const tip = document.createElement("div");
  tip.className = "tip";
  tip.setAttribute("role", "tooltip");
  document.body.appendChild(tip);
  let current = null;

  function show(el) {
    const id = (el.getAttribute("href") || "").split("#")[1];
    const t = byId[id];
    if (!t) return;
    current = el;
    tip.innerHTML = "";
    const b = document.createElement("b"); b.textContent = t.name; tip.appendChild(b);
    tip.appendChild(document.createTextNode(t.short));
    const s = document.createElement("small"); s.textContent = "kliknij → pełny opis w słowniku"; tip.appendChild(s);
    const r = el.getBoundingClientRect();
    tip.style.left = "0px"; tip.style.top = "0px";
    tip.classList.add("show");
    const w = tip.offsetWidth, h = tip.offsetHeight;
    let x = Math.min(Math.max(8, r.left + r.width / 2 - w / 2), window.innerWidth - w - 8);
    let y = r.top - h - 10;
    if (y < 8) y = r.bottom + 10;
    tip.style.left = x + "px"; tip.style.top = y + "px";
  }
  function hide() { current = null; tip.classList.remove("show"); }

  document.addEventListener("mouseover", e => { const a = e.target.closest("a.t"); if (a && a !== current) show(a); else if (!a && current) hide(); });
  document.addEventListener("focusin", e => { const a = e.target.closest("a.t"); if (a) show(a); });
  document.addEventListener("focusout", hide);
  window.addEventListener("scroll", hide, { passive: true });

  // --- spis treści: <nav class="toc" data-auto> budowany z <section id data-toc="..."> (data-sub = podpunkt)
  const toc = document.querySelector(".toc[data-auto]");
  if (toc) {
    const secs = [...document.querySelectorAll("section[id][data-toc]")];
    const ol = document.createElement("ol");
    ol.innerHTML = '<span class="track"></span><span class="fill"></span><span class="marker"></span>';
    const links = secs.map(s => {
      const li = document.createElement("li");
      if (s.hasAttribute("data-sub")) li.className = "sub";
      const a = document.createElement("a");
      a.href = "#" + s.id; a.textContent = s.dataset.toc;
      li.appendChild(a); ol.appendChild(li);
      return a;
    });
    toc.appendChild(ol);
    const prog = document.createElement("div"); prog.className = "progress"; toc.appendChild(prog);
    const marker = ol.querySelector(".marker"), fill = ol.querySelector(".fill");
    let active = -1;

    function update() {
      const line = window.innerHeight * 0.3;
      let idx = 0;
      secs.forEach((s, i) => { if (s.getBoundingClientRect().top - line <= 0) idx = i; });
      const doc = document.documentElement, max = doc.scrollHeight - window.innerHeight;
      const ratio = max > 0 ? Math.min(1, window.scrollY / max) : 0;
      if (ratio > 0.995) idx = secs.length - 1;
      if (idx !== active) {
        links.forEach((a, i) => a.classList.toggle("on", i === idx));
        active = idx;
      }
      const a = links[idx];
      if (a) {
        marker.style.transform = `translateY(${a.offsetTop}px)`;
        marker.style.height = a.offsetHeight + "px";
        fill.style.height = (a.offsetTop + a.offsetHeight) + "px";
      }
      prog.textContent = `przeczytane ${Math.round(ratio * 100)}%`;
    }
    window.addEventListener("scroll", () => requestAnimationFrame(update), { passive: true });
    window.addEventListener("resize", update);
    update();
  }

  // --- słownik: podświetl termin z adresu
  function flash() {
    const id = location.hash.slice(1);
    const el = id && document.getElementById(id);
    if (el && el.classList.contains("term")) { el.classList.remove("flash"); void el.offsetWidth; el.classList.add("flash"); setTimeout(() => el.classList.remove("flash"), 2200); }
  }
  window.addEventListener("hashchange", flash);
  flash();
})();
