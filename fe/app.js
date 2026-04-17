const API_VRTICI = "http://localhost:8081";
const API_AUTH = "http://localhost:8083";
const OPEN_DATA_URL = "http://localhost:8082/analytics";

const PAGE_BY_FILE = {
  "": "vrtici",
  "index.html": "vrtici",
  "statistika.html": "statistika",
  "ranking.html": "opstine",
  "upis.html": "upis",
  "dodaj.html": "dodaj",
  "zahtevi.html": "zahtevi",
  "analitika.html": "analitika",
  "login.html": "login",
  "registracija.html": "registracija",
  "profil.html": "profil",
};

const PUBLIC_PAGES = new Set(["vrtici", "statistika", "opstine", "login", "registracija"]);
const USER_ONLY_PAGES = new Set(["upis", "profil"]);
const ADMIN_ONLY_PAGES = new Set(["dodaj", "zahtevi", "analitika", "profil"]);

const NAV_LINKS = [
  { href: "index.html", nav: "vrtici", label: "Vrtici" },
  { href: "statistika.html", nav: "statistika", label: "Statistika" },
  { href: "ranking.html", nav: "opstine", label: "Najbolje opstine" },
  { href: "upis.html", nav: "upis", label: "Upis deteta" },
  { href: "dodaj.html", nav: "dodaj", label: "CRUD" },
  { href: "zahtevi.html", nav: "zahtevi", label: "Zahtevi" },
  { href: "analitika.html", nav: "analitika", label: "Open Data Analitika" },
  { href: "login.html", nav: "login", label: "Login" },
  { href: "registracija.html", nav: "registracija", label: "Registracija" },
  { href: "profil.html", nav: "profil", label: "Profil" },
];

const state = {
  vrtici: [],
  kriticni: [],
  opstinaReport: [],
  mojePrijave: [],
  adminPrijave: [],
  filterTip: "",
  filterGrad: "",
  filterOpstina: "",
  search: "",
  sortMode: "naziv",
  editingVrticId: "",
};

function byId(id) { return document.getElementById(id); }

const el = {
  cards: byId("cards"), manageCards: byId("manage-cards"), criticalCards: byId("critical-cards"), opstinaReport: byId("opstina-report"),
  myRequests: byId("my-requests-list"), adminRequests: byId("requests-admin-list"),
  filterTip: byId("filter-tip"), filterGrad: byId("filter-grad"), filterOpstina: byId("filter-opstina"), filterReset: byId("filter-reset"), search: byId("search"), sortMode: byId("sort-mode"),
  statTotal: byId("stat-total"), statPublic: byId("stat-public"), statPrivate: byId("stat-private"), statOccupancy: byId("stat-occupancy"),
  serviceStatus: byId("service-status"), apiBase: byId("api-base"),
  vrticForm: byId("vrtic-form"), vrticFormStatus: byId("form-status"), vrticFormTitle: byId("vrtic-form-title"), vrticIdInput: byId("vrtic-id"), saveVrticBtn: byId("save-vrtic-btn"), cancelEditBtn: byId("cancel-edit"),
  loginForm: byId("login-form"), loginStatus: byId("login-status"), userInfo: byId("user-info"),
  registerForm: byId("register-form"), registerStatus: byId("register-status"),
  upisForm: byId("upis-form"), upisStatus: byId("upis-status"), vrticSelect: byId("upis-vrtic-id"),
  downloadReport: byId("download-opstina-report"), reportStatus: byId("report-status"),
  profileEmail: byId("profile-email"), profileRole: byId("profile-role"), profileCreated: byId("profile-created"), profileStatus: byId("profile-status"), profileRefresh: byId("profile-refresh"), profileLogout: byId("profile-logout"),
};

function currentFileName() {
  const path = window.location.pathname || "";
  const file = path.split("/").pop() || "";
  return file === "" ? "index.html" : file;
}

const page = document.body?.dataset?.page || PAGE_BY_FILE[currentFileName()] || "vrtici";

const tokenStore = {
  get access() { return localStorage.getItem("access_token"); },
  set(token) { localStorage.setItem("access_token", token); },
  clear() { localStorage.removeItem("access_token"); },
};

function normalizeRole(role) { return String(role || "").trim().toLowerCase(); }

function decodeTokenPayload() {
  const token = tokenStore.access;
  if (!token) return null;
  try {
    const payload = token.split(".")[1];
    return JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")));
  } catch (_err) { return null; }
}

function currentSession() {
  const payload = decodeTokenPayload();
  if (!payload?.sub) return null;
  const exp = Number(payload.exp || 0);
  if (exp && Date.now() >= exp * 1000) { tokenStore.clear(); return null; }
  return { email: String(payload.sub || "").trim().toLowerCase(), role: normalizeRole(payload.role) };
}

function isAdminRole(role) { return role === "admin"; }
function isUserRole(role) { return role === "korisnik"; }
function roleLabel(role) { return role === "admin" ? "Admin" : role === "korisnik" ? "Korisnik" : "Gost"; }
function authHeaders() { return tokenStore.access ? { Authorization: `Bearer ${tokenStore.access}` } : null; }
function redirectTo(fileName) { if (currentFileName() !== fileName) window.location.replace(fileName); }

function hydrateNav() {
  document.querySelectorAll(".nav-links").forEach((nav) => {
    nav.innerHTML = NAV_LINKS.map((item) => `<a href="${item.href}" data-nav="${item.nav}">${item.label}</a>`).join("");
  });
}

function canAccessPage(session, targetPage) {
  if (!session) return PUBLIC_PAGES.has(targetPage);
  if (isAdminRole(session.role)) return !["login", "registracija"].includes(targetPage) && (PUBLIC_PAGES.has(targetPage) || ADMIN_ONLY_PAGES.has(targetPage));
  if (isUserRole(session.role)) return !["login", "registracija"].includes(targetPage) && (PUBLIC_PAGES.has(targetPage) || USER_ONLY_PAGES.has(targetPage));
  return false;
}

function guardPageAccess() {
  const session = currentSession();
  if (canAccessPage(session, page)) return true;
  if (!session) { redirectTo("login.html"); return false; }
  redirectTo("index.html");
  return false;
}

function setNavVisibility() {
  const session = currentSession();
  document.querySelectorAll(".nav-links a").forEach((link) => {
    const targetPage = PAGE_BY_FILE[(link.getAttribute("href") || "").trim()] || link.dataset.nav || "";
    link.style.display = canAccessPage(session, targetPage) ? "" : "none";
  });
}

function setNavActive() {
  document.querySelectorAll("[data-nav]").forEach((link) => {
    if (link.dataset.nav === page) link.classList.add("active-nav");
  });
}

function uniqueValues(key) {
  return Array.from(new Set(state.vrtici.map((v) => v[key]).filter(Boolean))).sort((a, b) => String(a).localeCompare(String(b), "sr"));
}

function freePlaces(v) {
  if (typeof v.slobodna_mesta === "number") return v.slobodna_mesta;
  return Number(v.max_kapacitet || 0) - Number(v.trenutno_upisano || 0);
}

function applyFilters(list) {
  return list
    .filter((v) => (state.filterTip ? v.tip === state.filterTip : true))
    .filter((v) => (state.filterGrad ? String(v.grad || "").toLowerCase() === state.filterGrad.toLowerCase() : true))
    .filter((v) => (state.filterOpstina ? String(v.opstina || "").toLowerCase() === state.filterOpstina.toLowerCase() : true))
    .filter((v) => (!state.search ? true : String(v.naziv || "").toLowerCase().includes(state.search.toLowerCase())));
}

function getDisplayedVrtici() {
  const filtered = applyFilters([...state.vrtici]);
  if (state.sortMode === "slobodna_mesta") filtered.sort((a, b) => freePlaces(b) - freePlaces(a));
  else filtered.sort((a, b) => String(a.naziv || "").localeCompare(String(b.naziv || ""), "sr"));
  return filtered;
}

function renderFilters() {
  if (!el.filterGrad || !el.filterOpstina) return;
  const grads = uniqueValues("grad");
  const opstine = uniqueValues("opstina");
  el.filterGrad.innerHTML = `<option value="">Svi</option>${grads.map((g) => `<option value="${g}">${g}</option>`).join("")}`;
  el.filterOpstina.innerHTML = `<option value="">Sve</option>${opstine.map((o) => `<option value="${o}">${o}</option>`).join("")}`;
  el.filterGrad.value = state.filterGrad;
  el.filterOpstina.value = state.filterOpstina;
}

function renderStats() {
  if (!el.statTotal || !el.statPublic || !el.statPrivate || !el.statOccupancy) return;
  const total = state.vrtici.length;
  const publicCount = state.vrtici.filter((v) => v.tip === "drzavni").length;
  const privateCount = state.vrtici.filter((v) => v.tip === "privatni").length;
  const occupancySum = state.vrtici.reduce((sum, v) => sum + Number(v.popunjenost || 0), 0);
  const avg = total ? Math.round((occupancySum / total) * 100) : 0;
  el.statTotal.textContent = String(total);
  el.statPublic.textContent = String(publicCount);
  el.statPrivate.textContent = String(privateCount);
  el.statOccupancy.textContent = `${avg}%`;
}

function vrticCardHTML(v, idx, mode = "public") {
  const pct = Number(v.max_kapacitet || 0) > 0 ? Math.min(100, Math.round((Number(v.trenutno_upisano || 0) / Number(v.max_kapacitet || 1)) * 100)) : 0;
  const session = currentSession();
  let actionHTML = "";

  if (mode === "public") {
    if (!session) actionHTML = `<a class="btn ghost small" href="login.html">Uloguj se za upis</a>`;
    else if (isUserRole(session.role)) {
      actionHTML = freePlaces(v) > 0 ? `<a class="btn secondary small" href="upis.html?vrtic=${v.id}">Posalji zahtev za upis</a>` : `<div class="muted">Trenutno nema slobodnih mesta.</div>`;
    }
  }

  if (mode === "admin") {
    actionHTML = `<div class="card-actions"><button class="btn ghost small" data-action="edit" data-id="${v.id}">Izmeni</button><button class="btn danger small" data-action="delete" data-id="${v.id}">Obrisi</button></div>`;
  }

  return `
    <div class="badge">${v.tip || "n/a"}</div>
    ${state.sortMode === "slobodna_mesta" && mode === "public" ? `<div class="rank-badge">Rang #${idx + 1}</div>` : ""}
    <h3>${v.naziv || "Bez naziva"}</h3>
    <div><strong>${v.grad || ""}</strong> - ${v.opstina || ""}</div>
    <div class="progress"><span style="width:${pct}%"></span></div>
    <div class="muted">${v.trenutno_upisano || 0} / ${v.max_kapacitet || 0} upisano</div>
    <div class="muted">Slobodna mesta: ${freePlaces(v)}</div>
    <div class="muted">${v.kriticno ? "Kriticno popunjen" : "Stabilna popunjenost"}</div>
    ${mode === "admin" ? `<div class="muted">Vlasnik: ${v.created_by || "sistem"}</div>` : ""}
    ${actionHTML}`;
}

function renderCards() {
  if (!el.cards) return;
  const displayed = getDisplayedVrtici();
  el.cards.innerHTML = "";
  displayed.forEach((v, idx) => {
    const card = document.createElement("article");
    card.className = "card";
    card.style.animationDelay = `${idx * 0.04}s`;
    card.innerHTML = vrticCardHTML(v, idx, "public");
    el.cards.appendChild(card);
  });
  if (!displayed.length) el.cards.innerHTML = "<div class='card'>Nema vrtica za izabrane filtere.</div>";
}

function renderManageCards() {
  if (!el.manageCards) return;
  const displayed = getDisplayedVrtici();
  el.manageCards.innerHTML = "";
  displayed.forEach((v, idx) => {
    const card = document.createElement("article");
    card.className = "card";
    card.style.animationDelay = `${idx * 0.03}s`;
    card.innerHTML = vrticCardHTML(v, idx, "admin");
    el.manageCards.appendChild(card);
  });
  if (!displayed.length) el.manageCards.innerHTML = "<div class='card'>Nema vrtica u bazi.</div>";
}

function renderCriticalCards() {
  if (!el.criticalCards) return;
  el.criticalCards.innerHTML = "";
  state.kriticni.forEach((v) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="badge">${v.tip || "n/a"}</div><h3>${v.naziv || "Bez naziva"}</h3><div><strong>${v.grad || ""}</strong> - ${v.opstina || ""}</div><div class="muted">Popunjenost: ${Math.round(Number(v.popunjenost || 0) * 100)}%</div><div class="muted">Slobodna mesta: ${freePlaces(v)}</div>`;
    el.criticalCards.appendChild(card);
  });
  if (!state.kriticni.length) el.criticalCards.innerHTML = "<div class='card'>Nema kriticno popunjenih vrtica.</div>";
}

function renderOpstinaReport() {
  if (!el.opstinaReport) return;
  el.opstinaReport.innerHTML = "";
  state.opstinaReport.forEach((row) => {
    const pct = Math.round(Number(row.popunjenost || 0) * 100);
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<h3>${row.opstina || "Nepoznata"}</h3><div class="muted">Broj vrtica: ${row.broj_vrtica || 0}</div><div class="muted">Kapacitet: ${row.ukupan_kapacitet || 0}</div><div class="muted">Upisano: ${row.ukupno_upisano || 0}</div><div class="muted">Popunjenost: ${pct}%</div>`;
    el.opstinaReport.appendChild(card);
  });
  if (!state.opstinaReport.length) el.opstinaReport.innerHTML = "<div class='card'>Nema podataka za izvestaj.</div>";
}

function populateVrticSelect() {
  if (!el.vrticSelect) return;
  const available = [...state.vrtici].filter((v) => freePlaces(v) > 0).sort((a, b) => String(a.naziv || "").localeCompare(String(b.naziv || ""), "sr"));
  el.vrticSelect.innerHTML = `<option value="">Izaberi vrtic</option>${available.map((v) => `<option value="${v.id}">${v.naziv} (${v.grad} - ${v.opstina})</option>`).join("")}`;
  const requestedId = new URLSearchParams(window.location.search).get("vrtic");
  if (requestedId && available.some((v) => String(v.id) === requestedId)) el.vrticSelect.value = requestedId;
}

function requestStatusClass(status) { return status === "odobren" ? "status-chip ok" : status === "odbijen" ? "status-chip danger" : "status-chip"; }
function canProcessRequest(item) { const session = currentSession(); return !!session && isAdminRole(session.role); }

function renderMyRequests() {
  if (!el.myRequests) return;
  el.myRequests.innerHTML = "";
  state.mojePrijave.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${item.status}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Roditelj: ${item.ime_roditelja}</div><div class="muted">Dete: ${item.ime_deteta}</div><div class="muted">Broj godina: ${item.broj_godina}</div><div class="muted">Poslato: ${new Date(item.created_at).toLocaleString("sr-RS")}</div>`;
    el.myRequests.appendChild(card);
  });
  if (!state.mojePrijave.length) el.myRequests.innerHTML = "<div class='card'>Jos nema poslatih zahteva.</div>";
}

function renderAdminRequests() {
  if (!el.adminRequests) return;
  el.adminRequests.innerHTML = "";
  state.adminPrijave.forEach((item) => {
    const actionable = item.status === "na_cekanju" && canProcessRequest(item);
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${item.status}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Vlasnik vrtica: ${item.vrtic_owner}</div><div class="muted">Korisnik: ${item.korisnik_email}</div><div class="muted">Ime roditelja: ${item.ime_roditelja}</div><div class="muted">Ime deteta: ${item.ime_deteta}</div><div class="muted">Broj godina: ${item.broj_godina}</div><div class="muted">Poslato: ${new Date(item.created_at).toLocaleString("sr-RS")}</div>${actionable ? `<div class="card-actions"><button class="btn secondary small" data-request-action="odobri" data-id="${item.id}">Odobri</button><button class="btn danger small" data-request-action="odbij" data-id="${item.id}">Odbij</button></div>` : `<div class="muted">Zahtev je vec obradjen.</div>`}`;
    el.adminRequests.appendChild(card);
  });
  if (!state.adminPrijave.length) el.adminRequests.innerHTML = "<div class='card'>Nema zahteva za upis.</div>";
}

function resetEditMode() {
  state.editingVrticId = "";
  if (el.vrticIdInput) el.vrticIdInput.value = "";
  if (el.vrticFormTitle) el.vrticFormTitle.textContent = "Dodaj novi vrtic";
  if (el.saveVrticBtn) el.saveVrticBtn.textContent = "Sacuvaj";
  if (el.cancelEditBtn) el.cancelEditBtn.style.display = "none";
}

function setEditMode(v) {
  state.editingVrticId = v.id;
  if (el.vrticIdInput) el.vrticIdInput.value = v.id;
  if (el.vrticFormTitle) el.vrticFormTitle.textContent = `Izmena: ${v.naziv}`;
  if (el.saveVrticBtn) el.saveVrticBtn.textContent = "Sacuvaj izmenu";
  if (el.cancelEditBtn) el.cancelEditBtn.style.display = "inline-flex";
  if (!el.vrticForm) return;
  el.vrticForm.naziv.value = v.naziv || "";
  el.vrticForm.tip.value = v.tip || "drzavni";
  el.vrticForm.grad.value = v.grad || "";
  el.vrticForm.opstina.value = v.opstina || "";
  el.vrticForm.max_kapacitet.value = v.max_kapacitet || "";
  el.vrticForm.trenutno_upisano.value = v.trenutno_upisano || "";
}

function renderAll() {
  renderFilters(); renderStats(); renderCards(); renderManageCards(); renderCriticalCards(); renderOpstinaReport(); populateVrticSelect(); renderMyRequests(); renderAdminRequests();
}

async function fetchVrtici() {
  try {
    const res = await fetch(`${API_VRTICI}/vrtici`);
    if (!res.ok) throw new Error("API error");
    state.vrtici = await res.json();
    if (el.serviceStatus) { el.serviceStatus.textContent = "Online"; el.serviceStatus.style.background = "rgba(38, 208, 206, 0.2)"; }
    renderAll();
  } catch (_err) {
    if (el.serviceStatus) { el.serviceStatus.textContent = "Offline"; el.serviceStatus.style.background = "rgba(249, 72, 72, 0.25)"; }
    if (el.cards) el.cards.innerHTML = "<div class='card'>Ne mogu da se povezem na servis.</div>";
    if (el.manageCards) el.manageCards.innerHTML = "<div class='card'>Ne mogu da ucitam podatke.</div>";
  }
}

async function fetchKriticni() {
  if (!el.criticalCards) return;
  try {
    const res = await fetch(`${API_VRTICI}/vrtici/kriticni`);
    if (!res.ok) throw new Error("API error");
    state.kriticni = await res.json();
    renderCriticalCards();
  } catch (_err) { el.criticalCards.innerHTML = "<div class='card'>Ne mogu da ucitam kriticne vrtice.</div>"; }
}

async function fetchOpstinaReportJson() {
  if (!el.opstinaReport) return;
  try {
    const res = await fetch(`${API_VRTICI}/vrtici/izvestaj/opstina`);
    if (!res.ok) throw new Error("API error");
    state.opstinaReport = await res.json();
    renderOpstinaReport();
  } catch (_err) { el.opstinaReport.innerHTML = "<div class='card'>Ne mogu da ucitam izvestaj po opstini.</div>"; }
}

async function fetchMyRequests() {
  if (!el.myRequests) return;
  const headers = authHeaders();
  if (!headers) { state.mojePrijave = []; renderMyRequests(); return; }
  try {
    const res = await fetch(`${API_VRTICI}/zahtevi-upisa/moji`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.mojePrijave = await res.json();
    renderMyRequests();
  } catch (err) { el.myRequests.innerHTML = `<div class='card'>${err.message || "Ne mogu da ucitam tvoje zahteve."}</div>`; }
}

async function fetchAdminRequests() {
  if (!el.adminRequests) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/zahtevi-upisa`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.adminPrijave = await res.json();
    renderAdminRequests();
  } catch (err) { el.adminRequests.innerHTML = `<div class='card'>${err.message || "Ne mogu da ucitam zahteve."}</div>`; }
}
async function createVrtic(payload) {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze da dodaje vrtice.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/vrtici`, { method: "POST", headers: { ...headers, "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
}

async function updateVrticById(id, payload) {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze da menja vrtice.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/vrtici/${id}`, { method: "PUT", headers: { ...headers, "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
}

async function deleteVrticById(id) {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze da brise vrtice.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/vrtici/${id}`, { method: "DELETE", headers });
  if (!res.ok) throw new Error(await res.text());
}

async function createEnrollmentRequest(payload) {
  const session = currentSession();
  if (!session || !isUserRole(session.role)) throw new Error("Samo ulogovan korisnik moze slati zahtev za upis.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa`, { method: "POST", headers: { ...headers, "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function processRequest(id, action) {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze obradjivati zahteve.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa/${id}/${action}`, { method: "PUT", headers });
  if (!res.ok) throw new Error(await res.text());
}

function bindListEvents() {
  if (el.filterTip) el.filterTip.addEventListener("change", (e) => { state.filterTip = e.target.value; renderCards(); renderManageCards(); });
  if (el.filterGrad) el.filterGrad.addEventListener("change", (e) => { state.filterGrad = e.target.value; renderCards(); renderManageCards(); });
  if (el.filterOpstina) el.filterOpstina.addEventListener("change", (e) => { state.filterOpstina = e.target.value; renderCards(); renderManageCards(); });
  if (el.search) el.search.addEventListener("input", (e) => { state.search = e.target.value.trim(); renderCards(); renderManageCards(); });
  if (el.sortMode) el.sortMode.addEventListener("change", (e) => { state.sortMode = e.target.value; renderCards(); renderManageCards(); });
  if (el.filterReset) {
    el.filterReset.addEventListener("click", () => {
      state.filterTip = ""; state.filterGrad = ""; state.filterOpstina = ""; state.search = ""; state.sortMode = "naziv";
      if (el.filterTip) el.filterTip.value = "";
      if (el.search) el.search.value = "";
      if (el.sortMode) el.sortMode.value = "naziv";
      renderFilters(); renderCards(); renderManageCards();
    });
  }
}

function bindCrudEvents() {
  if (!el.vrticForm) return;
  resetEditMode();
  if (!isAdminRole(currentSession()?.role) && el.vrticFormStatus) el.vrticFormStatus.textContent = "Samo admin moze da koristi CRUD nad vrticima.";

  el.vrticForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const formData = new FormData(el.vrticForm);
    const payload = {
      naziv: String(formData.get("naziv") || "").trim(),
      tip: String(formData.get("tip") || "drzavni").trim(),
      grad: String(formData.get("grad") || "").trim(),
      opstina: String(formData.get("opstina") || "").trim(),
      max_kapacitet: Number(formData.get("max_kapacitet") || 0),
      trenutno_upisano: Number(formData.get("trenutno_upisano") || 0),
    };
    if (el.vrticFormStatus) el.vrticFormStatus.textContent = "Sacuvam...";
    try {
      if (state.editingVrticId) { await updateVrticById(state.editingVrticId, payload); if (el.vrticFormStatus) el.vrticFormStatus.textContent = "Izmena sacuvana."; }
      else { await createVrtic(payload); if (el.vrticFormStatus) el.vrticFormStatus.textContent = "Vrtic dodat."; }
      el.vrticForm.reset(); resetEditMode(); await fetchVrtici();
    } catch (err) { if (el.vrticFormStatus) el.vrticFormStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
  });

  if (el.cancelEditBtn) el.cancelEditBtn.addEventListener("click", () => { el.vrticForm.reset(); resetEditMode(); if (el.vrticFormStatus) el.vrticFormStatus.textContent = ""; });

  if (el.manageCards) {
    el.manageCards.addEventListener("click", async (e) => {
      const target = e.target.closest("button[data-action]");
      if (!target) return;
      const { action, id } = target.dataset;
      const vrtic = state.vrtici.find((item) => String(item.id) === String(id));
      if (!vrtic) return;
      if (action === "edit") { setEditMode(vrtic); window.scrollTo({ top: 0, behavior: "smooth" }); return; }
      if (action === "delete") {
        if (!window.confirm(`Obrisi vrtic \"${vrtic.naziv}\"?`)) return;
        try { await deleteVrticById(id); if (el.vrticFormStatus) el.vrticFormStatus.textContent = "Vrtic obrisan."; await fetchVrtici(); }
        catch (err) { if (el.vrticFormStatus) el.vrticFormStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
      }
    });
  }
}

function bindLoginEvents() {
  if (!el.loginForm || !el.loginStatus || !el.userInfo) return;
  const session = currentSession();
  if (session) el.userInfo.textContent = `Trenutno ulogovan: ${session.email} (${roleLabel(session.role)})`;
  el.loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const formData = new FormData(el.loginForm);
    const payload = { email: String(formData.get("email") || "").trim(), password: String(formData.get("password") || "") };
    el.loginStatus.textContent = "Prijava...";
    try {
      const res = await fetch(`${API_AUTH}/auth/login`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      tokenStore.set(data.access_token);
      el.loginStatus.textContent = "Ulogovan.";
      el.userInfo.textContent = `Trenutno ulogovan: ${data.email} (${roleLabel(data.role)})`;
      setTimeout(() => redirectTo("index.html"), 500);
    } catch (err) { el.loginStatus.textContent = `Neuspesna prijava: ${err.message}`; }
  });
}

function bindRegisterEvents() {
  if (!el.registerForm || !el.registerStatus) return;
el.registerForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const formData = new FormData(el.registerForm);
    const payload = { email: String(formData.get("email") || "").trim(), password: String(formData.get("password") || "") };
    el.registerStatus.textContent = "Registracija...";
    try {
      const res = await fetch(`${API_AUTH}/auth/register`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(await res.text());
      el.registerStatus.textContent = "Registracija uspesna. Preusmeravam na login...";
      el.registerForm.reset();
      setTimeout(() => redirectTo("login.html"), 700);
    } catch (err) { el.registerStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
  });
}

async function fetchProfile() {
  if (!el.profileStatus || !el.profileEmail || !el.profileRole || !el.profileCreated) return;
  const headers = authHeaders();
  if (!headers) { el.profileStatus.textContent = "Nisi ulogovan."; el.profileEmail.textContent = "-"; el.profileRole.textContent = "-"; el.profileCreated.textContent = "-"; return; }
  el.profileStatus.textContent = "Ucitavam profil...";
  try {
    const res = await fetch(`${API_AUTH}/auth/profile`, { headers });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    el.profileEmail.textContent = data.email || "-";
    el.profileRole.textContent = roleLabel(data.role || "");
    el.profileCreated.textContent = data.created_at ? new Date(data.created_at).toLocaleString("sr-RS") : "-";
    el.profileStatus.textContent = "Profil je ucitan.";
  } catch (err) { el.profileStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
}

function bindProfileEvents() {
  if (el.profileRefresh) el.profileRefresh.addEventListener("click", fetchProfile);
  if (el.profileLogout) el.profileLogout.addEventListener("click", () => { tokenStore.clear(); if (el.profileStatus) el.profileStatus.textContent = "Odjavljen."; setTimeout(() => redirectTo("login.html"), 400); });
}
function bindUpisEvents() {
  if (!el.upisForm || !el.upisStatus) return;
  el.upisForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const formData = new FormData(el.upisForm);
    const payload = {
      vrtic_id: String(formData.get("vrtic_id") || "").trim(),
      ime_roditelja: String(formData.get("ime_roditelja") || "").trim(),
      ime_deteta: String(formData.get("ime_deteta") || "").trim(),
      broj_godina: Number(formData.get("broj_godina") || 0),
    };
    el.upisStatus.textContent = "Saljem zahtev...";
    try {
      await createEnrollmentRequest(payload);
      el.upisStatus.textContent = "Zahtev je poslat adminu na odobravanje.";
      el.upisForm.reset();
      populateVrticSelect();
      await fetchMyRequests();
    } catch (err) { el.upisStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
  });
}

function bindRequestAdminEvents() {
  if (!el.adminRequests) return;
  el.adminRequests.addEventListener("click", async (e) => {
    const button = e.target.closest("button[data-request-action]");
    if (!button) return;
    try {
      await processRequest(button.dataset.id, button.dataset.requestAction);
      await fetchAdminRequests();
      await fetchVrtici();
      if (el.criticalCards) await fetchKriticni();
    } catch (err) { window.alert(err.message || "Neuspesna obrada zahteva."); }
  });
}

async function downloadOpstinaPdf() {
  if (!el.downloadReport) return;
  const oldLabel = el.downloadReport.textContent;
  if (el.reportStatus) el.reportStatus.textContent = "Preuzimam...";
  el.downloadReport.disabled = true;
  try {
    const res = await fetch(`${API_VRTICI}/vrtici/izvestaj/opstina?format=pdf`);
    if (!res.ok) throw new Error("PDF nije dostupan");
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `izvestaj-opstina-${new Date().toISOString().slice(0, 10)}.pdf`;
    document.body.appendChild(a); a.click(); a.remove(); window.URL.revokeObjectURL(url);
    if (el.reportStatus) el.reportStatus.textContent = "PDF preuzet.";
  } catch (err) { if (el.reportStatus) el.reportStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
  finally { el.downloadReport.disabled = false; el.downloadReport.textContent = oldLabel; }
}

function bindReportActions() { if (el.downloadReport) el.downloadReport.addEventListener("click", downloadOpstinaPdf); }

async function loadRanking() {
  const container = byId("ranking-cards");
  if (!container) return;
  container.innerHTML = "<p>Ucitavanje...</p>";
  try {
    const res = await fetch(`${OPEN_DATA_URL}/ranking`);
    if (!res.ok) throw new Error("Ne mogu da ucitam ranking");
    const data = await res.json();
    container.innerHTML = "";
    data.forEach((r, index) => {
      container.innerHTML += `<div class="card"><span class="rank-badge">#${index + 1}</span><h3>${r.opstina}</h3><p>Kapacitet: ${r.ukupan_kapacitet}</p><p>Upisano: ${r.ukupno_upisano}</p><p>Popunjenost: ${(r.popunjenost * 100).toFixed(2)}%</p></div>`;
    });
  } catch (_err) { container.innerHTML = "<p>Greska pri ucitavanju.</p>"; }
}

function bindAnalyticsEvents() {
  const coverageBtn = byId("coverage-btn");
  const projectionBtn = byId("projection-btn");
  if (coverageBtn) {
    coverageBtn.addEventListener("click", async () => {
      const opstina = byId("coverage-opstina")?.value || "";
      const res = await fetch(`${OPEN_DATA_URL}/coverage?opstina=${encodeURIComponent(opstina)}`);
      const data = await res.json();
      const container = byId("coverage-result");
      if (container) container.innerHTML = `<div class="card"><h3>${data.opstina}</h3><p>Broj dece: ${data.broj_dece}</p><p>Kapacitet: ${data.kapacitet}</p><p>Deficit: ${data.deficit}</p><p>Pokrivenost: ${Number(data.pokrivenost).toFixed(2)}%</p></div>`;
    });
  }
  if (projectionBtn) {
    projectionBtn.addEventListener("click", async () => {
      const years = byId("projection-years")?.value || "1";
      const res = await fetch(`${OPEN_DATA_URL}/projection?years=${encodeURIComponent(years)}`);
      const data = await res.json();
      const container = byId("projection-cards");
      if (!container) return;
      container.innerHTML = "";
      data.forEach((r) => {
        container.innerHTML += `<div class="card"><h3>${r.opstina}</h3><p>Projekcija upisanih: ${r.ukupno_upisano}</p><p>Nova popunjenost: ${(r.popunjenost * 100).toFixed(2)}%</p></div>`;
      });
    });
  }
}

async function downloadPublicData() {
  const btn = byId("download-public-btn");
  const status = byId("public-download-status");
  if (!btn || !status) return;
  const oldLabel = btn.textContent;
  status.textContent = "Preuzimam...";
  btn.disabled = true;
  try {
    const res = await fetch(`${OPEN_DATA_URL}/public-data?format=json`);
    if (!res.ok) throw new Error("Podaci nisu dostupni");
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `public-data-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(a); a.click(); a.remove(); window.URL.revokeObjectURL(url);
    status.textContent = "Podaci preuzeti.";
  } catch (err) { status.textContent = `Greska: ${err.message || "Neuspesno"}`; }
  finally { btn.disabled = false; btn.textContent = oldLabel; }
}

function bindPublicDownload() { const btn = byId("download-public-btn"); if (btn) btn.addEventListener("click", downloadPublicData); }

async function bootstrap() {
  hydrateNav();
  if (!guardPageAccess()) return;
  setNavVisibility();
  setNavActive();
  if (el.apiBase) el.apiBase.textContent = API_VRTICI;
  bindListEvents();
  bindCrudEvents();
  bindLoginEvents();
  bindRegisterEvents();
  bindProfileEvents();
  bindUpisEvents();
  bindRequestAdminEvents();
  bindReportActions();
  bindAnalyticsEvents();
  bindPublicDownload();
  loadRanking();

  if (el.cards || el.manageCards || el.statTotal || el.vrticSelect) await fetchVrtici();
  if (el.opstinaReport) await fetchOpstinaReportJson();
  if (el.criticalCards) await fetchKriticni();
  if (el.myRequests) await fetchMyRequests();
  if (el.adminRequests) await fetchAdminRequests();
  if (page === "profil") await fetchProfile();
}

bootstrap();

state.ratingsSummary = {};
state.compareSelection = { left: "", right: "" };
Object.assign(el, { compareFirst: null, compareSecond: null, compareBtn: null, compareClear: null, compareStatus: null, compareResult: null });

function ratingSummaryFor(vrticId) {
  return state.ratingsSummary[String(vrticId)] || { prosecna_ocena: 0, broj_ocena: 0 };
}

function formatRatingSummary(item) {
  return item.broj_ocena ? `${item.prosecna_ocena.toFixed(1)}/5 (${item.broj_ocena} ocena)` : "Jos nema ocena";
}

async function fetchRatingsSummary() {
  try {
    const res = await fetch(`${API_VRTICI}/ocene`);
    if (!res.ok) throw new Error("Ne mogu da ucitam ocene");
    const items = await res.json();
    state.ratingsSummary = {};
    items.forEach((item) => { state.ratingsSummary[String(item.vrtic_id)] = item; });
  } catch (_err) {
    state.ratingsSummary = {};
  }
}

async function submitRating(payload) {
  const session = currentSession();
  if (!session || !isUserRole(session.role)) throw new Error("Samo ulogovan korisnik moze da ocenjuje vrtice.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/ocene`, { method: "POST", headers: { ...headers, "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
}

function ensureCompareSection() {
  if (page !== "vrtici" || byId("compare-first")) return;
  const main = document.querySelector("main.page-main") || document.querySelector("main");
  if (!main) return;
  const section = document.createElement("section");
  section.className = "vrtici compare-section";
  section.id = "poredi";
  section.innerHTML = `
    <div class="section-head">
      <div>
        <h2>Uporedi vrtice</h2>
        <p>Gost moze da uporedi dva vrtica po kapacitetu, popunjenosti, slobodnim mestima i prosecnoj oceni.</p>
      </div>
    </div>
    <div class="form-card compare-card">
      <div class="grid">
        <label>Prvi vrtic<select id="compare-first"></select></label>
        <label>Drugi vrtic<select id="compare-second"></select></label>
      </div>
      <div class="form-actions">
        <button class="btn secondary" id="compare-btn" type="button">Uporedi</button>
        <button class="btn ghost" id="compare-clear" type="button">Reset</button>
        <span id="compare-status" class="muted"></span>
      </div>
      <div id="compare-result" class="cards compare-results"></div>
    </div>`;
  main.appendChild(section);
  el.compareFirst = byId("compare-first");
  el.compareSecond = byId("compare-second");
  el.compareBtn = byId("compare-btn");
  el.compareClear = byId("compare-clear");
  el.compareStatus = byId("compare-status");
  el.compareResult = byId("compare-result");
}

function renderCompareOptions() {
  if (!el.compareFirst || !el.compareSecond) return;
  const options = `<option value="">Izaberi vrtic</option>${state.vrtici.map((v) => `<option value="${v.id}">${v.naziv} (${v.grad} - ${v.opstina})</option>`).join("")}`;
  el.compareFirst.innerHTML = options;
  el.compareSecond.innerHTML = options;
  el.compareFirst.value = state.compareSelection.left;
  el.compareSecond.value = state.compareSelection.right;
}

function compareWinner(a, b, leftValue, rightValue, prefersHigher = true) {
  if (leftValue === rightValue) return "Izjednaceno";
  const leftWins = prefersHigher ? leftValue > rightValue : leftValue < rightValue;
  return leftWins ? a.naziv : b.naziv;
}

function compareScore(v) {
  const rating = ratingSummaryFor(v.id);
  return freePlaces(v) * 2 + (100 - Math.round(Number(v.popunjenost || 0) * 100)) + rating.prosecna_ocena * 20;
}

function renderCompareResult() {
  if (!el.compareResult || !el.compareStatus) return;
  const left = state.vrtici.find((item) => String(item.id) === String(state.compareSelection.left));
  const right = state.vrtici.find((item) => String(item.id) === String(state.compareSelection.right));
  if (!left || !right) {
    el.compareStatus.textContent = "Izaberi dva vrtica za poredenje.";
    el.compareResult.innerHTML = "";
    return;
  }
  if (String(left.id) === String(right.id)) {
    el.compareStatus.textContent = "Izaberi dva razlicita vrtica.";
    el.compareResult.innerHTML = "";
    return;
  }

  const leftRating = ratingSummaryFor(left.id);
  const rightRating = ratingSummaryFor(right.id);
  const winner = compareScore(left) >= compareScore(right) ? left.naziv : right.naziv;
  const occupancyLeft = Math.round(Number(left.popunjenost || 0) * 100);
  const occupancyRight = Math.round(Number(right.popunjenost || 0) * 100);

  el.compareStatus.textContent = `Predlog sistema: ${winner} deluje pogodnije za upis.`;
  el.compareResult.innerHTML = `
    <article class="card compare-column">
      <h3>${left.naziv}</h3>
      <div class="muted">${left.grad} - ${left.opstina}</div>
      <div class="muted">Tip: ${left.tip}</div>
      <div class="muted">Kapacitet: ${left.max_kapacitet}</div>
      <div class="muted">Upisano: ${left.trenutno_upisano}</div>
      <div class="muted">Slobodna mesta: ${freePlaces(left)}</div>
      <div class="muted">Popunjenost: ${occupancyLeft}%</div>
      <div class="muted">Ocena: ${formatRatingSummary(leftRating)}</div>
    </article>
    <article class="card compare-column">
      <h3>${right.naziv}</h3>
      <div class="muted">${right.grad} - ${right.opstina}</div>
      <div class="muted">Tip: ${right.tip}</div>
      <div class="muted">Kapacitet: ${right.max_kapacitet}</div>
      <div class="muted">Upisano: ${right.trenutno_upisano}</div>
      <div class="muted">Slobodna mesta: ${freePlaces(right)}</div>
      <div class="muted">Popunjenost: ${occupancyRight}%</div>
      <div class="muted">Ocena: ${formatRatingSummary(rightRating)}</div>
    </article>
    <article class="card compare-summary-card">
      <h3>Zakljucak poredenja</h3>
      <div class="muted">Vise slobodnih mesta: ${compareWinner(left, right, freePlaces(left), freePlaces(right), true)}</div>
      <div class="muted">Manja popunjenost: ${compareWinner(left, right, occupancyLeft, occupancyRight, false)}</div>
      <div class="muted">Bolja ocena korisnika: ${compareWinner(left, right, leftRating.prosecna_ocena, rightRating.prosecna_ocena, true)}</div>
      <div class="muted">Veci kapacitet: ${compareWinner(left, right, left.max_kapacitet, right.max_kapacitet, true)}</div>
      <div class="muted">Predlog: ${winner}</div>
    </article>`;
}

function bindCompareEvents() {
  ensureCompareSection();
  if (el.compareFirst) el.compareFirst.addEventListener("change", (e) => { state.compareSelection.left = e.target.value; renderCompareResult(); });
  if (el.compareSecond) el.compareSecond.addEventListener("change", (e) => { state.compareSelection.right = e.target.value; renderCompareResult(); });
  if (el.compareBtn) el.compareBtn.addEventListener("click", renderCompareResult);
  if (el.compareClear) el.compareClear.addEventListener("click", () => {
    state.compareSelection = { left: "", right: "" };
    renderCompareOptions();
    renderCompareResult();
  });
  if (el.cards) {
    el.cards.addEventListener("click", (e) => {
      const compareBtn = e.target.closest("button[data-compare-add]");
      if (!compareBtn) return;
      const vrticId = String(compareBtn.dataset.compareAdd || "");
      if (!state.compareSelection.left || state.compareSelection.left === vrticId) state.compareSelection.left = vrticId;
      else state.compareSelection.right = vrticId;
      renderCompareOptions();
      renderCompareResult();
      const section = byId("poredi");
      if (section) section.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }
}

function bindRatingEvents() {
  if (!el.cards) return;
  el.cards.addEventListener("click", async (e) => {
    const button = e.target.closest("button[data-rate-submit]");
    if (!button) return;
    const vrticId = String(button.dataset.rateSubmit || "");
    const select = document.querySelector(`select[data-rate-select="${vrticId}"]`);
    if (!select) return;
    button.disabled = true;
    const oldLabel = button.textContent;
    button.textContent = "Cuvam...";
    try {
      await submitRating({ vrtic_id: vrticId, ocena: Number(select.value || 0) });
      await fetchRatingsSummary();
      renderAll();
    } catch (err) {
      window.alert(err.message || "Ocena nije sacuvana.");
    } finally {
      button.disabled = false;
      button.textContent = oldLabel;
    }
  });
}

const __originalVrticCardHTML = vrticCardHTML;
vrticCardHTML = function(v, idx, mode = "public") {
  const session = currentSession();
  const rating = ratingSummaryFor(v.id);
  let html = __originalVrticCardHTML(v, idx, mode);
  if (mode === "public") {
    html += `<div class="muted rating-summary">Ocena korisnika: ${formatRatingSummary(rating)}</div>`;
    html += `<div class="card-actions"><button class="btn ghost small" type="button" data-compare-add="${v.id}">Dodaj za poredenje</button></div>`;
    if (session && isUserRole(session.role)) {
      html += `<div class="rating-box"><label>Oceni vrtic<select data-rate-select="${v.id}"><option value="5">5 - Odlicno</option><option value="4">4 - Vrlo dobro</option><option value="3">3 - Dobro</option><option value="2">2 - Slabo</option><option value="1">1 - Lose</option></select></label><button class="btn secondary small" type="button" data-rate-submit="${v.id}">Sacuvaj ocenu</button></div>`;
    }
  }
  return html;
};

const __originalRenderAll = renderAll;
renderAll = function() {
  __originalRenderAll();
  renderCompareOptions();
  renderCompareResult();
};

ensureCompareSection();
bindCompareEvents();
bindRatingEvents();
fetchRatingsSummary().then(() => renderAll());



