// ============================================================================
// QueueDrop frontend — talks to the Go/Fiber API (register, login, events, queue)
// No build step: open via any static server. Configure the API origin below
// or via localStorage key "queuedrop_api_base".
// ============================================================================

const DEFAULT_API_BASE = "https://queuedrop-production-d25f.up.railway.app";
const API_BASE = localStorage.getItem("queuedrop_api_base") || DEFAULT_API_BASE;

const state = {
  token: localStorage.getItem("queuedrop_token") || null,
  user: null, // { username, role }
  events: [],
  search: "",
};

// ---------------------------------------------------------------------------
// DOM refs
// ---------------------------------------------------------------------------
const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => Array.from(document.querySelectorAll(sel));

const el = {
  bootGate: $("#boot-gate"),
  siteHead: $("#site-head"),
  serverPill: $("#server-pill"),
  serverDot: $("#server-dot"),
  serverLabel: $("#server-label"),

  viewAuth: $("#view-auth"),
  viewDrops: $("#view-drops"),

  authTabs: $$(".auth-tab"),
  formLogin: $("#form-login"),
  formRegister: $("#form-register"),
  loginError: $("#login-error"),
  registerError: $("#register-error"),
  noteLogin: $("#note-login"),
  noteRegister: $("#note-register"),

  headUser: $("#head-user"),
  userChip: $("#user-chip"),
  userRoleBadge: $("#user-role-badge"),
  userNameLabel: $("#user-name-label"),
  btnLogout: $("#btn-logout"),
  btnNewDrop: $("#btn-new-drop"),
  btnNewDrop2: $("#btn-new-drop-2"),
  btnEmptyCreate: $("#btn-empty-create"),

  searchInput: $("#search-events"),
  eventsGrid: $("#events-grid"),
  eventsEmpty: $("#events-empty"),
  emptyCopy: $("#empty-copy"),

  modalDrop: $("#modal-drop"),
  formDrop: $("#form-drop"),
  dropModalTitle: $("#drop-modal-title"),
  dropModalHint: $("#drop-form-hint"),
  dropModalError: $("#drop-form-error"),
  dropModalSubmit: $("#drop-modal-submit"),
  dropModalClose: $("#drop-modal-close"),
  dropModalCancel: $("#drop-modal-cancel"),

  modalJoined: $("#modal-joined"),
  joinedIcon: $("#joined-icon"),
  joinedTitle: $("#joined-title"),
  joinedCopy: $("#joined-copy"),
  joinedEventName: $("#joined-event-name"),
  joinedUsername: $("#joined-username"),
  joinedClose: $("#joined-close"),
  joinedStatus: $("#joined-status"),

  toastRack: $("#toast-rack"),
};

// ---------------------------------------------------------------------------
// API client
// ---------------------------------------------------------------------------
async function api(path, { method = "GET", body, auth = false } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (auth && state.token) headers["Authorization"] = `Bearer ${state.token}`;

  let res;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (networkErr) {
    setServerStatus(false);
    throw new Error(
      `Can't reach the QueueDrop API at ${API_BASE}. Is the backend running and is CORS enabled?`
    );
  }

  setServerStatus(true);

  let data = null;
  const text = await res.text();
  if (text) {
    try { data = JSON.parse(text); } catch { /* non-JSON response */ }
  }

  if (!res.ok) {
    const message = (data && data.error) || `Request failed (${res.status})`;
    throw new Error(message);
  }
  return data;
}

function setServerStatus(online) {
  el.serverPill.classList.toggle("is-online", online);
  el.serverPill.classList.toggle("is-offline", !online);
  el.serverLabel.textContent = online ? "box office online" : "box office unreachable";
}

// ---------------------------------------------------------------------------
// Auth / JWT
// ---------------------------------------------------------------------------
function decodeJwt(token) {
  try {
    const payload = token.split(".")[1];
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(decodeURIComponent(escape(json)));
  } catch {
    return null;
  }
}

function setSession(token) {
  state.token = token;
  localStorage.setItem("queuedrop_token", token);
  const claims = decodeJwt(token);
  state.user = claims ? { username: claims.username, role: claims.role } : null;
}

function clearSession() {
  state.token = null;
  state.user = null;
  localStorage.removeItem("queuedrop_token");
}

function isOrganizer() {
  return state.user && (state.user.role === "organizer" || state.user.role === "admin");
}

// ---------------------------------------------------------------------------
// Toasts
// ---------------------------------------------------------------------------
function toast(message, type = "default") {
  const node = document.createElement("div");
  node.className = `toast${type === "error" ? " toast-error" : ""}${type === "success" ? " toast-success" : ""}`;
  node.textContent = message;
  el.toastRack.appendChild(node);
  setTimeout(() => {
    node.classList.add("is-leaving");
    setTimeout(() => node.remove(), 220);
  }, 4200);
}

// ---------------------------------------------------------------------------
// View switching
// ---------------------------------------------------------------------------
function renderAuthState() {
  const loggedIn = !!state.token && !!state.user;

  el.siteHead.hidden = false;
  el.viewAuth.hidden = loggedIn;
  el.viewDrops.hidden = !loggedIn;

  el.userChip.hidden = !loggedIn;
  el.btnLogout.hidden = !loggedIn;
  el.btnNewDrop.hidden = !(loggedIn && isOrganizer());
  el.btnNewDrop2.hidden = !(loggedIn && isOrganizer());

  if (loggedIn) {
    el.userRoleBadge.textContent = state.user.role;
    el.userNameLabel.textContent = state.user.username;
    loadEvents();
  }
}

function switchAuthTab(tab) {
  el.authTabs.forEach((btn) => btn.classList.toggle("is-active", btn.dataset.tab === tab));
  el.formLogin.hidden = tab !== "login";
  el.formRegister.hidden = tab !== "register";
  el.noteLogin.hidden = tab !== "login";
  el.noteRegister.hidden = tab !== "register";
  el.loginError.hidden = true;
  el.registerError.hidden = true;
}

$$("[data-tab]").forEach((node) => {
  node.addEventListener("click", () => switchAuthTab(node.dataset.tab));
});

// ---------------------------------------------------------------------------
// Login / Register
// ---------------------------------------------------------------------------
function setButtonLoading(btn, loading) {
  btn.disabled = loading;
  btn.classList.toggle("is-loading", loading);
}

el.formLogin.addEventListener("submit", async (e) => {
  e.preventDefault();
  el.loginError.hidden = true;
  const fd = new FormData(el.formLogin);
  const submitBtn = el.formLogin.querySelector("button[type=submit]");
  setButtonLoading(submitBtn, true);
  try {
    const data = await api("/api/login", {
      method: "POST",
      body: { username: fd.get("username").trim(), password: fd.get("password") },
    });
    setSession(data.token);
    toast(`Welcome back, ${state.user.username}.`, "success");
    el.formLogin.reset();
    renderAuthState();
  } catch (err) {
    el.loginError.textContent = err.message;
    el.loginError.hidden = false;
  } finally {
    setButtonLoading(submitBtn, false);
  }
});

el.formRegister.addEventListener("submit", async (e) => {
  e.preventDefault();
  el.registerError.hidden = true;
  const fd = new FormData(el.formRegister);
  const submitBtn = el.formRegister.querySelector("button[type=submit]");
  setButtonLoading(submitBtn, true);
  try {
    await api("/api/register", {
      method: "POST",
      body: {
        username: fd.get("username").trim(),
        password: fd.get("password"),
        role: fd.get("role"),
      },
    });
    toast("Account created. Sign in to continue.", "success");
    el.formRegister.reset();
    switchAuthTab("login");
    el.formLogin.username.value = fd.get("username").trim();
    el.formLogin.password.focus();
  } catch (err) {
    el.registerError.textContent = err.message;
    el.registerError.hidden = false;
  } finally {
    setButtonLoading(submitBtn, false);
  }
});

$$(".role-option input").forEach((input) => {
  input.addEventListener("change", () => {
    $$(".role-option").forEach((opt) => opt.classList.toggle("is-active", opt.contains(input) && input.checked));
  });
});

el.btnLogout.addEventListener("click", () => {
  clearSession();
  switchAuthTab("login");
  toast("Signed out.");
  renderAuthState();
});

// ---------------------------------------------------------------------------
// Events (drops) list
// ---------------------------------------------------------------------------
async function loadEvents() {
  try {
    const events = await api("/api/events");
    state.events = Array.isArray(events) ? events : [];
    renderEvents();
  } catch (err) {
    toast(err.message, "error");
    state.events = [];
    renderEvents();
  }
}

function renderEvents() {
  const q = state.search.trim().toLowerCase();
  const filtered = state.events.filter((ev) => {
    if (!q) return true;
    return (ev.name || "").toLowerCase().includes(q) || (ev.organizer_username || "").toLowerCase().includes(q);
  });

  el.eventsGrid.innerHTML = "";

  if (filtered.length === 0) {
    el.eventsGrid.hidden = true;
    el.eventsEmpty.hidden = false;
    if (state.events.length === 0) {
      el.emptyCopy.textContent = isOrganizer()
        ? "Publish your first drop and it'll show up here for everyone."
        : "Check back soon — organizers are still setting up.";
      el.btnEmptyCreate.hidden = !isOrganizer();
    } else {
      el.emptyCopy.textContent = `Nothing matches "${state.search}".`;
      el.btnEmptyCreate.hidden = true;
    }
    return;
  }

  el.eventsGrid.hidden = false;
  el.eventsEmpty.hidden = true;

  filtered.forEach((ev) => el.eventsGrid.appendChild(renderTicketCard(ev)));
}

function renderTicketCard(ev) {
  const canEdit = state.user && (state.user.role === "admin" || ev.organizer_username === state.user.username);

  const card = document.createElement("article");
  card.className = "ticket";
  card.innerHTML = `
    <div class="ticket-main">
      <p class="ticket-organizer">Hosted by <strong>${escapeHtml(ev.organizer_username || "unknown")}</strong></p>
      <h3 class="ticket-name">${escapeHtml(ev.name)}</h3>
      ${canEdit ? `<span class="ticket-owner-badge">Your drop</span>` : ""}
      ${canEdit ? `<button class="ticket-edit" type="button" data-edit="${ev.id}">Edit drop details</button>` : ""}
    </div>
    <div class="ticket-stub">
      <span class="stub-label">Capacity</span>
      <span class="stub-count">${Number(ev.total_tickets).toLocaleString()}</span>
      <button class="btn btn-primary stub-join" type="button" data-join="${ev.id}" data-name="${escapeHtml(ev.name)}">
        <span class="btn-label">Join queue</span>
      </button>
    </div>
  `;

  const joinBtn = card.querySelector("[data-join]");
  joinBtn.addEventListener("click", () => joinQueue(ev.id, ev.name, joinBtn));

  const editBtn = card.querySelector("[data-edit]");
  if (editBtn) editBtn.addEventListener("click", () => openDropModal("edit", ev));

  return card;
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str ?? "";
  return div.innerHTML;
}

el.searchInput.addEventListener("input", (e) => {
  state.search = e.target.value;
  renderEvents();
});

// ---------------------------------------------------------------------------
// Join queue
// ---------------------------------------------------------------------------
async function joinQueue(eventId, eventName, btn) {
  setButtonLoading(btn, true);
  try {
    const data = await api(`/api/events/${eventId}/queue`, { method: "POST", auth: true });
    el.joinedEventName.textContent = eventName;
    el.joinedUsername.textContent = data.username || state.user.username;
    el.joinedCopy.textContent =
      "Your spot is locked in the order it arrived. QueueDrop's worker allocates tickets first-come, first-served, so there's nothing more to do here.";
    el.joinedStatus.textContent = "Checking your spot…";
    el.joinedStatus.className = "joined-status joined-status-pending";
    openModal(el.modalJoined);
    pollOrderStatus(eventId);
  } catch (err) {
    toast(err.message, "error");
  } finally {
    setButtonLoading(btn, false);
  }
}

// Polls the worker's result for this event a few times, a second apart.
// The purchase is processed asynchronously (via RabbitMQ), so it's
// rarely instant — this gives it a few seconds to land before giving up.
async function pollOrderStatus(eventId, attempt = 0) {
  const maxAttempts = 8;
  try {
    const data = await api(`/api/events/${eventId}/status`, { auth: true });
    if (data.status === "confirmed") {
      el.joinedStatus.textContent = "🎉 Confirmed — you got a ticket!";
      el.joinedStatus.className = "joined-status joined-status-confirmed";
      return;
    }
    if (data.status === "sold_out") {
      el.joinedStatus.textContent = "Sold out — no tickets left for you this time.";
      el.joinedStatus.className = "joined-status joined-status-soldout";
      return;
    }
    // still "pending" — try again shortly, unless we've hit the cap.
    if (attempt < maxAttempts) {
      setTimeout(() => pollOrderStatus(eventId, attempt + 1), 1000);
    } else {
      el.joinedStatus.textContent = "Still processing — check My Orders in a moment.";
      el.joinedStatus.className = "joined-status joined-status-pending";
    }
  } catch (err) {
    el.joinedStatus.textContent = "Couldn't check status — check My Orders instead.";
    el.joinedStatus.className = "joined-status joined-status-pending";
  }
}

el.joinedClose.addEventListener("click", () => closeModal(el.modalJoined));

// ---------------------------------------------------------------------------
// Create / edit drop modal
// ---------------------------------------------------------------------------
let editingEventId = null;

function openDropModal(mode, ev = null) {
  el.dropModalError.hidden = true;
  el.formDrop.reset();
  if (mode === "edit" && ev) {
    editingEventId = ev.id;
    el.dropModalTitle.textContent = "Edit drop";
    el.formDrop.name.value = ev.name;
    el.formDrop.total_tickets.value = ev.total_tickets;
    el.dropModalHint.textContent = "Lowering capacity below tickets already sold isn't allowed.";
    el.dropModalSubmit.querySelector(".btn-label").textContent = "Save changes";
  } else {
    editingEventId = null;
    el.dropModalTitle.textContent = "New drop";
    el.dropModalHint.textContent = "This goes live on the board the moment you publish it.";
    el.dropModalSubmit.querySelector(".btn-label").textContent = "Publish drop";
  }
  openModal(el.modalDrop);
  el.formDrop.name.focus();
}

el.formDrop.addEventListener("submit", async (e) => {
  e.preventDefault();
  el.dropModalError.hidden = true;
  const fd = new FormData(el.formDrop);
  const payload = {
    name: fd.get("name").trim(),
    total_tickets: Number(fd.get("total_tickets")),
  };
  setButtonLoading(el.dropModalSubmit, true);
  try {
    if (editingEventId) {
      await api(`/api/events/${editingEventId}`, { method: "PUT", auth: true, body: payload });
      toast("Drop updated.", "success");
    } else {
      await api("/api/events", { method: "POST", auth: true, body: payload });
      toast("Drop published.", "success");
    }
    closeModal(el.modalDrop);
    loadEvents();
  } catch (err) {
    el.dropModalError.textContent = err.message;
    el.dropModalError.hidden = false;
  } finally {
    setButtonLoading(el.dropModalSubmit, false);
  }
});

[el.btnNewDrop, el.btnNewDrop2, el.btnEmptyCreate].forEach((btn) =>
  btn.addEventListener("click", () => openDropModal("create"))
);
el.dropModalClose.addEventListener("click", () => closeModal(el.modalDrop));
el.dropModalCancel.addEventListener("click", () => closeModal(el.modalDrop));

// ---------------------------------------------------------------------------
// Modal helpers
// ---------------------------------------------------------------------------
function openModal(node) {
  node.hidden = false;
}
function closeModal(node) {
  node.hidden = true;
}
$$(".modal-overlay").forEach((overlay) => {
  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) closeModal(overlay);
  });
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") $$(".modal-overlay").forEach((m) => (m.hidden = true));
});

// ---------------------------------------------------------------------------
// Boot
// ---------------------------------------------------------------------------
$("[data-action='go-home']").addEventListener("click", (e) => {
  e.preventDefault();
  state.search = "";
  el.searchInput.value = "";
  renderEvents();
});

async function boot() {
  if (state.token) {
    const claims = decodeJwt(state.token);
    if (claims && claims.exp && claims.exp * 1000 > Date.now()) {
      state.user = { username: claims.username, role: claims.role };
    } else {
      clearSession();
    }
  }

  // Ping the API once so the connection pill reflects reality immediately.
  try {
    await api("/api/events");
  } catch {
    /* status already set by api() */
  }

  renderAuthState();
  el.bootGate.classList.add("is-hidden");
}

boot();