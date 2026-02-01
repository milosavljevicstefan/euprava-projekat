const API_BASE = "http://localhost:8081";

const state = {
  vrtici: [],
  filterTip: "",
  filterGrad: "",
  filterOpstina: "",
  search: "",
};

const el = {
  cards: document.getElementById("cards"),
  filterTip: document.getElementById("filter-tip"),
  filterGrad: document.getElementById("filter-grad"),
  filterOpstina: document.getElementById("filter-opstina"),
  filterReset: document.getElementById("filter-reset"),
  search: document.getElementById("search"),
  statTotal: document.getElementById("stat-total"),
  statPublic: document.getElementById("stat-public"),
  statPrivate: document.getElementById("stat-private"),
  statOccupancy: document.getElementById("stat-occupancy"),
  serviceStatus: document.getElementById("service-status"),
  apiBase: document.getElementById("api-base"),
  form: document.getElementById("vrtic-form"),
  formStatus: document.getElementById("form-status"),
  loginForm: document.getElementById("login-form"),
  loginStatus: document.getElementById("login-status"),
};

el.apiBase.textContent = API_BASE;

function uniqueValues(key) {
  return Array.from(new Set(state.vrtici.map((v) => v[key]).filter(Boolean))).sort();
}

function applyFilters(list) {
  return list
    .filter((v) => (state.filterTip ? v.tip === state.filterTip : true))
    .filter((v) => (state.filterGrad ? v.grad === state.filterGrad : true))
    .filter((v) => (state.filterOpstina ? v.opstina === state.filterOpstina : true))
    .filter((v) =>
      state.search
        ? v.naziv.toLowerCase().includes(state.search.toLowerCase())
        : true
    )
    .sort((a, b) => (a.naziv || "").localeCompare(b.naziv || ""));
}

function renderFilters() {
  const grads = uniqueValues("grad");
  const opstine = uniqueValues("opstina");

  el.filterGrad.innerHTML =
    `<option value="">Sve</option>` +
    grads.map((g) => `<option value="${g}">${g}</option>`).join("");
  el.filterOpstina.innerHTML =
    `<option value="">Sve</option>` +
    opstine.map((o) => `<option value="${o}">${o}</option>`).join("");

  el.filterGrad.value = state.filterGrad;
  el.filterOpstina.value = state.filterOpstina;
}

function renderStats() {
  const total = state.vrtici.length;
  const publicCount = state.vrtici.filter((v) => v.tip === "drzavni").length;
  const privateCount = state.vrtici.filter((v) => v.tip === "privatni").length;
  const occupancy = state.vrtici.reduce((sum, v) => {
    const pct = v.max_kapacitet ? v.trenutno_upisano / v.max_kapacitet : 0;
    return sum + pct;
  }, 0);
  const avg = total ? Math.round((occupancy / total) * 100) : 0;

  el.statTotal.textContent = total;
  el.statPublic.textContent = publicCount;
  el.statPrivate.textContent = privateCount;
  el.statOccupancy.textContent = `${avg}%`;
}

function renderCards() {
  const filtered = applyFilters(state.vrtici);
  el.cards.innerHTML = "";

  filtered.forEach((v, idx) => {
    const pct = v.max_kapacitet
      ? Math.min(100, Math.round((v.trenutno_upisano / v.max_kapacitet) * 100))
      : 0;
    const card = document.createElement("article");
    card.className = "card";
    card.style.animationDelay = `${idx * 0.05}s`;
    card.innerHTML = `
      <div class="badge">${v.tip || "n/a"}</div>
      <h3>${v.naziv || "Bez naziva"}</h3>
      <div><strong>${v.grad || ""}</strong> • ${v.opstina || ""}</div>
      <div class="progress"><span style="width:${pct}%"></span></div>
      <div class="muted">${v.trenutno_upisano || 0} / ${v.max_kapacitet || 0} upisano</div>
    `;
    el.cards.appendChild(card);
  });

  if (filtered.length === 0) {
    el.cards.innerHTML = "<div class='card'>Nema rezultata za izabrane filtere.</div>";
  }
}

function renderAll() {
  renderFilters();
  renderStats();
  renderCards();
}

async function fetchVrtici() {
  try {
    const res = await fetch(`${API_BASE}/vrtici`);
    if (!res.ok) throw new Error("API error");
    const data = await res.json();
    state.vrtici = Array.isArray(data) ? data : [];
    el.serviceStatus.textContent = "Online";
    el.serviceStatus.style.background = "rgba(38, 208, 206, 0.2)";
    renderAll();
  } catch (err) {
    el.serviceStatus.textContent = "Offline";
    el.serviceStatus.style.background = "rgba(249, 72, 72, 0.25)";
    el.cards.innerHTML = "<div class='card'>Ne mogu da se povezem na servis.</div>";
  }
}

el.filterTip.addEventListener("change", (e) => {
  state.filterTip = e.target.value;
  renderCards();
});

eleFilterBind();

function eleFilterBind() {
  el.filterGrad.addEventListener("change", (e) => {
    state.filterGrad = e.target.value;
    renderCards();
  });

  el.filterOpstina.addEventListener("change", (e) => {
    state.filterOpstina = e.target.value;
    renderCards();
  });

  el.filterReset.addEventListener("click", () => {
    state.filterTip = "";
    state.filterGrad = "";
    state.filterOpstina = "";
    state.search = "";
    el.search.value = "";
    renderAll();
  });

  el.search.addEventListener("input", (e) => {
    state.search = e.target.value;
    renderCards();
  });
}

el.form.addEventListener("submit", async (e) => {
  e.preventDefault();
  el.formStatus.textContent = "Saljem...";
  const formData = new FormData(e.target);
  const payload = Object.fromEntries(formData.entries());
  payload.max_kapacitet = Number(payload.max_kapacitet);
  payload.trenutno_upisano = Number(payload.trenutno_upisano);

  try {
    const res = await fetch(`${API_BASE}/vrtici`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!res.ok) throw new Error("API error");
    el.formStatus.textContent = "Sacuvano.";
    e.target.reset();
    await fetchVrtici();
  } catch (err) {
    el.formStatus.textContent = "Greska pri upisu.";
  }
});

el.loginForm.addEventListener("submit", (e) => {
  e.preventDefault();
  el.loginStatus.textContent = "Backend autentikacija jos nije povezana.";
});

fetchVrtici();
