const API_VRTICI = "http://localhost:8081";
const API_AUTH = "http://localhost:8083";
const OPEN_DATA_URL = "http://localhost:8082/analytics";
const API_OPEN_DATA = "http://localhost:8084";
const PAGE_BY_FILE = {
  "": "vrtici",
  "index.html": "vrtici",
  "statistika.html": "statistika",
  "ranking.html": "opstine",
  "upis.html": "upis",
  "dodaj.html": "dodaj",
  "zahtevi.html": "zahtevi",
  "vaspitac.html": "vaspitac",
  "javni_podaci.html": "javni_podaci",
  "login.html": "login",
  "registracija.html": "registracija",
  "profil.html": "profil",
};

const PUBLIC_PAGES = new Set(["vrtici", "statistika", "javni_podaci", "login", "registracija"]);
const USER_ONLY_PAGES = new Set(["upis", "profil"]);
const ADMIN_ONLY_PAGES = new Set(["dodaj", "zahtevi", "profil"]);
const EDUCATOR_ONLY_PAGES = new Set(["vaspitac", "profil"]);

const NAV_LINKS = [
  { href: "index.html", nav: "vrtici", label: "Vrtici" },
  { href: "statistika.html", nav: "statistika", label: "Statistika" },
  { href: "ranking.html", nav: "opstine", label: "Najbolje opstine" },
  { href: "upis.html", nav: "upis", label: "Upis deteta" },
  { href: "dodaj.html", nav: "dodaj", label: "CRUD" },
  { href: "zahtevi.html", nav: "zahtevi", label: "Zahtevi" },
  { href: "analitika.html", nav: "analitika", label: "Open Data Analitika" },
  { href: "vaspitac.html", nav: "vaspitac", label: "Vaspitac" },
  { href: "login.html", nav: "login", label: "Login" },
  { href: "registracija.html", nav: "registracija", label: "Registracija" },
  { href: "profil.html", nav: "profil", label: "Profil" },
  { href: "javni_podaci.html", nav: "javni_podaci", label: "Javni podaci" },
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
  editingRequestId: "",
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
  assignmentForm: byId("assignment-form"), assignmentStatus: byId("assignment-status"), assignmentVrticSelect: byId("assignment-vrtic-id"), assignmentEducatorSelect: byId("assignment-educator-email"), assignmentCards: byId("assignment-cards"),
  roditeljOptions: byId("parent-educator-options"), sastanakForm: byId("sastanak-form"), sastanakStatus: byId("sastanak-status"), sastanakZahtevSelect: byId("sastanak-zahtev-id"), sastanakEducatorSelect: byId("sastanak-vaspitac-email"), myMeetings: byId("my-meetings-list"), myNotifications: byId("my-notifications-list"),
  educatorChildren: byId("educator-children-list"), educatorMeetings: byId("educator-meetings-list"), educatorNoticeForm: byId("educator-notice-form"), educatorNoticeStatus: byId("educator-notice-status"), educatorChildSelect: byId("educator-zahtev-id"),
  upisFormTitle: byId("upis-form-title"), upisSubmitBtn: byId("upis-submit-btn"), upisCancelEdit: byId("upis-cancel-edit"),
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
function isUserRole(role) { return role === "roditelj" || role === "korisnik"; }
function isEducatorRole(role) { return role === "vaspitac"; }
function roleLabel(role) { return role === "admin" ? "Admin" : isUserRole(role) ? "Roditelj" : role === "vaspitac" ? "Vaspitac" : "Gost"; }
function landingPageForRole(role) {
  if (isAdminRole(normalizeRole(role))) return "dodaj.html";
  if (isEducatorRole(normalizeRole(role))) return "vaspitac.html";
  return "index.html";
}
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
  if (isEducatorRole(session.role)) return !["login", "registracija"].includes(targetPage) && (PUBLIC_PAGES.has(targetPage) || EDUCATOR_ONLY_PAGES.has(targetPage));
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

function requestStatusClass(status) {
  switch (String(status || "").toLowerCase()) {
    case "odobren":
      return "status-chip ok";
    case "odbijen":
      return "status-chip danger";
    case "dopuna_dokumentacije":
    case "na_listi_cekanja":
      return "status-chip warn";
    case "podnet":
    case "na_cekanju":
    case "u_obradi":
    case "u_proveri":
      return "status-chip info";
    default:
      return "status-chip";
  }
}
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
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${item.status}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Roditelj nalog: ${item.korisnik_email}</div><div class="muted">Ime roditelja: ${item.ime_roditelja}</div><div class="muted">Ime deteta: ${item.ime_deteta}</div><div class="muted">Broj godina: ${item.broj_godina}</div><div class="muted">Poslato: ${new Date(item.created_at).toLocaleString("sr-RS")}</div>${actionable ? `<div class="card-actions"><button class="btn secondary small" data-request-action="odobri" data-id="${item.id}">Odobri</button><button class="btn danger small" data-request-action="odbij" data-id="${item.id}">Odbij</button></div>` : `<div class="muted">Zahtev je vec obradjen.</div>`}`;
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
async function openMap() {
    try {
        const response = await fetch(`${API_OPEN_DATA}/open-data/vrtici/csv`);
        if (!response.ok) throw new Error("Problem sa mrežom");
        
        const csvText = await response.text();
        const rows = csvText.trim().split('\n');

        if (rows.length > 1) {
            // Detekcija separatora (zarez ili tačka-zarez)
            const separator = csvText.includes(';') ? ';' : ',';
            
            // Uzimamo prvi red podataka (index 1, jer je 0 header)
            const cols = rows[1].split(separator);
            
            // cols[0] je naziv, cols[2] je grad
            const naziv = cols[0] ? cols[0].trim() : "";
            const grad = cols[2] ? cols[2].trim() : "";

            const pretraga = encodeURIComponent(`${naziv} ${grad}`);
            const mapUrl = `https://www.google.com/maps/search/?api=1&query=${pretraga}`;

            window.open(mapUrl, '_blank');
        } else {
            console.warn("CSV nema dovoljno redova za mapu.");
        }
    } catch (err) {
        console.error("Greška pri otvaranju mape:", err);
        // Fallback: otvori bar opštu pretragu za vrtiće u Srbiji
        window.open(`https://www.google.com/maps/search/vrtici+srbija`, '_blank');
    }
}
async function downloadData(resource, format) {
    let url = "";
    
    // Ako je generički download
    if (resource === 'download') {
        url = `${API_OPEN_DATA}/open-data/download`;
    } else {
        // Formiranje putanje prema tvom Go mux-u: /open-data/{vrtici}/{csv}
        url = `${API_OPEN_DATA}/open-data/${resource}/${format}`;
    }

    try {
        console.log(`Preuzimam sa: ${url}`);
        
        // Za CSV/Download najbolje je koristiti direktan link
        if (format === 'csv' || resource === 'download') {
            window.location.href = url;
            return;
        }

        // Za JSON možemo da otvorimo u novom tabu ili uradimo fetch
        const response = await fetch(url);
        if (!response.ok) throw new Error("Servis nije dostupan.");
        
        const blob = await response.blob();
        const downloadUrl = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = downloadUrl;
        a.download = `${resource}_${new Date().getTime()}.${format}`;
        document.body.appendChild(a);
        a.click();
        a.remove();
    } catch (err) {
        alert("Greška pri preuzimanju: " + err.message);
    }
}

async function previewData(resource) {
    const section = document.getElementById('preview-section');
    const container = document.getElementById('preview-container');
    const title = document.getElementById('preview-title');
    
    section.style.display = 'block';
    container.innerHTML = '<p class="muted">Učitavam CSV podatke...</p>';
    title.innerText = `Pregled: ${resource.toUpperCase()}`;

    const url = `${API_OPEN_DATA}/open-data/${resource}/csv`;

    try {
        const response = await fetch(url);
        if (!response.ok) throw new Error("CSV servis nije dostupan.");
        
        const csvText = await response.text();
        const rows = csvText.trim().split('\n');

        if (rows.length < 1) {
            container.innerHTML = '<p class="muted">Fajl je prazan.</p>';
            return;
        }

        const separator = csvText.includes(';') ? ';' : ',';
        let html = '<table class="preview-table">';
        
        // Header
        const headers = rows[0].split(separator);
        html += '<thead><tr>';
        headers.forEach(h => html += `<th>${h.trim()}</th>`);
        
        // DODAJEMO KOLONU ZA AKCIJU SAMO AKO SU VRTICI
        if (resource === 'vrtici') {
            html += '<th>Lokacija</th>';
        }
        html += '</tr></thead><tbody>';

        // Podaci (prvih 10 redova da bi izgledalo bogatije)
        const dataRows = rows.slice(1, 11); 
        dataRows.forEach(row => {
            const columns = row.split(separator);
            html += '<tr>';
            columns.forEach(col => html += `<td>${col.trim()}</td>`);

            // AKO SU VRTICI, DODAJ DUGME KOJE KORISTI NAZIV I GRAD IZ OVOG REDA
            if (resource === 'vrtici') {
                const naziv = columns[0].trim();
                const grad = columns[2].trim();
                const query = encodeURIComponent(`${naziv} ${grad}`);
                const mapUrl = `https://www.google.com/maps/search/?api=1&query=${query}`;
                
                html += `<td>
                            <a href="${mapUrl}" target="_blank" class="btn ghost small" style="padding: 4px 8px; font-size: 11px;">
                                📍 Vidi
                            </a>
                         </td>`;
            }
            html += '</tr>';
        });

        html += '</tbody></table>';
        if (rows.length > 11) {
            html += `<p class="muted" style="margin-top: 1rem;">Prikazano prvih 10 od ${rows.length - 1} zapisa.</p>`;
        }

        container.innerHTML = html;
        section.scrollIntoView({ behavior: 'smooth' });

    } catch (err) {
        console.error(err);
        container.innerHTML = `<p class="error-text">Greška pri čitanju CSV-a: ${err.message}</p>`;
    }
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
  if (!session || !isUserRole(session.role)) throw new Error("Samo ulogovan roditelj moze slati zahtev za upis.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa`, { method: "POST", headers: { ...headers, "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function updateEnrollmentRequestById(id, payload) {
  const session = currentSession();
  if (!session || !isUserRole(session.role)) throw new Error("Samo ulogovan roditelj moze menjati zahtev za upis.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa/${id}/izmeni`, {
    method: "PUT",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

function setRequestEditMode(item) {
  state.editingRequestId = String(item?.id || "");
  if (el.upisFormTitle) el.upisFormTitle.textContent = "Izmeni zahtev za upis";
  if (el.upisSubmitBtn) el.upisSubmitBtn.textContent = "Sačuvaj izmene";
  if (el.upisCancelEdit) el.upisCancelEdit.hidden = false;
  if (!el.upisForm) return;
  el.upisForm.elements["vrtic_id"].value = String(item.vrtic_id || "");
  el.upisForm.elements["ime_roditelja"].value = String(item.ime_roditelja || "");
  el.upisForm.elements["ime_deteta"].value = String(item.ime_deteta || "");
  el.upisForm.elements["broj_godina"].value = Number(item.broj_godina || 0) || "";
  el.upisForm.elements["potvrda_vakcinacije"].checked = !!item.potvrda_vakcinacije;
  el.upisForm.elements["izvod_iz_maticne_knjige"].checked = !!item.izvod_iz_maticne_knjige;
}

function resetRequestEditMode() {
  state.editingRequestId = "";
  if (el.upisFormTitle) el.upisFormTitle.textContent = "Posalji zahtev za upis";
  if (el.upisSubmitBtn) el.upisSubmitBtn.textContent = "Posalji zahtev";
  if (el.upisCancelEdit) el.upisCancelEdit.hidden = true;
}

async function processRequest(id, action, reason = "") {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze obradjivati zahteve.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa/${id}/${action}`, {
    method: "PUT",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({ reason: String(reason || "").trim() }),
  });
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
      setTimeout(() => redirectTo(landingPageForRole(data.role)), 500);
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
  if (el.upisCancelEdit && !el.upisCancelEdit.dataset.bound) {
    el.upisCancelEdit.dataset.bound = "1";
    el.upisCancelEdit.addEventListener("click", () => {
      el.upisForm.reset();
      resetRequestEditMode();
      populateVrticSelect();
      if (el.upisStatus) el.upisStatus.textContent = "";
    });
  }
  el.upisForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const formData = new FormData(el.upisForm);
    const payload = {
      vrtic_id: String(formData.get("vrtic_id") || "").trim(),
      ime_roditelja: String(formData.get("ime_roditelja") || "").trim(),
      ime_deteta: String(formData.get("ime_deteta") || "").trim(),
      broj_godina: Number(formData.get("broj_godina") || 0),
      potvrda_vakcinacije: formData.get("potvrda_vakcinacije") !== null,
      izvod_iz_maticne_knjige: formData.get("izvod_iz_maticne_knjige") !== null,
    };
    el.upisStatus.textContent = state.editingRequestId ? "Cuvam izmene..." : "Saljem zahtev...";
    try {
      const created = state.editingRequestId
        ? await updateEnrollmentRequestById(state.editingRequestId, payload)
        : await createEnrollmentRequest(payload);
      const status = String(created?.status || "").toLowerCase();
      const statusLabel = requestStatusLabel(status);
      const reason = String(created?.reason || "").trim();
      if (status === "na_listi_cekanja") {
        el.upisStatus.textContent = state.editingRequestId
          ? (reason || `Izmene su sačuvane. Trenutni status: ${statusLabel}.`)
          : (reason || `Zahtev je evidentiran. Trenutni status: ${statusLabel}.`);
      } else {
        el.upisStatus.textContent = state.editingRequestId
          ? `Izmene su sačuvane. Trenutni status: ${statusLabel}.`
          : `Zahtev je poslat. Trenutni status: ${statusLabel}.`;
      }
      el.upisForm.reset();
      resetRequestEditMode();
      populateVrticSelect();
    } catch (err) { el.upisStatus.textContent = `Greska: ${err.message || "Neuspesno"}`; }
  });
}

function bindRequestAdminEvents() {
  if (!el.adminRequests || el.adminRequests.dataset.boundRequests) return;
  el.adminRequests.dataset.boundRequests = "1";
  el.adminRequests.addEventListener("click", async (e) => {
    const button = e.target.closest("button[data-request-action]");
    if (!button) return;
    const action = String(button.dataset.requestAction || "").trim();
    let reason = "";
    if (action === "dopuna") {
      reason = window.prompt("Upisi sta nedostaje u dokumentaciji:", "") || "";
      if (!reason.trim()) return;
    }
    if (action === "odbij") {
      reason = window.prompt("Upisi razlog odbijanja:", "") || "";
      if (!reason.trim()) return;
    }
    try {
      await processRequest(button.dataset.id, action, reason);
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
  bindMyRequestEvents();
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
  if (!session || !isUserRole(session.role)) throw new Error("Samo ulogovan roditelj moze da ocenjuje vrtice.");
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
      <div class="muted">Bolja ocena roditelja: ${compareWinner(left, right, leftRating.prosecna_ocena, rightRating.prosecna_ocena, true)}</div>
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
    html += `<div class="muted rating-summary">Ocena roditelja: ${formatRatingSummary(rating)}</div>`;
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

state.konkursi = [];
Object.assign(el, {
  konkursForm: byId("konkurs-form"),
  konkursStatus: byId("konkurs-status"),
  konkursVrticSelect: byId("konkurs-vrtic-id"),
  konkursCards: byId("konkurs-cards"),
  upisKonkursInfo: byId("upis-konkurs-info"),
});

function formatDateTimeLocal(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return date.toLocaleString("sr-RS");
}

function requestStatusLabel(status) {
  switch (String(status || "").toLowerCase()) {
    case "podnet":
    case "na_cekanju":
      return "Podnet";
    case "u_obradi":
    case "u_proveri":
      return "U obradi";
    case "dopuna_dokumentacije":
      return "Dopuna dokumentacije";
    case "na_listi_cekanja":
      return "Na listi cekanja";
    case "odobren":
      return "Odobren";
    case "odbijen":
      return "Odbijen";
    default:
      return status || "Nepoznato";
  }
}

function requestCanDownloadDecision(item) {
  const status = String(item?.status || "").toLowerCase();
  return status === "odobren" || status === "odbijen";
}

function meetingStatusLabel(status) {
  switch (String(status || "").toLowerCase()) {
    case "na_cekanju":
      return "Na cekanju";
    case "zakazan":
    case "prihvacen":
      return "Prihvacen";
    case "odbijen":
      return "Odbijen";
    default:
      return status || "Nepoznato";
  }
}

function meetingStatusClass(status) {
  switch (String(status || "").toLowerCase()) {
    case "zakazan":
    case "prihvacen":
      return "status-chip ok";
    case "odbijen":
      return "status-chip danger";
    case "na_cekanju":
      return "status-chip info";
    default:
      return "status-chip";
  }
}

function meetingMetaHtml(item) {
  const details = [];
  if (item.processed_by) details.push(`<div class="muted">Obradio vaspitac: ${item.processed_by}</div>`);
  if (item.processed_at) details.push(`<div class="muted">Datum odluke: ${formatDateTimeLocal(item.processed_at)}</div>`);
  if (item.reason) details.push(`<div class="muted">Napomena: ${item.reason}</div>`);
  return details.join("");
}

function canEducatorProcessMeeting(item) {
  return String(item?.status || "").toLowerCase() === "na_cekanju";
}

function requestMetaHtml(item) {
  const details = [
    `<div class="muted">Poslato: ${formatDateTimeLocal(item.created_at)}</div>`,
  ];
  if (item.processed_by) details.push(`<div class="muted">Obradio admin: ${item.processed_by}</div>`);
  if (item.processed_at) details.push(`<div class="muted">Datum obrade: ${formatDateTimeLocal(item.processed_at)}</div>`);
  if (item.reason) details.push(`<div class="muted">Napomena: ${item.reason}</div>`);
  return details.join("");
}

function buildAdminRequestActions(item) {
  const status = String(item?.status || "").toLowerCase();
  if (status === "odobren" || status === "odbijen") return `<div class="muted">Zahtev je zavrsen.</div>`;
  const actions = [];
  if (status === "podnet" || status === "na_cekanju" || status === "dopuna_dokumentacije" || status === "na_listi_cekanja") {
    actions.push(`<button class="btn ghost small" data-request-action="obrada" data-id="${item.id}">U obradi</button>`);
  }
  if (status === "podnet" || status === "na_cekanju" || status === "u_obradi" || status === "u_proveri" || status === "dopuna_dokumentacije" || status === "na_listi_cekanja") {
    actions.push(`<button class="btn ghost small" data-request-action="dopuna" data-id="${item.id}">Dopuna</button>`);
    actions.push(`<button class="btn secondary small" data-request-action="odobri" data-id="${item.id}">Odobri</button>`);
    actions.push(`<button class="btn danger small" data-request-action="odbij" data-id="${item.id}">Odbij</button>`);
  }
  return actions.length ? `<div class="card-actions">${actions.join("")}</div>` : `<div class="muted">Zahtev je zavrsen.</div>`;
}

async function downloadRequestDecisionPdf(id) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa/${id}/dokument`, { headers });
  if (!res.ok) throw new Error(await res.text());
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") || "";
  const match = disposition.match(/filename="?([^";]+)"?/i);
  const fileName = match?.[1] || `zahtev-${id}.pdf`;
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  a.remove();
  window.URL.revokeObjectURL(url);
}

function bindMyRequestEvents() {
  if (!el.myRequests || el.myRequests.dataset.boundRequestPdf) return;
  el.myRequests.dataset.boundRequestPdf = "1";
  el.myRequests.addEventListener("click", async (e) => {
    const button = e.target.closest("button[data-request-pdf]");
    if (!button) return;
    const oldLabel = button.textContent;
    button.disabled = true;
    button.textContent = "Preuzimam...";
    try {
      await downloadRequestDecisionPdf(button.dataset.requestPdf);
    } catch (err) {
      window.alert(err.message || "PDF nije dostupan.");
    } finally {
      button.disabled = false;
      button.textContent = oldLabel;
    }
  });
}

function konkursForVrtic(vrticId, preferActive = true) {
  const matches = state.konkursi.filter((item) => String(item.vrtic_id) === String(vrticId));
  if (!matches.length) return null;
  matches.sort((a, b) => new Date(b.datum_pocetka || 0).getTime() - new Date(a.datum_pocetka || 0).getTime());
  if (preferActive) {
    const active = matches.find((item) => item.status === "aktivan");
    if (active) return active;
  }
  return matches[0];
}

function activeKonkursForVrtic(vrticId) {
  return state.konkursi.find((item) => String(item.vrtic_id) === String(vrticId) && item.status === "aktivan") || null;
}

function populateKonkursVrticSelect() {
  if (!el.konkursVrticSelect) return;
  const options = [...state.vrtici]
    .sort((a, b) => String(a.naziv || "").localeCompare(String(b.naziv || ""), "sr"))
    .map((v) => `<option value="${v.id}">${v.naziv} (${v.grad} - ${v.opstina})</option>`)
    .join("");
  el.konkursVrticSelect.innerHTML = `<option value="">Izaberi vrtic</option>${options}`;
}

const __basePopulateVrticSelectForKonkurs = populateVrticSelect;
populateVrticSelect = function() {
  if (!el.vrticSelect) return;
  const available = [...state.vrtici]
    .filter((v) => activeKonkursForVrtic(v.id))
    .sort((a, b) => String(a.naziv || "").localeCompare(String(b.naziv || ""), "sr"));

  el.vrticSelect.innerHTML = `<option value="">Izaberi vrtic sa aktivnim konkursom</option>${available.map((v) => {
    const konkurs = activeKonkursForVrtic(v.id);
    const konkursSeats = Number(konkurs?.slobodna_mesta || 0);
    const vrticSeats = freePlaces(v);
    const waitingList = konkursSeats <= 0 || vrticSeats <= 0;
    const suffix = waitingList ? "lista cekanja" : `slobodna mesta: ${Math.min(konkursSeats, vrticSeats)}`;
    return `<option value="${v.id}">${v.naziv} (${v.grad} - ${v.opstina}) - ${suffix}</option>`;
  }).join("")}`;

  const requestedId = new URLSearchParams(window.location.search).get("vrtic");
  if (requestedId && available.some((v) => String(v.id) === requestedId)) el.vrticSelect.value = requestedId;
  renderUpisKonkursInfo();
};

renderMyRequests = function() {
  if (!el.myRequests) return;
  el.myRequests.innerHTML = "";
  state.mojePrijave.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    const pdfButton = requestCanDownloadDecision(item)
      ? `<div class="card-actions"><button class="btn secondary small" type="button" data-request-pdf="${item.id}">${String(item.status || "").toLowerCase() === "odobren" ? "Preuzmi potvrdu" : "Preuzmi odbijenicu"}</button></div>`
      : "";
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${requestStatusLabel(item.status)}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Roditelj: ${item.ime_roditelja}</div><div class="muted">Dete: ${item.ime_deteta}</div><div class="muted">Broj godina: ${item.broj_godina}</div>${requestMetaHtml(item)}${pdfButton}`;
    el.myRequests.appendChild(card);
  });
  if (!state.mojePrijave.length) el.myRequests.innerHTML = "<div class='card'>Jos nema poslatih zahteva.</div>";
};

renderAdminRequests = function() {
  if (!el.adminRequests) return;
  el.adminRequests.innerHTML = "";
  state.adminPrijave.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    const actions = buildAdminRequestActions(item);
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${requestStatusLabel(item.status)}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Roditelj nalog: ${item.korisnik_email}</div><div class="muted">Ime roditelja: ${item.ime_roditelja}</div><div class="muted">Ime deteta: ${item.ime_deteta}</div><div class="muted">Broj godina: ${item.broj_godina}</div>${requestMetaHtml(item)}${actions}`;
    el.adminRequests.appendChild(card);
  });
  if (!state.adminPrijave.length) el.adminRequests.innerHTML = "<div class='card'>Nema zahteva za upis.</div>";
};

function renderKonkursCards() {
  if (!el.konkursCards) return;
  el.konkursCards.innerHTML = "";
  state.konkursi.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    const closable = item.status === "aktivan" || item.status === "zakazan";
    card.innerHTML = `<div class="${requestStatusClass(item.status === "aktivan" ? "odobren" : item.status === "zatvoren" ? "odbijen" : "u_proveri")}">${item.status}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Period: ${formatDateTimeLocal(item.datum_pocetka)} - ${formatDateTimeLocal(item.datum_zavrsetka)}</div><div class="muted">Mesta na konkursu: ${item.max_mesta}</div><div class="muted">Popunjeno kroz konkurs: ${item.popunjeno}</div><div class="muted">Slobodna mesta na konkursu: ${item.slobodna_mesta}</div>${closable ? `<div class="card-actions"><button class="btn danger small" type="button" data-konkurs-close="${item.id}">Zatvori konkurs</button></div>` : `<div class="muted">Konkurs nije moguce dodatno menjati.</div>`}`;
    el.konkursCards.appendChild(card);
  });
  if (!state.konkursi.length) el.konkursCards.innerHTML = "<div class='card'>Jos nema raspisanih konkursa.</div>";
}

function renderUpisKonkursInfo() {
  if (!el.upisKonkursInfo) return;
  const selectedId = el.vrticSelect?.value || new URLSearchParams(window.location.search).get("vrtic") || "";
  const selectedKonkurs = selectedId ? konkursForVrtic(selectedId) : null;
  const list = selectedKonkurs ? [selectedKonkurs] : state.konkursi.filter((item) => item.status === "aktivan");

  el.upisKonkursInfo.innerHTML = "";
  list.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    const vrtic = state.vrtici.find((row) => String(row.id) === String(item.vrtic_id));
    const vrticSeats = vrtic ? freePlaces(vrtic) : 0;
    const konkursSeats = Number(item.slobodna_mesta || 0);
    const waitingList = konkursSeats <= 0 || vrticSeats <= 0;
    card.innerHTML = `<div class="${requestStatusClass(item.status === "aktivan" ? (waitingList ? "na_listi_cekanja" : "odobren") : "u_obradi")}">${item.status}</div><h3>${item.vrtic_naziv}</h3><div class="muted">Period konkursa: ${formatDateTimeLocal(item.datum_pocetka)} - ${formatDateTimeLocal(item.datum_zavrsetka)}</div><div class="muted">Mesta na konkursu: ${item.max_mesta}</div><div class="muted">Preostalo mesta za prijavu: ${item.slobodna_mesta}</div>${waitingList ? `<div class="muted">Trenutno nema slobodnih mesta. Novi zahtevi idu na listu cekanja.</div>` : ""}`;
    el.upisKonkursInfo.appendChild(card);
  });
  if (!list.length) el.upisKonkursInfo.innerHTML = "<div class='card'>Trenutno nema aktivnih konkursa. Upis je moguc tek kada admin raspise konkurs za odredjeni vrtic.</div>";
}

async function fetchKonkursi() {
  try {
    const res = await fetch(`${API_VRTICI}/konkursi`);
    if (!res.ok) throw new Error("Ne mogu da ucitam konkurse");
    const data = await res.json(); state.konkursi = Array.isArray(data) ? data : [];
    renderAll();
  } catch (_err) {
    state.konkursi = [];
    renderAll();
  }
}

async function createKonkurs(payload) {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze da raspisuje konkurs.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/konkursi`, { method: "POST", headers: { ...headers, "Content-Type": "application/json" }, body: JSON.stringify(payload) });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function closeKonkursById(id) {
  const session = currentSession();
  if (!session || !isAdminRole(session.role)) throw new Error("Samo admin moze da zatvori konkurs.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/konkursi/${id}/zatvori`, { method: "PUT", headers });
  if (!res.ok) throw new Error(await res.text());
}

function bindKonkursEvents() {
  if (el.konkursForm && !el.konkursForm.dataset.bound) {
    el.konkursForm.dataset.bound = "1";
    el.konkursForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      const formData = new FormData(el.konkursForm);
      const payload = {
        vrtic_id: String(formData.get("vrtic_id") || "").trim(),
        datum_pocetka: String(formData.get("datum_pocetka") || "").trim(),
        datum_zavrsetka: String(formData.get("datum_zavrsetka") || "").trim(),
        max_mesta: Number(formData.get("max_mesta") || 0),
      };
      if (el.konkursStatus) el.konkursStatus.textContent = "Raspisujem konkurs...";
      try {
        await createKonkurs(payload);
        if (el.konkursStatus) el.konkursStatus.textContent = "Konkurs je uspesno raspisan.";
        el.konkursForm.reset();
        await fetchKonkursi();
      } catch (err) {
        if (el.konkursStatus) el.konkursStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
      }
    });
  }

  if (el.konkursCards && !el.konkursCards.dataset.bound) {
    el.konkursCards.dataset.bound = "1";
    el.konkursCards.addEventListener("click", async (e) => {
      const button = e.target.closest("button[data-konkurs-close]");
      if (!button) return;
      if (!window.confirm("Zatvori ovaj konkurs?")) return;
      try {
        await closeKonkursById(button.dataset.konkursClose);
        if (el.konkursStatus) el.konkursStatus.textContent = "Konkurs je zatvoren.";
        await fetchKonkursi();
      } catch (err) {
        if (el.konkursStatus) el.konkursStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
      }
    });
  }

  if (el.vrticSelect && !el.vrticSelect.dataset.konkursBound) {
    el.vrticSelect.dataset.konkursBound = "1";
    el.vrticSelect.addEventListener("change", renderUpisKonkursInfo);
  }
}

const __vrticCardHTMLWithRatings = vrticCardHTML;
vrticCardHTML = function(v, idx, mode = "public") {
  const html = __vrticCardHTMLWithRatings(v, idx, mode);
  const konkurs = konkursForVrtic(v.id, mode !== "admin");
  if (!konkurs) return `${html}<div class="muted">Konkurs: trenutno nije raspisan.</div>`;
  return `${html}<div class="muted">Konkurs: ${konkurs.status} | ${formatDateTimeLocal(konkurs.datum_pocetka)} - ${formatDateTimeLocal(konkurs.datum_zavrsetka)}</div><div class="muted">Mesta na konkursu: ${konkurs.slobodna_mesta}/${konkurs.max_mesta}</div>`;
};

const __renderAllWithCompareAndRatings = renderAll;
renderAll = function() {
  __renderAllWithCompareAndRatings();
  populateKonkursVrticSelect();
  renderKonkursCards();
  renderUpisKonkursInfo();
};

async function initKonkursFeature() {
  bindKonkursEvents();
  await fetchKonkursi();
  if (el.myRequests) await fetchMyRequests();
  if (el.adminRequests) await fetchAdminRequests();
}

initKonkursFeature();

Object.assign(state, {
  assignments: [],
  educators: [],
  parentEducatorOptions: [],
  myMeetingsData: [],
  myNotificationsData: [],
  educatorChildrenData: [],
  educatorMeetingsData: [],
});

function boolLabel(value) {
  return value ? "Priloženo" : "Nedostaje";
}

function requestDocumentsHtml(item) {
  return `<div class="muted">Potvrda o vakcinaciji: ${boolLabel(item.potvrda_vakcinacije)}</div>
    <div class="muted">Izvod iz matične knjige rođenih: ${boolLabel(item.izvod_iz_maticne_knjige)}</div>`;
}

async function updateRequestDocumentsPayload(id, payload) {
  const session = currentSession();
  if (!session || !isUserRole(session.role)) throw new Error("Samo roditelj može da dopuni dokumentaciju.");
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/zahtevi-upisa/${id}/dokumenta`, {
    method: "PUT",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(await res.text());
}

renderMyRequests = function() {
  if (!el.myRequests) return;
  el.myRequests.innerHTML = "";
  state.mojePrijave.forEach((item) => {
    const canEditRequest = String(item.status || "").toLowerCase() === "dopuna_dokumentacije";
    const pdfButton = requestCanDownloadDecision(item)
      ? `<button class="btn secondary small" type="button" data-request-pdf="${item.id}">${String(item.status || "").toLowerCase() === "odobren" ? "Preuzmi potvrdu" : "Preuzmi odbijenicu"}</button>`
      : "";
    const editButton = canEditRequest
      ? `<button class="btn ghost small" type="button" data-request-edit="${item.id}">Izmeni zahtev</button>`
      : "";
    const actions = pdfButton || editButton ? `<div class="card-actions">${pdfButton}${editButton}</div>` : "";
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${requestStatusLabel(item.status)}</div>
      <h3>${item.vrtic_naziv}</h3>
      <div class="muted">Roditelj: ${item.ime_roditelja}</div>
      <div class="muted">Dete: ${item.ime_deteta}</div>
      <div class="muted">Broj godina: ${item.broj_godina}</div>
      ${requestDocumentsHtml(item)}
      ${requestMetaHtml(item)}
      ${actions}`;
    el.myRequests.appendChild(card);
  });
  if (!state.mojePrijave.length) el.myRequests.innerHTML = "<div class='card'>Još nema poslatih zahteva.</div>";
};

renderAdminRequests = function() {
  if (!el.adminRequests) return;
  el.adminRequests.innerHTML = "";
  state.adminPrijave.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="${requestStatusClass(item.status)}">${requestStatusLabel(item.status)}</div>
      <h3>${item.vrtic_naziv}</h3>
      <div class="muted">Roditelj nalog: ${item.korisnik_email}</div>
      <div class="muted">Ime roditelja: ${item.ime_roditelja}</div>
      <div class="muted">Ime deteta: ${item.ime_deteta}</div>
      <div class="muted">Broj godina: ${item.broj_godina}</div>
      ${requestDocumentsHtml(item)}
      ${requestMetaHtml(item)}
      ${buildAdminRequestActions(item)}`;
    el.adminRequests.appendChild(card);
  });
  if (!state.adminPrijave.length) el.adminRequests.innerHTML = "<div class='card'>Nema zahteva za upis.</div>";
};

function bindMyRequestDocumentEvents() {
  if (!el.myRequests || el.myRequests.dataset.boundDocs) return;
  el.myRequests.dataset.boundDocs = "1";
  el.myRequests.addEventListener("click", (e) => {
    const button = e.target.closest("button[data-request-edit]");
    if (!button) return;
    const item = state.mojePrijave.find((entry) => entry.id === button.dataset.requestEdit);
    if (!item) return;
    setRequestEditMode(item);
    if (el.upisStatus) el.upisStatus.textContent = "Izmeni podatke i ponovo pošalji zahtev.";
    window.scrollTo({ top: 0, behavior: "smooth" });
  });
}

function populateAssignmentVrticSelect() {
  if (!el.assignmentVrticSelect) return;
  el.assignmentVrticSelect.innerHTML = `<option value="">Izaberi vrtic</option>${state.vrtici
    .map((v) => `<option value="${v.id}">${v.naziv} (${v.grad} - ${v.opstina})</option>`)
    .join("")}`;
}

function populateEducatorSelect() {
  if (!el.assignmentEducatorSelect) return;
  el.assignmentEducatorSelect.innerHTML = `<option value="">Izaberi vaspitaca</option>${state.educators
    .map((item) => `<option value="${item.email}">${item.email}</option>`)
    .join("")}`;
}

function renderAssignments() {
  if (!el.assignmentCards) return;
  el.assignmentCards.innerHTML = "";
  state.assignments.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<h3>${item.vrtic_naziv}</h3>
      <div class="muted">Vaspitač: ${item.vaspitac_email}</div>
      <div class="muted">Dodeljeno: ${formatDateTimeLocal(item.created_at)}</div>
      <div class="card-actions"><button class="btn danger small" type="button" data-assignment-delete="${item.id}">Ukloni raspored</button></div>`;
    el.assignmentCards.appendChild(card);
  });
  if (!state.assignments.length) el.assignmentCards.innerHTML = "<div class='card'>Još nema raspoređenih vaspitača.</div>";
}

async function fetchEducators() {
  if (!el.assignmentEducatorSelect) return;
  const headers = authHeaders();
  if (!headers) {
    state.educators = [];
    populateEducatorSelect();
    return;
  }
  try {
    const res = await fetch(`${API_AUTH}/auth/users?role=vaspitac`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.educators = await res.json();
    populateEducatorSelect();
  } catch (_err) {
    state.educators = [];
    populateEducatorSelect();
  }
}

async function fetchAssignments() {
  if (!el.assignmentCards) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/rasporedi-vaspitaca`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.assignments = await res.json();
    renderAssignments();
  } catch (err) {
    el.assignmentCards.innerHTML = `<div class='card'>${err.message || "Ne mogu da učitam rasporede."}</div>`;
  }
}

async function createAssignmentRequest(payload) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/rasporedi-vaspitaca`, {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function deleteAssignmentRequest(id) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/rasporedi-vaspitaca/${id}`, { method: "DELETE", headers });
  if (!res.ok) throw new Error(await res.text());
}

function bindAssignmentEvents() {
  if (el.assignmentForm && !el.assignmentForm.dataset.bound) {
    el.assignmentForm.dataset.bound = "1";
    el.assignmentForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      const formData = new FormData(el.assignmentForm);
      const payload = {
        vrtic_id: String(formData.get("vrtic_id") || "").trim(),
        vaspitac_email: String(formData.get("vaspitac_email") || "").trim(),
      };
      if (el.assignmentStatus) el.assignmentStatus.textContent = "Čuvam raspored...";
      try {
        await createAssignmentRequest(payload);
        if (el.assignmentStatus) el.assignmentStatus.textContent = "Raspored vaspitača je sačuvan.";
        el.assignmentForm.reset();
        await fetchAssignments();
      } catch (err) {
        if (el.assignmentStatus) el.assignmentStatus.textContent = `Greška: ${err.message || "Neuspešno"}`;
      }
    });
  }

  if (el.assignmentCards && !el.assignmentCards.dataset.bound) {
    el.assignmentCards.dataset.bound = "1";
    el.assignmentCards.addEventListener("click", async (e) => {
      const button = e.target.closest("button[data-assignment-delete]");
      if (!button) return;
      if (!window.confirm("Ukloni ovaj raspored vaspitača?")) return;
      try {
        await deleteAssignmentRequest(button.dataset.assignmentDelete);
        if (el.assignmentStatus) el.assignmentStatus.textContent = "Raspored je uklonjen.";
        await fetchAssignments();
      } catch (err) {
        if (el.assignmentStatus) el.assignmentStatus.textContent = `Greška: ${err.message || "Neuspešno"}`;
      }
    });
  }
}

function populateMeetingRequestOptions() {
  if (!el.sastanakZahtevSelect) return;
  el.sastanakZahtevSelect.innerHTML = `<option value="">Izaberi dete</option>${state.parentEducatorOptions
    .map((item) => `<option value="${item.zahtev_id}">${item.ime_deteta} — ${item.vrtic_naziv}</option>`)
    .join("")}`;
  syncMeetingEducatorOptions();
}

function syncMeetingEducatorOptions() {
  if (!el.sastanakEducatorSelect || !el.sastanakZahtevSelect) return;
  const selected = state.parentEducatorOptions.find((item) => String(item.zahtev_id) === String(el.sastanakZahtevSelect.value));
  const educators = selected?.vaspitaci || [];
  el.sastanakEducatorSelect.innerHTML = `<option value="">Izaberi vaspitača</option>${educators
    .map((email) => `<option value="${email}">${email}</option>`)
    .join("")}`;
}

function renderParentEducatorOptions() {
  if (!el.roditeljOptions) return;
  el.roditeljOptions.innerHTML = "";
  state.parentEducatorOptions.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<h3>${item.ime_deteta}</h3>
      <div class="muted">Vrtić: ${item.vrtic_naziv}</div>
      <div class="muted">Vaspitači: ${item.vaspitaci.join(", ")}</div>`;
    el.roditeljOptions.appendChild(card);
  });
  if (!state.parentEducatorOptions.length) el.roditeljOptions.innerHTML = "<div class='card'>Još nema dodeljenih vaspitača za odobrene upise.</div>";
}

function renderMyMeetingsList() {
  if (!el.myMeetings) return;
  el.myMeetings.innerHTML = "";
  state.myMeetingsData.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="${meetingStatusClass(item.status)}">${meetingStatusLabel(item.status)}</div>
      <h3>${item.ime_deteta}</h3>
      <div class="muted">Vrtic: ${item.vrtic_naziv}</div>
      <div class="muted">Vaspitac: ${item.vaspitac_email}</div>
      <div class="muted">Termin: ${formatDateTimeLocal(item.termin)}</div>
      ${item.napomena ? `<div class="muted">Napomena roditelja: ${item.napomena}</div>` : ""}
      ${meetingMetaHtml(item)}`;
    el.myMeetings.appendChild(card);
  });
  if (!state.myMeetingsData.length) el.myMeetings.innerHTML = "<div class='card'>Jos nema poslatih zahteva za sastanak.</div>";
}

function renderMyNotificationsList() {
  if (!el.myNotifications) return;
  el.myNotifications.innerHTML = "";
  state.myNotificationsData.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="status-chip warn">Obaveštenje</div>
      <h3>${item.ime_deteta}</h3>
      <div class="muted">Vrtić: ${item.vrtic_naziv}</div>
      <div class="muted">Vaspitač: ${item.vaspitac_email}</div>
      <div class="muted">Poslato: ${formatDateTimeLocal(item.created_at)}</div>
      <div class="muted">Poruka: ${item.poruka}</div>`;
    el.myNotifications.appendChild(card);
  });
  if (!state.myNotificationsData.length) el.myNotifications.innerHTML = "<div class='card'>Još nema obaveštenja od vaspitača.</div>";
}

async function fetchParentEducatorOptions() {
  if (!el.roditeljOptions && !el.sastanakZahtevSelect) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/roditelj/vaspitaci`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.parentEducatorOptions = await res.json();
    renderParentEducatorOptions();
    populateMeetingRequestOptions();
  } catch (err) {
    if (el.roditeljOptions) el.roditeljOptions.innerHTML = `<div class='card'>${err.message || "Ne mogu da učitam vaspitače."}</div>`;
  }
}

async function createMeetingRequest(payload) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/sastanci`, {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function updateMeetingDecisionRequest(id, action, reason = "") {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/vaspitac/sastanci/${id}/${action}`, {
    method: "PUT",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  });
  if (!res.ok) throw new Error(await res.text());
}

async function fetchMyMeetingsList() {
  if (!el.myMeetings) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/sastanci/moji`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.myMeetingsData = await res.json();
    renderMyMeetingsList();
  } catch (err) {
    el.myMeetings.innerHTML = `<div class='card'>${err.message || "Ne mogu da učitam sastanke."}</div>`;
  }
}

async function fetchMyNotificationsList() {
  if (!el.myNotifications) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/obavestenja/moja`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.myNotificationsData = await res.json();
    renderMyNotificationsList();
  } catch (err) {
    el.myNotifications.innerHTML = `<div class='card'>${err.message || "Ne mogu da učitam obaveštenja."}</div>`;
  }
}

function bindMeetingEvents() {
  if (el.sastanakZahtevSelect && !el.sastanakZahtevSelect.dataset.bound) {
    el.sastanakZahtevSelect.dataset.bound = "1";
    el.sastanakZahtevSelect.addEventListener("change", syncMeetingEducatorOptions);
  }
  if (el.sastanakForm && !el.sastanakForm.dataset.bound) {
    el.sastanakForm.dataset.bound = "1";
    el.sastanakForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      const formData = new FormData(el.sastanakForm);
      const payload = {
        zahtev_id: String(formData.get("zahtev_id") || "").trim(),
        vaspitac_email: String(formData.get("vaspitac_email") || "").trim(),
        termin: String(formData.get("termin") || "").trim(),
        napomena: String(formData.get("napomena") || "").trim(),
      };
      if (el.sastanakStatus) el.sastanakStatus.textContent = "Saljem zahtev za sastanak...";
      try {
        await createMeetingRequest(payload);
        if (el.sastanakStatus) el.sastanakStatus.textContent = "Zahtev za sastanak je poslat vaspitacu na potvrdu.";
        el.sastanakForm.reset();
        syncMeetingEducatorOptions();
        await fetchMyMeetingsList();
      } catch (err) {
        if (el.sastanakStatus) el.sastanakStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
      }
    });
  }
}

function populateEducatorChildSelect() {
  if (!el.educatorChildSelect) return;
  el.educatorChildSelect.innerHTML = `<option value="">Izaberi dete</option>${state.educatorChildrenData
    .map((item) => `<option value="${item.id}">${item.ime_deteta} — ${item.vrtic_naziv}</option>`)
    .join("")}`;
}

function renderEducatorChildrenCards() {
  if (!el.educatorChildren) return;
  el.educatorChildren.innerHTML = "";
  state.educatorChildrenData.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<h3>${item.ime_deteta}</h3>
      <div class="muted">Roditelj: ${item.ime_roditelja}</div>
      <div class="muted">Roditelj nalog: ${item.korisnik_email}</div>
      <div class="muted">Vrtić: ${item.vrtic_naziv}</div>`;
    el.educatorChildren.appendChild(card);
  });
  if (!state.educatorChildrenData.length) el.educatorChildren.innerHTML = "<div class='card'>Nema odobrenih upisa u tvojim vrtićima.</div>";
}

function renderEducatorMeetingsCards() {
  if (!el.educatorMeetings) return;
  el.educatorMeetings.innerHTML = "";
  state.educatorMeetingsData.forEach((item) => {
    const actions = canEducatorProcessMeeting(item)
      ? `<div class="card-actions"><button class="btn secondary small" type="button" data-meeting-action="prihvati" data-meeting-id="${item.id}">Prihvati</button><button class="btn danger small" type="button" data-meeting-action="odbij" data-meeting-id="${item.id}">Odbij</button></div>`
      : "";
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<div class="${meetingStatusClass(item.status)}">${meetingStatusLabel(item.status)}</div>
      <h3>${item.ime_deteta}</h3>
      <div class="muted">Roditelj: ${item.roditelj_email}</div>
      <div class="muted">Vrtic: ${item.vrtic_naziv}</div>
      <div class="muted">Termin: ${formatDateTimeLocal(item.termin)}</div>
      ${item.napomena ? `<div class="muted">Napomena roditelja: ${item.napomena}</div>` : ""}
      ${meetingMetaHtml(item)}
      ${actions}`;
    el.educatorMeetings.appendChild(card);
  });
  if (!state.educatorMeetingsData.length) el.educatorMeetings.innerHTML = "<div class='card'>Jos nema zahteva za sastanak.</div>";
}

async function fetchEducatorChildrenData() {
  if (!el.educatorChildren && !el.educatorChildSelect) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/vaspitac/deca`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.educatorChildrenData = await res.json();
    renderEducatorChildrenCards();
    populateEducatorChildSelect();
  } catch (err) {
    if (el.educatorChildren) el.educatorChildren.innerHTML = `<div class='card'>${err.message || "Ne mogu da učitam podatke o deci."}</div>`;
  }
}

async function fetchEducatorMeetingsData() {
  if (!el.educatorMeetings) return;
  const headers = authHeaders();
  if (!headers) return;
  try {
    const res = await fetch(`${API_VRTICI}/vaspitac/sastanci`, { headers });
    if (!res.ok) throw new Error(await res.text());
    state.educatorMeetingsData = await res.json();
    renderEducatorMeetingsCards();
  } catch (err) {
    el.educatorMeetings.innerHTML = `<div class='card'>${err.message || "Ne mogu da učitam sastanke."}</div>`;
  }
}

async function createEducatorNoticeRequest(payload) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");
  const res = await fetch(`${API_VRTICI}/vaspitac/obavestenja`, {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

function bindEducatorEvents() {
  if (el.educatorNoticeForm && !el.educatorNoticeForm.dataset.bound) {
    el.educatorNoticeForm.dataset.bound = "1";
    el.educatorNoticeForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      const formData = new FormData(el.educatorNoticeForm);
      const payload = {
        zahtev_id: String(formData.get("zahtev_id") || "").trim(),
        poruka: String(formData.get("poruka") || "").trim(),
      };
      if (el.educatorNoticeStatus) el.educatorNoticeStatus.textContent = "Saljem obavestenje...";
      try {
        await createEducatorNoticeRequest(payload);
        if (el.educatorNoticeStatus) el.educatorNoticeStatus.textContent = "Obavestenje je poslato roditelju.";
        el.educatorNoticeForm.reset();
      } catch (err) {
        if (el.educatorNoticeStatus) el.educatorNoticeStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
      }
    });
  }

  if (el.educatorMeetings && !el.educatorMeetings.dataset.boundActions) {
    el.educatorMeetings.dataset.boundActions = "1";
    el.educatorMeetings.addEventListener("click", async (e) => {
      const button = e.target.closest("button[data-meeting-action]");
      if (!button) return;
      const action = String(button.dataset.meetingAction || "").trim();
      const id = String(button.dataset.meetingId || "").trim();
      let reason = "";
      if (action === "odbij") {
        reason = window.prompt("Unesi razlog odbijanja sastanka:", "") || "";
        if (!reason.trim()) {
          if (el.educatorNoticeStatus) el.educatorNoticeStatus.textContent = "Odbijanje sastanka zahteva razlog.";
          return;
        }
      }
      if (el.educatorNoticeStatus) {
        el.educatorNoticeStatus.textContent = action === "prihvati" ? "Potvrdjujem sastanak..." : "Odbijam sastanak...";
      }
      try {
        await updateMeetingDecisionRequest(id, action, reason.trim());
        if (el.educatorNoticeStatus) {
          el.educatorNoticeStatus.textContent = action === "prihvati" ? "Sastanak je prihvacen." : "Sastanak je odbijen.";
        }
        await fetchEducatorMeetingsData();
      } catch (err) {
        if (el.educatorNoticeStatus) el.educatorNoticeStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
      }
    });
  }
}

const __renderAllWithAssignmentsAndMeetings = renderAll;
renderAll = function() {
  __renderAllWithAssignmentsAndMeetings();
  populateAssignmentVrticSelect();
  populateEducatorSelect();
  renderAssignments();
  renderParentEducatorOptions();
  populateMeetingRequestOptions();
  renderMyMeetingsList();
  renderMyNotificationsList();
  renderEducatorChildrenCards();
  populateEducatorChildSelect();
  renderEducatorMeetingsCards();
};

async function initExtendedEnrollmentFeatures() {
  bindMyRequestDocumentEvents();
  bindAssignmentEvents();
  bindMeetingEvents();
  bindEducatorEvents();

  if (el.assignmentForm || el.assignmentCards) {
    await fetchEducators();
    await fetchAssignments();
  }
  if (el.roditeljOptions || el.sastanakForm || el.myMeetings || el.myNotifications) {
    await fetchParentEducatorOptions();
    await fetchMyMeetingsList();
    await fetchMyNotificationsList();
  }
  if (el.educatorChildren || el.educatorMeetings || el.educatorNoticeForm) {
    await fetchEducatorChildrenData();
    await fetchEducatorMeetingsData();
  }
}

initExtendedEnrollmentFeatures();

