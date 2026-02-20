const API_VRTICI = "http://localhost:8081";
const API_AUTH = "http://localhost:8083";

const state = {
  vrtici: [],
  kriticni: [],
  opstinaReport: [],
  filterTip: "",
  filterGrad: "",
  filterOpstina: "",
  search: "",
  sortMode: "naziv",
  editingVrticId: "",
};

function byId(id) {
  return document.getElementById(id);
}

const el = {
  cards: byId("cards"),
  manageCards: byId("manage-cards"),
  criticalCards: byId("critical-cards"),
  opstinaReport: byId("opstina-report"),

  filterTip: byId("filter-tip"),
  filterGrad: byId("filter-grad"),
  filterOpstina: byId("filter-opstina"),
  filterReset: byId("filter-reset"),
  search: byId("search"),
  sortMode: byId("sort-mode"),

  statTotal: byId("stat-total"),
  statPublic: byId("stat-public"),
  statPrivate: byId("stat-private"),
  statOccupancy: byId("stat-occupancy"),

  serviceStatus: byId("service-status"),
  apiBase: byId("api-base"),

  vrticForm: byId("vrtic-form"),
  vrticFormStatus: byId("form-status"),
  vrticFormTitle: byId("vrtic-form-title"),
  vrticIdInput: byId("vrtic-id"),
  saveVrticBtn: byId("save-vrtic-btn"),
  cancelEditBtn: byId("cancel-edit"),

  loginForm: byId("login-form"),
  loginStatus: byId("login-status"),
  userInfo: byId("user-info"),

  registerForm: byId("register-form"),
  registerStatus: byId("register-status"),

  downloadReport: byId("download-opstina-report"),
  reportStatus: byId("report-status"),

  profileEmail: byId("profile-email"),
  profileRole: byId("profile-role"),
  profileCreated: byId("profile-created"),
  profileStatus: byId("profile-status"),
  profileRefresh: byId("profile-refresh"),
  profileLogout: byId("profile-logout"),
};

const page = document.body?.dataset?.page || "";

const tokenStore = {
  get access() {
    return localStorage.getItem("access_token");
  },
  set(token) {
    localStorage.setItem("access_token", token);
  },
  clear() {
    localStorage.removeItem("access_token");
  },
};

function authHeaders() {
  const token = tokenStore.access;
  if (!token) return null;
  return {
    Authorization: `Bearer ${token}`,
  };
}

function decodeTokenPayload() {
  const token = tokenStore.access;
  if (!token) return null;

  try {
    const payload = token.split(".")[1];
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(json);
  } catch (_err) {
    return null;
  }
}

function setNavActive() {
  document.querySelectorAll("[data-nav]").forEach((link) => {
    if (link.dataset.nav === page) {
      link.classList.add("active-nav");
    }
  });
}

function uniqueValues(key) {
  return Array.from(new Set(state.vrtici.map((v) => v[key]).filter(Boolean))).sort();
}

function applyFilters(list) {
  return list
    .filter((v) => (state.filterTip ? v.tip === state.filterTip : true))
    .filter((v) => (state.filterGrad ? v.grad === state.filterGrad : true))
    .filter((v) => (state.filterOpstina ? v.opstina === state.filterOpstina : true))
    .filter((v) => {
      if (!state.search) return true;
      return (v.naziv || "").toLowerCase().includes(state.search.toLowerCase());
    });
}

function freePlaces(v) {
  if (typeof v.slobodna_mesta === "number") return v.slobodna_mesta;
  return Number(v.max_kapacitet || 0) - Number(v.trenutno_upisano || 0);
}

function getDisplayedVrtici() {
  const filtered = applyFilters([...state.vrtici]);

  if (state.sortMode === "slobodna_mesta") {
    filtered.sort((a, b) => freePlaces(b) - freePlaces(a));
  } else {
    filtered.sort((a, b) => (a.naziv || "").localeCompare(b.naziv || "", "sr"));
  }

  return filtered;
}

function renderFilters() {
  if (!el.filterGrad || !el.filterOpstina) return;

  const grads = uniqueValues("grad");
  const opstine = uniqueValues("opstina");

  el.filterGrad.innerHTML = `<option value="">Sve</option>${grads
    .map((g) => `<option value="${g}">${g}</option>`)
    .join("")}`;

  el.filterOpstina.innerHTML = `<option value="">Sve</option>${opstine
    .map((o) => `<option value="${o}">${o}</option>`)
    .join("")}`;

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

function vrticCardHTML(v, idx, includeActions) {
  const pct =
    Number(v.max_kapacitet || 0) > 0
      ? Math.min(100, Math.round((Number(v.trenutno_upisano || 0) / Number(v.max_kapacitet || 1)) * 100))
      : 0;

  const rankHTML =
    state.sortMode === "slobodna_mesta"
      ? `<div class="rank-badge">Rang #${idx + 1}</div>`
      : "";

  const actionsHTML = includeActions
    ? `<div class="card-actions">
         <button class="btn ghost small" data-action="edit" data-id="${v.id}">Izmeni</button>
         <button class="btn danger small" data-action="delete" data-id="${v.id}">Obrisi</button>
       </div>`
    : "";

  return `
    ${rankHTML}
    <div class="badge">${v.tip || "n/a"}</div>
    <h3>${v.naziv || "Bez naziva"}</h3>
    <div><strong>${v.grad || ""}</strong> � ${v.opstina || ""}</div>
    <div class="progress"><span style="width:${pct}%"></span></div>
    <div class="muted">${v.trenutno_upisano || 0} / ${v.max_kapacitet || 0} upisano</div>
    <div class="muted">Slobodna mesta: ${freePlaces(v)}</div>
    <div class="muted">${v.kriticno ? "Kriticno popunjen" : "Stabilno"}</div>
    ${actionsHTML}
  `;
}

function renderCards() {
  if (!el.cards) return;

  const displayed = getDisplayedVrtici();
  el.cards.innerHTML = "";

  displayed.forEach((v, idx) => {
    const card = document.createElement("article");
    card.className = "card";
    card.style.animationDelay = `${idx * 0.04}s`;
    card.innerHTML = vrticCardHTML(v, idx, false);
    el.cards.appendChild(card);
  });

  if (displayed.length === 0) {
    el.cards.innerHTML = "<div class='card'>Nema rezultata za izabrane filtere.</div>";
  }
}

function renderManageCards() {
  if (!el.manageCards) return;

  const displayed = getDisplayedVrtici();
  el.manageCards.innerHTML = "";

  displayed.forEach((v, idx) => {
    const card = document.createElement("article");
    card.className = "card";
    card.style.animationDelay = `${idx * 0.03}s`;
    card.innerHTML = vrticCardHTML(v, idx, true);
    el.manageCards.appendChild(card);
  });

  if (displayed.length === 0) {
    el.manageCards.innerHTML = "<div class='card'>Nema vrtica u bazi.</div>";
  }
}

function renderCriticalCards() {
  if (!el.criticalCards) return;

  el.criticalCards.innerHTML = "";
  state.kriticni.forEach((v) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <div class="badge">${v.tip || "n/a"}</div>
      <h3>${v.naziv || "Bez naziva"}</h3>
      <div><strong>${v.grad || ""}</strong> � ${v.opstina || ""}</div>
      <div class="muted">Popunjenost: ${Math.round(Number(v.popunjenost || 0) * 100)}%</div>
      <div class="muted">Slobodna mesta: ${freePlaces(v)}</div>
    `;
    el.criticalCards.appendChild(card);
  });

  if (state.kriticni.length === 0) {
    el.criticalCards.innerHTML = "<div class='card'>Nema kriticno popunjenih vrtica.</div>";
  }
}

function renderOpstinaReport() {
  if (!el.opstinaReport) return;

  el.opstinaReport.innerHTML = "";
  state.opstinaReport.forEach((row) => {
    const pct = Math.round(Number(row.popunjenost || 0) * 100);
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <h3>${row.opstina || "Nepoznata"}</h3>
      <div class="muted">Broj vrtica: ${row.broj_vrtica || 0}</div>
      <div class="muted">Kapacitet: ${row.ukupan_kapacitet || 0}</div>
      <div class="muted">Upisano: ${row.ukupno_upisano || 0}</div>
      <div class="muted">Popunjenost: ${pct}%</div>
    `;
    el.opstinaReport.appendChild(card);
  });

  if (state.opstinaReport.length === 0) {
    el.opstinaReport.innerHTML = "<div class='card'>Nema podataka za izvestaj.</div>";
  }
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
  renderFilters();
  renderStats();
  renderCards();
  renderManageCards();
  renderCriticalCards();
  renderOpstinaReport();
}

async function fetchVrtici() {
  try {
    const url = new URL(`${API_BASE}/vrtici`);
    if (state.sortMode === "slobodna_mesta") {
      url.searchParams.set("sort", "slobodna_mesta");
    }

    const res = await fetch(url);
    if (!res.ok) throw new Error("API error");

    const data = await res.json();
    state.vrtici = Array.isArray(data) ? data : [];

    if (el.serviceStatus) {
      el.serviceStatus.textContent = "Online";
      el.serviceStatus.style.background = "rgba(38, 208, 206, 0.2)";
    }

    renderAll();
  } catch (_err) {
    if (el.serviceStatus) {
      el.serviceStatus.textContent = "Offline";
      el.serviceStatus.style.background = "rgba(249, 72, 72, 0.25)";
    }
    if (el.cards) {
      el.cards.innerHTML = "<div class='card'>Ne mogu da se povezem na servis.</div>";
    }
    if (el.manageCards) {
      el.manageCards.innerHTML = "<div class='card'>Ne mogu da ucitam podatke.</div>";
    }
  }
}

async function fetchKriticni() {
  if (!el.criticalCards) return;
  try {
    const res = await fetch(`${API_VRTICI}/vrtici/kriticni`);
    if (!res.ok) throw new Error("API error");
    const data = await res.json();
    state.kriticni = Array.isArray(data) ? data : [];
    renderCriticalCards();
  } catch (_err) {
    el.criticalCards.innerHTML = "<div class='card'>Ne mogu da ucitam kriticne vrtice.</div>";
  }
}

async function fetchOpstinaReportJson() {
  if (!el.opstinaReport) return;
  try {
    const res = await fetch(`${API_VRTICI}/vrtici/izvestaj/opstina`);
    if (!res.ok) throw new Error("API error");
    const data = await res.json();
    state.opstinaReport = Array.isArray(data) ? data : [];
    renderOpstinaReport();
  } catch (_err) {
    el.opstinaReport.innerHTML = "<div class='card'>Ne mogu da ucitam izvestaj po opstini.</div>";
  }
}

async function createVrtic(payload) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");

  const res = await fetch(`${API_VRTICI}/vrtici`, {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) throw new Error(await res.text());
}

async function updateVrticById(id, payload) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");

  const res = await fetch(`${API_VRTICI}/vrtici/${id}`, {
    method: "PUT",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) throw new Error(await res.text());
}

async function deleteVrticById(id) {
  const headers = authHeaders();
  if (!headers) throw new Error("Prvo se uloguj.");

  const res = await fetch(`${API_VRTICI}/vrtici/${id}`, {
    method: "DELETE",
    headers,
  });

  if (!res.ok) throw new Error(await res.text());
}

function bindListEvents() {
  if (el.filterTip) {
    el.filterTip.addEventListener("change", (e) => {
      state.filterTip = e.target.value;
      renderCards();
      renderManageCards();
    });
  }

  if (el.filterGrad) {
    el.filterGrad.addEventListener("change", (e) => {
      state.filterGrad = e.target.value;
      renderCards();
      renderManageCards();
    });
  }

  if (el.filterOpstina) {
    el.filterOpstina.addEventListener("change", (e) => {
      state.filterOpstina = e.target.value;
      renderCards();
      renderManageCards();
    });
  }

  if (el.search) {
    el.search.addEventListener("input", (e) => {
      state.search = e.target.value;
      renderCards();
      renderManageCards();
    });
  }

  if (el.sortMode) {
    el.sortMode.addEventListener("change", (e) => {
      state.sortMode = e.target.value;
      fetchVrtici();
    });
  }

  if (el.filterReset) {
    el.filterReset.addEventListener("click", () => {
      state.filterTip = "";
      state.filterGrad = "";
      state.filterOpstina = "";
      state.search = "";
      state.sortMode = "naziv";

      if (el.search) el.search.value = "";
      if (el.sortMode) el.sortMode.value = "naziv";
      if (el.filterTip) el.filterTip.value = "";

      fetchVrtici();
    });
  }
}

function bindCrudEvents() {
  if (el.cancelEditBtn) {
    el.cancelEditBtn.style.display = "none";
    el.cancelEditBtn.addEventListener("click", () => {
      resetEditMode();
      if (el.vrticForm) el.vrticForm.reset();
      if (el.vrticFormStatus) el.vrticFormStatus.textContent = "";
    });
  }

  if (el.vrticForm) {
    el.vrticForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      if (!el.vrticFormStatus) return;

      el.vrticFormStatus.textContent = "Saljem...";

      const formData = new FormData(el.vrticForm);
      const payload = {
        naziv: String(formData.get("naziv") || "").trim(),
        tip: String(formData.get("tip") || "").trim(),
        grad: String(formData.get("grad") || "").trim(),
        opstina: String(formData.get("opstina") || "").trim(),
        max_kapacitet: Number(formData.get("max_kapacitet") || 0),
        trenutno_upisano: Number(formData.get("trenutno_upisano") || 0),
      };

      try {
        if (state.editingVrticId) {
          await updateVrticById(state.editingVrticId, payload);
          el.vrticFormStatus.textContent = "Vrtic je izmenjen.";
        } else {
          await createVrtic(payload);
          el.vrticFormStatus.textContent = "Vrtic je dodat.";
        }

        el.vrticForm.reset();
        resetEditMode();
        await fetchVrtici();
      } catch (err) {
        el.vrticFormStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
      }
    });
  }

  if (el.manageCards) {
    el.manageCards.addEventListener("click", async (event) => {
      const target = event.target;
      if (!(target instanceof HTMLElement)) return;

      const action = target.dataset.action;
      const id = target.dataset.id;
      if (!action || !id) return;

      if (action === "edit") {
        const vrtic = state.vrtici.find((v) => String(v.id) === String(id));
        if (!vrtic) return;
        setEditMode(vrtic);
        if (el.vrticFormStatus) el.vrticFormStatus.textContent = "Rezim izmene aktivan.";
        window.scrollTo({ top: 0, behavior: "smooth" });
        return;
      }

      if (action === "delete") {
        const ok = window.confirm("Da li sigurno zelis da obrises vrtic?");
        if (!ok) return;

        try {
          await deleteVrticById(id);
          if (el.vrticFormStatus) el.vrticFormStatus.textContent = "Vrtic je obrisan.";
          if (state.editingVrticId === id) {
            resetEditMode();
            if (el.vrticForm) el.vrticForm.reset();
          }
          await fetchVrtici();
        } catch (err) {
          if (el.vrticFormStatus) {
            el.vrticFormStatus.textContent = `Greska pri brisanju: ${err.message || "Neuspesno"}`;
          }
        }
      }
    });
  }
}

function bindLoginEvents() {
  if (!el.loginForm || !el.loginStatus || !el.userInfo) return;

  const payload = decodeTokenPayload();
  if (payload?.sub) {
    el.userInfo.textContent = `Trenutno ulogovan: ${payload.sub}${payload.role ? ` (${payload.role})` : ""}`;
  }

  el.loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const formData = new FormData(el.loginForm);
    const email = String(formData.get("email") || "").trim().toLowerCase();
    const password = String(formData.get("password") || "");

    el.loginStatus.textContent = "Prijava...";

    try {
      const res = await fetch(`${API_AUTH}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) throw new Error("Neispravni kredencijali");

      const data = await res.json();
      tokenStore.set(data.access_token);

      el.loginStatus.textContent = "Ulogovan";
      el.userInfo.textContent = `Trenutno ulogovan: ${data.email} (${data.role})`;
      setTimeout(() => {
        window.location.href = "profil.html";
      }, 500);
    } catch (err) {
      el.loginStatus.textContent = `Neuspesna prijava: ${err.message}`;
    }
  });
}

function bindRegisterEvents() {
  if (!el.registerForm || !el.registerStatus) return;

  el.registerForm.addEventListener("submit", async (e) => {
    e.preventDefault();

    const formData = new FormData(el.registerForm);
    const payload = {
      email: String(formData.get("email") || "").trim().toLowerCase(),
      password: String(formData.get("password") || ""),
      role: String(formData.get("role") || "korisnik"),
    };

    el.registerStatus.textContent = "Registracija...";

    try {
      const res = await fetch(`${API_AUTH}/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || "Neuspesna registracija");
      }

      el.registerStatus.textContent = "Registracija uspesna. Preusmeravam na login...";
      el.registerForm.reset();
      setTimeout(() => {
        window.location.href = "login.html";
      }, 800);
    } catch (err) {
      el.registerStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
    }
  });
}

async function fetchProfile() {
  if (!el.profileStatus || !el.profileEmail || !el.profileRole || !el.profileCreated) return;

  const headers = authHeaders();
  if (!headers) {
    el.profileStatus.textContent = "Nisi ulogovan. Idi na Login stranicu.";
    el.profileEmail.textContent = "-";
    el.profileRole.textContent = "-";
    el.profileCreated.textContent = "-";
    return;
  }

  el.profileStatus.textContent = "Ucitavam profil...";

  try {
    const res = await fetch(`${API_AUTH}/auth/profile`, {
      method: "GET",
      headers,
    });

    if (!res.ok) throw new Error("Ne mogu da ucitam profil");

    const data = await res.json();
    el.profileEmail.textContent = data.email || "-";
    el.profileRole.textContent = data.role || "-";
    el.profileCreated.textContent = data.created_at
      ? new Date(data.created_at).toLocaleString("sr-RS")
      : "-";
    el.profileStatus.textContent = "Profil je ucitan.";
  } catch (err) {
    el.profileStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
  }
}

function bindProfileEvents() {
  if (el.profileRefresh) {
    el.profileRefresh.addEventListener("click", fetchProfile);
  }

  if (el.profileLogout) {
    el.profileLogout.addEventListener("click", () => {
      tokenStore.clear();
      if (el.profileStatus) el.profileStatus.textContent = "Odjavljen.";
      setTimeout(() => {
        window.location.href = "login.html";
      }, 400);
    });
  }
}

async function downloadOpstinaPdf() {
  if (!el.downloadReport) return;

  const oldLabel = el.downloadReport.textContent;
  if (el.reportStatus) el.reportStatus.textContent = "Preuzimam...";
  el.downloadReport.disabled = true;

  try {
    const res = await fetch(`${API_BASE}/vrtici/izvestaj/opstina?format=pdf`);
    if (!res.ok) throw new Error("PDF nije dostupan");

    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `izvestaj-opstina-${new Date().toISOString().slice(0, 10)}.pdf`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);

    if (el.reportStatus) el.reportStatus.textContent = "PDF preuzet.";
  } catch (err) {
    if (el.reportStatus) el.reportStatus.textContent = `Greska: ${err.message || "Neuspesno"}`;
  } finally {
    el.downloadReport.disabled = false;
    el.downloadReport.textContent = oldLabel;
  }
}

function bindReportActions() {
  if (!el.downloadReport) return;
  el.downloadReport.addEventListener("click", downloadOpstinaPdf);
}

async function bootstrap() {
  if (el.apiBase) el.apiBase.textContent = API_BASE;
  setNavActive();

  bindListEvents();
  bindCrudEvents();
  bindLoginEvents();
  bindRegisterEvents();
  bindProfileEvents();
  bindReportActions();

  const needsVrtici = Boolean(
    el.cards ||
      el.manageCards ||
      el.statTotal ||
      el.filterTip ||
      el.filterGrad ||
      el.filterOpstina ||
      el.search ||
      el.sortMode
  );

  if (needsVrtici) {
    await fetchVrtici();
  }

  if (el.opstinaReport) {
    await fetchOpstinaReportJson();
  }

  if (el.criticalCards) {
    await fetchKriticni();
  }

  if (page === "profil") {
    await fetchProfile();
  }
}

bootstrap();
