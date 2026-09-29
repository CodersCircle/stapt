import * as Backend from "./wailsjs/go/main/App.js";
import { EventsOn, EventsOff } from "./wailsjs/runtime/runtime.js";

const VIEWS = ["servers", "terminal", "files", "commands", "notes", "settings"];

const state = {
  view: "servers",
  servers: [],
  projects: [],
  serverId: Number(localStorage.getItem("stapt.serverId") || 0),
  projectId: Number(localStorage.getItem("stapt.projectId") || 0),
  meta: {},
  cwd: "",
  files: [],
  edit: null,
  hostKey: "",
  dirty: false,
  method: "password",
  i18n: {},
  notes: [],
  noteOpen: null,
  commands: [],
  cmdOut: {},
  discovering: false,
  showConnect: false,
  connected: false,
  savedPaths: [],
  uploadHistory: [],
  theme: localStorage.getItem("stapt.theme") || "dark",
  pickPath: false,
  pickCwd: "",
};

let term, fit, cm;
let termSessionId = null;
let termBoundProject = 0;

const $ = (id) => document.getElementById(id);
const t = (key) => state.i18n[key] || key;

const JUNK_DOMAIN_NAMES = new Set(["www", "public_html", "html", "httpdocs", "htdocs"]);

function isDomainCard(p) {
  const n = String((p && p.name) || "").toLowerCase().trim().replace(/\.+$/, "");
  if (!n || JUNK_DOMAIN_NAMES.has(n) || (p && p.kind) === "document_root") return false;
  if (n.startsWith("www.")) return false;
  const first = n.split(".")[0];
  if (JUNK_DOMAIN_NAMES.has(first) && n.includes(".")) return false;
  return true;
}

function applyTheme() {
  document.documentElement.classList.toggle("light", state.theme === "light");
  localStorage.setItem("stapt.theme", state.theme);
}

function activePath() {
  const p = currentProject();
  return state.cwd || (p && (p.workPath || p.remotePath)) || "";
}

function fmtWhen(s) {
  if (!s) return "—";
  return String(s).replace("T", " ").replace("Z", "").slice(0, 19);
}

function loadHistory() {
  try {
    state.uploadHistory = JSON.parse(localStorage.getItem("stapt.uploadHistory") || "[]") || [];
  } catch {
    state.uploadHistory = [];
  }
}

function persistHistory() {
  localStorage.setItem("stapt.uploadHistory", JSON.stringify((state.uploadHistory || []).slice(0, 40)));
}

function errMsg(err) {
  if (!err) return "unknown error";
  if (typeof err === "string") return err;
  return err.message || String(err);
}

async function call(fn) {
  try {
    return await fn();
  } catch (err) {
    throw new Error(errMsg(err));
  }
}

function toast(msg, ok) {
  const el = $("toast");
  if (!el) return;
  el.textContent = msg;
  el.classList.remove("hidden", "border-red-800", "border-yellow-800");
  el.classList.add(ok ? "border-yellow-800" : "border-red-800");
  clearTimeout(toast._t);
  toast._t = setTimeout(() => el.classList.add("hidden"), 3500);
}

function modal({ text, input, value = "", ok = t("ok") }) {
  return new Promise((resolve) => {
    const box = $("modal");
    $("modal-text").textContent = text;
    const inp = $("modal-input");
    if (input) {
      inp.classList.remove("hidden");
      inp.value = value;
    } else {
      inp.classList.add("hidden");
    }
    $("modal-ok").textContent = ok;
    box.classList.remove("hidden");
    const done = (v) => {
      box.classList.add("hidden");
      $("modal-ok").onclick = null;
      $("modal-cancel").onclick = null;
      resolve(v);
    };
    $("modal-cancel").onclick = () => done(null);
    $("modal-ok").onclick = () => done(input ? inp.value : true);
    if (input) {
      inp.focus();
      inp.onkeydown = (e) => {
        if (e.key === "Enter") done(inp.value);
      };
    }
  });
}

function esc(s) {
  return String(s == null ? "" : s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function currentServer() {
  return state.servers.find((s) => s.id === state.serverId) || null;
}
function currentProject() {
  return state.projects.find((p) => p.id === state.projectId) || null;
}

function persist() {
  localStorage.setItem("stapt.serverId", String(state.serverId || 0));
  localStorage.setItem("stapt.projectId", String(state.projectId || 0));
}

function fillSelects() {
  const ss = $("sel-server");
  const ps = $("sel-project");
  if (!ss || !ps) return;
  ss.innerHTML =
    `<option value="0">${esc(t("label.select"))}</option>` +
    state.servers
      .map(
        (s) =>
          `<option value="${s.id}" ${s.id === state.serverId ? "selected" : ""}>${esc(s.name)} (${esc(s.host)})</option>`
      )
      .join("");
  ps.innerHTML =
    `<option value="0">${esc(t("label.select"))}</option>` +
    state.projects
      .map(
        (p) =>
          `<option value="${p.id}" ${p.id === state.projectId ? "selected" : ""}>${esc(p.name)}</option>`
      )
      .join("");
}

async function loadServers() {
  try {
    state.servers = (await call(() => Backend.ListServers())) || [];
  } catch (err) {
    state.servers = state.servers || [];
    toast(errMsg(err));
  }
  if (state.serverId && !state.servers.some((s) => s.id === state.serverId)) state.serverId = 0;
  fillSelects();
}

async function loadProjects() {
  if (!state.serverId) {
    state.projects = [];
    state.savedPaths = [];
    state.projectId = 0;
    fillSelects();
    return;
  }
  try {
    state.projects = ((await call(() => Backend.ListProjects(state.serverId))) || []).filter(isDomainCard);
  } catch (err) {
    state.projects = [];
    toast(errMsg(err));
  }
  try {
    state.savedPaths = (await call(() => Backend.ListSavedPaths(state.serverId, 0))) || [];
  } catch {
    state.savedPaths = [];
  }
  if (state.projectId && !state.projects.some((p) => p.id === state.projectId)) state.projectId = 0;
  fillSelects();
  persist();
}

function showView(name) {
  state.view = name;
  document.querySelectorAll(".nav-btn").forEach((b) => b.classList.toggle("active", b.dataset.view === name));
  VIEWS.forEach((v) => {
    const el = $("view-" + v);
    if (!el) return;
    const on = v === name;
    el.classList.toggle("hidden", !on);
    if (v === "terminal" || v === "files") el.classList.toggle("flex", on);
  });
  if (name === "servers") renderServers();
  if (name === "terminal") {
    requestAnimationFrame(() => {
      if (fit) fit.fit();
      sendResize();
    });
    if (currentProject() && (!termSessionId || termBoundProject !== state.projectId)) connectTerm();
  }
  if (name === "files") loadFiles(state.cwd || (currentProject() && currentProject().remotePath) || "");
  if (name === "commands") renderCommands();
  if (name === "notes") renderNotes();
  if (name === "settings") renderSettings();
}

function field(label, inner) {
  return `<label class="block text-xs text-zinc-400 mb-3">${esc(label)}${inner}</label>`;
}
function inp(id, type, value, extra = "") {
  return `<input id="${id}" type="${type}" value="${esc(value || "")}" ${extra} class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm" />`;
}

function statusLabel(s) {
  if (s === "needs_root") return t("card.status.needs_root");
  if (s === "offline") return t("card.status.offline");
  return t("card.status.ready");
}

function serverFormHTML() {
  const method = state.method;
  return `
        <form id="server-form">
          <h2 class="text-sm font-medium mb-3" id="server-form-title">${esc(t("ssh.title"))}</h2>
          <input type="hidden" id="sv-id" value="" />
          <div class="flex gap-1 mb-3 text-xs">
            <button type="button" data-method="command" class="method-btn px-2 py-1 rounded bg-zinc-800 ${method === "command" ? "active" : ""}">${esc(t("ssh.method.command"))}</button>
            <button type="button" data-method="password" class="method-btn px-2 py-1 rounded bg-zinc-800 ${method === "password" ? "active" : ""}">${esc(t("ssh.method.password"))}</button>
            <button type="button" data-method="key" class="method-btn px-2 py-1 rounded bg-zinc-800 ${method === "key" ? "active" : ""}">${esc(t("ssh.method.key"))}</button>
          </div>
          <div id="sv-cmd-wrap" class="${method === "command" ? "" : "hidden"}">
            ${field(t("ssh.command"), `<input id="sv-cmd" type="text" class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm font-mono" placeholder="${esc(t("ssh.command.hint"))}" autocomplete="off" />`)}
            <button type="button" id="sv-parse" class="mb-3 px-3 py-1.5 text-sm rounded bg-zinc-800">${esc(t("ssh.parse"))}</button>
          </div>
          ${field(t("ssh.name"), inp("sv-name", "text", ""))}
          ${field(t("ssh.host"), inp("sv-host", "text", "", 'autocomplete="off"'))}
          ${field(t("ssh.port"), inp("sv-port", "number", "22", 'min="1" max="65535"'))}
          ${field(t("ssh.username"), inp("sv-user", "text", "", 'autocomplete="off"'))}
          <div id="sv-pass-wrap" class="${method === "key" ? "hidden" : ""}">${field(t("ssh.password"), inp("sv-pass", "password", "", 'autocomplete="new-password"'))}</div>
          <div id="sv-key-wrap" class="${method === "key" ? "" : "hidden"}">
            ${field(t("ssh.key"), `<textarea id="sv-key" rows="5" class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm font-mono"></textarea>`)}
            ${field(t("ssh.passphrase"), inp("sv-phrase", "password", "", 'autocomplete="new-password"'))}
          </div>
          <p id="sv-keep" class="hidden text-[11px] text-zinc-500 mb-3">${esc(t("ssh.keep"))}</p>
          <label id="sv-forget-wrap" class="hidden text-xs text-zinc-400 mb-3">
            <input id="sv-forget" type="checkbox" class="mr-1 align-middle" /> ${esc(t("ssh.forget_host"))}
          </label>
          <div class="flex gap-2 flex-wrap">
            <button type="button" id="sv-connect" class="px-3 py-1.5 text-sm rounded bg-yellow-400 text-zinc-950 font-medium">${esc(t("ssh.connect"))}</button>
            <button type="button" id="sv-test" class="px-3 py-1.5 text-sm rounded bg-zinc-800">${esc(t("ssh.test"))}</button>
            <button type="submit" class="px-3 py-1.5 text-sm rounded bg-zinc-800">${esc(t("ssh.save"))}</button>
            <button type="button" id="sv-new" class="px-3 py-1.5 text-sm rounded bg-zinc-800">${esc(t("ssh.clear"))}</button>
            <button type="button" id="sv-del" class="hidden ml-auto px-3 py-1.5 text-sm rounded bg-red-900">${esc(t("ssh.delete"))}</button>
          </div>
        </form>`;
}

function bindServerForm() {
  document.querySelectorAll("[data-method]").forEach((b) => {
    b.onclick = () => {
      state.method = b.dataset.method;
      const id = Number(($("sv-id") && $("sv-id").value) || 0);
      renderServers();
      if (id) fillServerForm(id);
    };
  });
  if ($("sv-new")) $("sv-new").onclick = () => fillServerForm(0);
  if ($("sv-test")) $("sv-test").onclick = () => testServer();
  if ($("sv-connect")) $("sv-connect").onclick = () => connectServer();
  if ($("sv-del")) $("sv-del").onclick = () => deleteServer();
  if ($("sv-parse")) $("sv-parse").onclick = () => parseCommand();
  const form = $("server-form");
  if (form) {
    form.onsubmit = (e) => {
      e.preventDefault();
      saveServer();
    };
  }
}

function renderShimmerCards() {
  const skel = Array.from({ length: 6 }, () => `
    <div class="border border-zinc-800 rounded-lg p-3 h-40 flex flex-col gap-2">
      <div class="h-4 w-3/5 rounded skel"></div>
      <div class="h-3 w-2/5 rounded skel"></div>
      <div class="h-3 w-full rounded skel"></div>
      <div class="h-3 w-4/5 rounded skel"></div>
      <div class="mt-auto h-8 w-full rounded skel"></div>
    </div>`).join("");
  return `<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">${skel}</div>`;
}

function renderDomainCards() {
  const cards =
    state.projects
      .filter(isDomainCard)
      .map((p) => {
        const active = p.id === state.projectId;
        const root = p.workPath || p.remotePath || "";
        return `
    <div class="border rounded-lg p-3 flex flex-col ${active ? "border-yellow-400/70 bg-yellow-950/20" : "border-zinc-800"}">
      <div class="text-sm font-medium truncate">${esc(p.name)}</div>
      <div class="text-[11px] text-zinc-500 mt-1">${esc(t("domains.type"))}: ${esc(p.kind || "project")}</div>
      <div class="text-[11px] text-zinc-500 font-mono break-all">${esc(t("domains.root"))}: ${esc(p.remotePath || "")}</div>
      <div class="text-[11px] text-zinc-500 font-mono break-all">${esc(t("path.current"))}: ${esc(root)}</div>
      <div class="text-[11px] mt-1 text-yellow-400">${esc(t("domains.status"))}: ${esc(state.connected ? statusLabel(p.status) : t("card.status.offline"))}</div>
      <div class="text-[11px] text-zinc-600">${esc(t("path.last_upload"))}: ${esc(p.lastUploadFile || "—")} ${p.lastUploadSize ? "· " + fmtSize(p.lastUploadSize) : ""} ${p.lastUploadAt ? "· " + fmtWhen(p.lastUploadAt) : ""}</div>
      <div class="text-[11px] text-zinc-600">${esc(t("path.test_status"))}: ${esc(p.pathTestStatus || "—")} ${p.pathTestAt ? "· " + fmtWhen(p.pathTestAt) : ""}</div>
      <div class="text-[11px] text-zinc-600">${esc(t("domains.last"))}: ${esc(fmtWhen(p.lastChecked))}</div>
      <button data-open-pj="${p.id}" class="mt-auto pt-3 w-full px-2 py-1.5 rounded bg-yellow-400 text-zinc-950 text-xs font-medium">${esc(t("domains.open"))}</button>
    </div>`;
      })
      .join("");

  if (!cards) {
    return `<p class="text-sm text-zinc-500 text-center">${esc(t("domains.empty"))}</p>`;
  }
  return `<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">${cards}</div>`;
}

function renderSavedServerCards() {
  if (!state.servers.length) return "";
  const cards = state.servers
    .map((s) => {
      const on = s.id === state.serverId;
      const live = on && state.connected;
      return `
    <div class="border rounded-lg p-3 ${on ? "border-yellow-400/70 bg-yellow-950/20" : "border-zinc-800"}">
      <div class="text-sm font-medium truncate">${esc(s.name)}</div>
      <div class="text-[11px] text-zinc-500">${esc(s.host)}:${esc(String(s.port || 22))}</div>
      <div class="text-[11px] text-zinc-500">${esc(s.username)} · ${esc(s.authType || "")}</div>
      <div class="text-[11px] text-zinc-600">${esc(t("ssh.saved_projects"))}: ${esc(String(s.projectCount || 0))} · ${esc(t("ssh.saved_paths"))}: ${esc(String(s.pathCount || 0))}</div>
      <div class="text-[11px] text-zinc-600">${esc(t("ssh.last_connected"))}: ${esc(fmtWhen(s.lastConnected))}</div>
      <div class="text-[11px] mt-1 ${live ? "text-yellow-400" : "text-zinc-500"}">${live ? esc(t("ssh.connected")) : esc(t("card.status.offline"))}</div>
      <div class="flex gap-2 mt-3">
        <button data-open-form="${s.id}" class="flex-1 px-2 py-1.5 rounded bg-zinc-800 text-xs">${esc(t("ssh.edit"))}</button>
        ${live
          ? `<button data-disc="${s.id}" class="flex-1 px-2 py-1.5 rounded bg-zinc-800 text-xs">${esc(t("ssh.disconnect"))}</button>`
          : `<button data-re="${s.id}" class="flex-1 px-2 py-1.5 rounded bg-yellow-400 text-zinc-950 text-xs font-medium">${esc(t("ssh.reconnect"))}</button>`}
      </div>
    </div>`;
    })
    .join("");
  return `<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 mb-4">${cards}</div>`;
}

function renderServers() {
  const el = $("view-servers");
  if (!el) return;
  try {
  const chips = state.servers
    .map(
      (s) => `
    <button data-edit="${s.id}" class="text-left px-3 py-1.5 rounded border text-xs ${s.id === state.serverId ? "border-yellow-400/60 bg-yellow-950/20" : "border-zinc-800"}">
      ${esc(s.name)}
    </button>`
    )
    .join("");

  const hasCards = state.projects.length > 0;
  el.innerHTML = `
    <div class="h-full flex flex-col">
      <div class="flex items-center gap-2 mb-4 flex-wrap">
        <h1 class="text-lg font-medium">${esc(t("nav.servers"))}</h1>
        <div class="flex flex-wrap gap-1">${chips}</div>
        <div class="ml-auto flex gap-2">
          ${state.serverId && state.connected ? `<button id="pj-scan" class="text-xs px-2 py-1.5 rounded bg-zinc-800">${esc(t("domains.scan"))}</button>` : ""}
          ${state.connected ? `<button id="btn-disconnect" class="px-3 py-1.5 text-sm rounded bg-zinc-800">${esc(t("ssh.disconnect"))}</button>` : ""}
          <button id="btn-connect" class="px-3 py-1.5 text-sm rounded bg-yellow-400 text-zinc-950 font-medium">${esc(state.connected ? t("ssh.connect_server") : t("ssh.reconnect"))}</button>
        </div>
      </div>
      ${state.discovering ? `<p class="text-xs text-yellow-400 mb-2">${esc(t("ssh.discovering"))}</p>` : ""}
      ${
        state.discovering
          ? `<div class="flex-1 overflow-auto">${renderShimmerCards()}</div>`
          : `<div class="flex-1 overflow-auto space-y-4">
              ${renderSavedServerCards()}
              ${hasCards ? renderDomainCards() : !state.servers.length ? `<div class="flex items-center justify-center py-16">
              <button id="btn-connect-center" class="px-8 py-4 rounded-lg bg-yellow-400 text-zinc-950 text-lg font-semibold shadow">${esc(t("ssh.connect_server"))}</button>
            </div>` : ""}
            </div>`
      }
    </div>
    <div id="connect-popup" class="${state.showConnect ? "" : "hidden"} fixed inset-0 z-40 bg-black/70 flex items-center justify-center p-4">
      <div class="w-full max-w-md max-h-[90vh] overflow-auto bg-zinc-900 border border-zinc-800 rounded-lg p-5">
        <div class="flex items-center mb-3">
          <span class="text-sm font-medium">${esc(t("ssh.connect_server"))}</span>
          <button type="button" id="sv-popup-close" class="ml-auto text-zinc-400 text-sm">${esc(t("ssh.close"))}</button>
        </div>
        ${serverFormHTML()}
      </div>
    </div>`;

  bindServerForm();
  $("view-servers").querySelectorAll("[data-edit]").forEach((btn) => {
    btn.onclick = async () => {
      state.serverId = Number(btn.dataset.edit);
      persist();
      fillSelects();
      await loadProjects();
      renderServers();
    };
  });
  $("view-servers").querySelectorAll("[data-open-pj]").forEach((btn) => {
    btn.onclick = () => openProject(Number(btn.dataset.openPj));
  });
  $("view-servers").querySelectorAll("[data-open-form]").forEach((btn) => {
    btn.onclick = () => {
      state.serverId = Number(btn.dataset.openForm);
      persist();
      state.showConnect = true;
      renderServers();
      fillServerForm(state.serverId);
    };
  });
  $("view-servers").querySelectorAll("[data-re]").forEach((btn) => {
    btn.onclick = () => {
      state.serverId = Number(btn.dataset.re);
      persist();
      state.showConnect = true;
      renderServers();
      fillServerForm(state.serverId);
    };
  });
  $("view-servers").querySelectorAll("[data-disc]").forEach((btn) => {
    btn.onclick = () => disconnectServer(Number(btn.dataset.disc));
  });
  const openPopup = () => {
    state.showConnect = true;
    renderServers();
    if (state.serverId) fillServerForm(state.serverId);
    else fillServerForm(0);
  };
  if ($("btn-connect")) $("btn-connect").onclick = openPopup;
  if ($("btn-connect-center")) $("btn-connect-center").onclick = openPopup;
  if ($("btn-disconnect")) $("btn-disconnect").onclick = () => disconnectServer(state.serverId);
  if ($("sv-popup-close")) {
    $("sv-popup-close").onclick = () => {
      state.showConnect = false;
      renderServers();
    };
  }
  const scan = $("pj-scan");
  if (scan) scan.onclick = () => scanServer();
  } catch (err) {
    el.innerHTML = `<p class="p-6 text-sm text-red-400">${esc(errMsg(err))}</p>`;
  }
}

function fillServerForm(id) {
  state.hostKey = "";
  const s = state.servers.find((x) => x.id === id);
  if (!$("sv-id")) return;
  $("sv-id").value = id || "";
  $("server-form-title").textContent = s ? t("ssh.edit") : t("ssh.title");
  $("sv-name").value = s ? s.name : "";
  $("sv-host").value = s ? s.host : "";
  $("sv-port").value = s ? s.port : 22;
  $("sv-user").value = s ? s.username : "";
  $("sv-pass").value = "";
  if ($("sv-pass")) $("sv-pass").placeholder = s && s.authType !== "key" ? t("ssh.password_again") : "";
  if ($("sv-key")) $("sv-key").value = "";
  if ($("sv-phrase")) $("sv-phrase").value = "";
  $("sv-keep").classList.toggle("hidden", !s);
  $("sv-del").classList.toggle("hidden", !s);
  $("sv-forget-wrap").classList.toggle("hidden", !(s && s.hasHostKey));
  if ($("sv-forget")) $("sv-forget").checked = false;
  if (s) {
    state.method = s.authType === "key" ? "key" : "password";
    state.serverId = id;
    persist();
    fillSelects();
  }
}

function serverPayload(forTest) {
  const id = Number($("sv-id").value || 0);
  const auth = state.method === "key" ? "key" : "password";
  const body = {
    name: $("sv-name").value.trim() || $("sv-host").value.trim(),
    host: $("sv-host").value.trim(),
    port: Number($("sv-port").value || 22),
    username: $("sv-user").value.trim(),
    authType: auth,
  };
  const pass = $("sv-pass") ? $("sv-pass").value : "";
  const key = $("sv-key") ? $("sv-key").value : "";
  const phrase = $("sv-phrase") ? $("sv-phrase").value : "";
  if (pass) body.password = pass;
  if (key) body.privateKey = key;
  if (phrase || key) body.passphrase = phrase;
  if (!forTest) {
    if (state.hostKey) body.hostKey = state.hostKey;
    if ($("sv-forget") && $("sv-forget").checked) body.clearHostKey = true;
  }
  return { id, body };
}

async function parseCommand(silent) {
  const raw = $("sv-cmd") ? $("sv-cmd").value : "";
  try {
    const p = await call(() => Backend.ParseSSHCommand(raw));
    $("sv-host").value = p.host || "";
    $("sv-port").value = p.port || 22;
    $("sv-user").value = p.username || "";
    $("sv-name").value = "";
    if (!silent) toast(t("ssh.parse"), true);
    return true;
  } catch (err) {
    if (!silent) toast(errMsg(err));
    return false;
  }
}

async function runTest(accept) {
  const { id, body } = serverPayload(true);
  const input = {
    serverId: id || 0,
    host: body.host,
    port: body.port,
    username: body.username,
    authType: body.authType,
    acceptHostKey: accept,
  };
  if (body.password) input.password = body.password;
  if (body.privateKey) input.privateKey = body.privateKey;
  if (body.passphrase) input.passphrase = body.passphrase;
  return Backend.TestConnection(input);
}

async function withHostKey(fn) {
  let res = await fn(false);
  const probe = res && res.test ? res.test : res;
  if (probe && probe.needHostKey) {
    const ok = await modal({
      text: t("ssh.hostkey") + ":\n" + (probe.fingerprintUI || "") + "\n\nTrust and continue?",
      ok: t("ssh.trust"),
    });
    if (!ok) return null;
    res = await fn(true);
  }
  return res;
}

async function testServer() {
  try {
    const res = await withHostKey(runTest);
    if (!res) return;
    if (res.error && !res.ok) {
      toast(res.error);
      return;
    }
    if (res.hostKey) state.hostKey = res.hostKey;
    toast(t("ssh.connected") + " · " + (res.fingerprint || "ok"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function ensureSaved() {
  const { id, body } = serverPayload(false);
  if (id) {
    await call(() => Backend.UpdateServer(id, body));
    return id;
  }
  const newId = await call(() => Backend.CreateServer(body));
  state.serverId = newId;
  persist();
  return newId;
}

async function connectServer() {
  try {
    if (state.method === "command") await parseCommand(true);
    const captured = serverPayload(true);
    if ((captured.body.authType || "password") === "password" && !captured.body.password && captured.id) {
      toast(t("ssh.password_again"));
      return;
    }
    const id = await ensureSaved();
    await loadServers();
    state.showConnect = false;
    state.discovering = true;
    renderServers();
    const send = (accept) =>
      Backend.ConnectAndDiscover({
        serverId: id,
        host: captured.body.host,
        port: captured.body.port,
        username: captured.body.username,
        authType: captured.body.authType,
        password: captured.body.password,
        privateKey: captured.body.privateKey,
        passphrase: captured.body.passphrase,
        acceptHostKey: accept,
      });
    let res = await withHostKey(send);
    state.discovering = false;
    if (!res) {
      state.showConnect = true;
      renderServers();
      fillServerForm(id);
      return;
    }
    const test = res.test || res;
    if (test.error && !test.ok) {
      toast(test.error);
      state.showConnect = true;
      renderServers();
      fillServerForm(id);
      return;
    }
    if (test.hostKey) state.hostKey = test.hostKey;
    await loadProjects();
    state.showConnect = false;
    state.connected = true;
    const p = currentProject();
    if (p) state.cwd = p.workPath || p.remotePath || "";
    renderServers();
    toast(t("ssh.connected"), true);
  } catch (err) {
    state.discovering = false;
    toast(errMsg(err));
    state.showConnect = true;
    renderServers();
    if (state.serverId) fillServerForm(state.serverId);
  }
}

async function scanServer() {
  if (!state.serverId) return;
  try {
    state.discovering = true;
    renderServers();
    await call(() => Backend.DiscoverProjects(state.serverId));
    await loadProjects();
    state.connected = true;
    toast(t("domains.scan"), true);
  } catch (err) {
    toast(errMsg(err));
  } finally {
    state.discovering = false;
    renderServers();
  }
}

async function disconnectServer(id) {
  const sid = id || state.serverId;
  if (!sid) return;
  try {
    disconnectTerm();
    await call(() => Backend.DisconnectServer(sid));
    state.connected = false;
    await loadServers();
    renderServers();
    toast(t("ssh.disconnect"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function saveServer() {
  try {
    const { id, body } = serverPayload(false);
    if (id) await call(() => Backend.UpdateServer(id, body));
    else {
      const newId = await call(() => Backend.CreateServer(body));
      state.serverId = newId;
    }
    toast(t("saved"), true);
    await loadServers();
    await loadProjects();
    renderServers();
    if (state.serverId) fillServerForm(state.serverId);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function deleteServer() {
  const id = Number($("sv-id").value || 0);
  if (!id) return;
  const ok = await modal({ text: t("ssh.delete") + "?", ok: t("ssh.delete") });
  if (!ok) return;
  try {
    await call(() => Backend.DeleteServer(id, true));
    if (state.serverId === id) {
      state.serverId = 0;
      state.connected = false;
    }
    state.projectId = 0;
    persist();
    await loadServers();
    await loadProjects();
    renderServers();
    toast(t("deleted"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

function fillProjectForm(id) {
  const p = state.projects.find((x) => x.id === id);
  if (!$("pj-id")) return;
  $("pj-id").value = id || "";
  $("pj-name").value = p ? p.name : "";
  $("pj-path").value = p ? p.remotePath : "";
  $("pj-del").classList.toggle("hidden", !p);
}

async function saveProject() {
  try {
    const id = Number($("pj-id").value || 0);
    const body = {
      serverId: state.serverId,
      name: $("pj-name").value.trim(),
      remotePath: $("pj-path").value.trim(),
      kind: "project",
      status: "ready",
    };
    if (id) await call(() => Backend.UpdateProject(id, body));
    else {
      const newId = await call(() => Backend.CreateProject(body));
      state.projectId = newId;
    }
    persist();
    await loadProjects();
    renderServers();
    toast(t("saved"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function deleteProject() {
  const id = Number($("pj-id").value || 0);
  if (!id) return;
  const ok = await modal({ text: t("project.delete") + "?", ok: t("project.delete") });
  if (!ok) return;
  try {
    await call(() => Backend.DeleteProject(id, true));
    if (state.projectId === id) state.projectId = 0;
    persist();
    await loadProjects();
    renderServers();
    toast(t("deleted"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function openProject(id) {
  const p = state.projects.find((x) => x.id === id);
  if (!p) return;
  const root = (p.workPath || p.remotePath || "").trim();
  if (!root) {
    toast(t("files.select"));
    return;
  }
  state.projectId = id;
  state.cwd = root;
  state.edit = null;
  persist();
  fillSelects();
  $("term-title").textContent = p.name + " · " + root;
  if (termSessionId && termBoundProject !== id) disconnectTerm();
  showView("files");
}

function ensureTerm() {
  if (term) return;
  if (typeof Terminal === "undefined") {
    toast("xterm failed to load");
    return;
  }
  term = new Terminal({
    cursorBlink: true,
    fontFamily: '"IBM Plex Mono", ui-monospace, monospace',
    fontSize: 13,
    theme: {
      background: "#14120a",
      foreground: "#e4e4e7",
      cursor: "#facc15",
      selectionBackground: "#713f12",
    },
  });
  fit = typeof FitAddon !== "undefined" ? new FitAddon.FitAddon() : null;
  if (fit) term.loadAddon(fit);
  term.open($("term"));
  term.attachCustomKeyEventHandler((ev) => {
    if (ev.ctrlKey && ev.key === "c" && term.hasSelection && term.hasSelection()) return true;
    return true;
  });
  term.onData((d) => {
    if (termSessionId) Backend.TerminalInput(termSessionId, d).catch(() => {});
  });
  window.addEventListener("resize", () => {
    if (state.view === "terminal" && fit) {
      fit.fit();
      sendResize();
    }
  });
}

function sendResize() {
  if (!term || !termSessionId) return;
  Backend.TerminalResize(termSessionId, term.cols, term.rows).catch(() => {});
}

function setTermButtons(on) {
  $("term-connect").classList.toggle("hidden", on);
  $("term-disconnect").classList.toggle("hidden", !on);
  $("term-reconnect").classList.toggle("hidden", !on);
}

function teardownTermEvents() {
  EventsOff("terminal:output", "terminal:closed");
}

function disconnectTerm() {
  if (termSessionId) {
    Backend.StopTerminal(termSessionId);
    termSessionId = null;
  }
  termBoundProject = 0;
  teardownTermEvents();
  setTermButtons(false);
}

async function connectTerm() {
  const p = currentProject();
  if (!p) {
    toast(t("term.select"));
    return;
  }
  ensureTerm();
  if (!term) return;
  disconnectTerm();
  term.reset();
  if (fit) fit.fit();
  $("term-title").textContent = p.name + " · " + activePath();

  EventsOn("terminal:output", (ev) => {
    if (!ev || ev.sessionId !== termSessionId || !ev.dataB64) return;
    const raw = atob(ev.dataB64);
    const bytes = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
    term.write(bytes);
  });
  EventsOn("terminal:closed", (id) => {
    if (id === termSessionId) {
      termSessionId = null;
      termBoundProject = 0;
      setTermButtons(false);
      teardownTermEvents();
    }
  });

  try {
    termSessionId = await call(() => Backend.StartTerminal(p.id, term.cols, term.rows));
    termBoundProject = p.id;
    setTermButtons(true);
    sendResize();
    term.focus();
  } catch (err) {
    disconnectTerm();
    toast(errMsg(err));
  }
}

function relPath(p) {
  const root = currentProject() && currentProject().remotePath;
  if (!root) return p;
  if (p === root) return "/";
  if (p.startsWith(root + "/")) return p.slice(root.length);
  return p;
}

function crumbs(path) {
  const root = currentProject().remotePath;
  const parts = path === root ? [] : path.slice(root.length).split("/").filter(Boolean);
  let cur = root;
  let html = `<button data-cd="${esc(root)}" class="hover:text-yellow-400">root</button>`;
  for (const part of parts) {
    cur += "/" + part;
    html += ` <span class="text-zinc-600">/</span> <button data-cd="${esc(cur)}" class="hover:text-yellow-400">${esc(part)}</button>`;
  }
  return html;
}

function fmtSize(n) {
  if (n < 1024) return n + " B";
  if (n < 1048576) return (n / 1024).toFixed(1) + " KB";
  return (n / 1048576).toFixed(1) + " MB";
}

function projectHistory() {
  const pid = state.projectId;
  return (state.uploadHistory || []).filter((h) => !pid || h.projectId === pid);
}

function renderUploadHistory() {
  const items = projectHistory();
  if (!items.length) return "";
  const tiles = items
    .map(
      (h) => `
    <div class="shrink-0 w-52 border border-zinc-800 rounded-lg p-2 bg-zinc-900">
      <div class="text-[11px] ${h.ok ? "text-yellow-400" : "text-red-400"}">${esc(h.ok ? t("files.uploaded") : t("files.upload_failed"))}</div>
      <div class="text-xs truncate mt-0.5">${esc(h.label || "")}</div>
      <div class="text-[11px] text-zinc-500">${esc(h.size ? fmtSize(h.size) : "")} · ${esc(fmtWhen(h.time))}</div>
      <div class="text-[10px] text-zinc-600 font-mono truncate">${esc(h.dest || "")}</div>
    </div>`
    )
    .join("");
  return `<div class="px-4 py-2 border-b border-zinc-800">
    <div class="text-[10px] uppercase tracking-wider text-zinc-500 mb-1">${esc(t("files.history"))}</div>
    <div class="flex gap-2 overflow-x-auto pb-1">${tiles}</div>
  </div>`;
}

function renderPathPicker() {
  const files = (state.pickFiles || []).filter((f) => f.dir);
  const rows = files
    .map(
      (f) => `<button data-pick="${esc(f.path)}" class="block w-full text-left px-2 py-1.5 text-sm text-yellow-400 hover:bg-zinc-800 rounded">${esc(f.name)}/</button>`
    )
    .join("");
  return `<div class="fixed inset-0 z-40 bg-black/70 flex items-center justify-center p-4">
    <div class="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-lg p-4 max-h-[80vh] flex flex-col">
      <div class="text-sm font-medium mb-1">${esc(t("path.change"))}</div>
      <div class="text-[11px] font-mono text-zinc-500 break-all mb-3">${esc(state.pickCwd || "")}</div>
      <div class="flex-1 overflow-auto border border-zinc-800 rounded p-1 min-h-[12rem]">${rows || `<p class="px-2 py-3 text-zinc-500 text-sm">${esc(t("files.empty"))}</p>`}</div>
      <div class="mt-3 flex justify-end gap-2">
        <button id="pick-up" class="px-3 py-1.5 text-sm rounded bg-zinc-800">..</button>
        <button id="pick-cancel" class="px-3 py-1.5 text-sm rounded bg-zinc-800">${esc(t("cancel"))}</button>
        <button id="pick-ok" class="px-3 py-1.5 text-sm rounded bg-yellow-400 text-zinc-950 font-medium">${esc(t("path.select"))}</button>
      </div>
    </div>
  </div>`;
}

async function openPathPicker() {
  const p = currentProject();
  if (!p) return;
  state.pickPath = true;
  state.pickCwd = state.cwd || p.remotePath;
  await refreshPickFiles();
  renderFiles();
}

async function refreshPickFiles() {
  const p = currentProject();
  try {
    const data = await call(() => Backend.ListFiles(p.id, state.pickCwd));
    state.pickFiles = data.files || [];
  } catch (err) {
    toast(errMsg(err));
    state.pickFiles = [];
  }
}

function bindPathPicker() {
  if (!state.pickPath) return;
  $("view-files").querySelectorAll("[data-pick]").forEach((b) => {
    b.onclick = async () => {
      state.pickCwd = b.dataset.pick;
      await refreshPickFiles();
      renderFiles();
    };
  });
  if ($("pick-up")) {
    $("pick-up").onclick = async () => {
      const p = currentProject();
      const root = p.remotePath;
      if (!state.pickCwd || state.pickCwd === root) return;
      state.pickCwd = state.pickCwd.replace(/\/[^/]+$/, "") || root;
      await refreshPickFiles();
      renderFiles();
    };
  }
  if ($("pick-cancel")) {
    $("pick-cancel").onclick = () => {
      state.pickPath = false;
      renderFiles();
    };
  }
  if ($("pick-ok")) {
    $("pick-ok").onclick = async () => {
      const p = currentProject();
      try {
        const dest = await call(() => Backend.SetActivePath(p.id, state.pickCwd));
        state.cwd = dest || state.pickCwd;
        state.pickPath = false;
        await loadProjects();
        await loadFiles(state.cwd);
        toast(t("path.current"), true);
      } catch (err) {
        toast(errMsg(err));
      }
    };
  }
}

async function saveCurrentPath() {
  const p = currentProject();
  if (!p) return;
  const name = await modal({ text: t("path.name"), input: true, value: p.name });
  if (!name) return;
  try {
    await call(() => Backend.SaveRemotePath({ projectId: p.id, name, dir: state.cwd }));
    await loadProjects();
    toast(t("saved"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function testCurrentPath() {
  const p = currentProject();
  if (!p) return;
  try {
    const res = await call(() => Backend.TestPath(p.id, state.cwd));
    if (!res || !res.ok) {
      toast((res && res.error) || t("files.upload_failed"));
      return;
    }
    const msg = t("path.test_ok") + (res.url ? "\n" + res.url : "") + "\n" + (res.dest || "");
    const remove = await modal({ text: msg + "\n\n" + t("path.test_remove") + "?", ok: t("path.test_remove") });
    if (remove) {
      try {
        await call(() => Backend.RemovePathTest(p.id, state.cwd));
        toast(t("deleted"), true);
      } catch (err) {
        toast(errMsg(err));
      }
    } else {
      toast(t("path.test_ok"), true);
    }
    await loadProjects();
    await loadFiles(state.cwd);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function loadFiles(path) {
  const p = currentProject();
  const el = $("view-files");
  if (!p) {
    el.innerHTML = `<div class="p-6 text-sm text-zinc-500">${esc(t("files.select"))}</div>`;
    return;
  }
  try {
    const data = await call(() => Backend.ListFiles(p.id, path || p.workPath || p.remotePath));
    state.cwd = path || p.workPath || p.remotePath;
    state.files = data.files || [];
    Backend.SetActivePath(p.id, state.cwd).catch(() => {});
    renderFiles();
  } catch (err) {
    toast(errMsg(err));
    el.innerHTML = `<div class="p-6 text-sm text-red-400">${esc(errMsg(err))}</div>`;
  }
}

function renderFiles() {
  const p = currentProject();
  if (!p) return;
  if (state.edit) {
    renderEditor();
    return;
  }
  const parentPath = state.cwd.replace(/\/[^/]+$/, "") || p.remotePath;
  const parentRow =
    state.cwd !== p.remotePath
      ? `<tr class="border-t border-zinc-800 hover:bg-zinc-900/60">
        <td class="px-3 py-1.5" colspan="3">
          <button data-open="${esc(parentPath)}" data-dir="true" class="text-left text-zinc-500">..</button>
        </td>
      </tr>`
      : "";
  const rows = state.files
    .map(
      (f) => `
    <tr class="border-t border-zinc-800 hover:bg-zinc-900/60">
      <td class="px-3 py-1.5">
        <button data-open="${esc(f.path)}" data-dir="${f.dir}" class="text-left ${f.dir ? "text-yellow-400" : "text-zinc-200"}">${esc(f.name)}${f.dir ? "/" : ""}</button>
      </td>
      <td class="px-3 py-1.5 text-zinc-500 whitespace-nowrap">${f.dir ? "—" : fmtSize(f.size)}</td>
      <td class="px-3 py-1.5 text-right whitespace-nowrap">
        <button data-ren="${esc(f.path)}" data-name="${esc(f.name)}" class="text-zinc-500 hover:text-zinc-200 mr-2">${esc(t("files.rename"))}</button>
        ${f.dir ? "" : `<button data-dl="${esc(f.path)}" class="text-zinc-500 hover:text-zinc-200 mr-2">${esc(t("files.download"))}</button>`}
        <button data-rm="${esc(f.path)}" data-name="${esc(f.name)}" class="text-red-400/80 hover:text-red-300">${esc(t("files.delete"))}</button>
      </td>
    </tr>`
    )
    .join("");

  $("view-files").innerHTML = `
    <div class="px-4 py-2 border-b border-zinc-800 space-y-2">
      <div class="flex items-center gap-2 text-sm">
        <div class="min-w-0 flex-1">
          <div class="text-[10px] uppercase tracking-wider text-zinc-500">${esc(t("path.current"))}</div>
          <div class="truncate text-zinc-200 font-mono text-xs">${esc(state.cwd || p.remotePath)}</div>
        </div>
        <button id="f-change-path" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("path.change"))}</button>
        <button id="f-save-path" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("path.save"))}</button>
        <button id="f-test-path" class="px-2 py-1 rounded bg-yellow-400 text-zinc-950 text-xs font-medium">${esc(t("path.test"))}</button>
      </div>
      <div class="flex items-center gap-2 text-sm">
        <div class="truncate text-zinc-400 font-mono text-xs">${crumbs(state.cwd)}</div>
        <div class="ml-auto flex gap-2 flex-wrap justify-end">
          <button id="f-new-file" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("files.new_file"))}</button>
          <button id="f-new-dir" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("files.new_folder"))}</button>
          <button id="f-upload" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("files.upload"))}</button>
          <button id="f-upload-dir" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("files.upload_folder"))}</button>
          <button id="f-upload-zip" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("files.upload_zip"))}</button>
          <button id="f-refresh" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("files.refresh"))}</button>
        </div>
      </div>
    </div>
    ${renderUploadHistory()}
    <div class="flex-1 overflow-auto">
      <table class="w-full text-sm">
        <tbody>${parentRow}${rows || `<tr><td class="px-3 py-6 text-zinc-500">${esc(t("files.empty"))}</td></tr>`}</tbody>
      </table>
    </div>
    ${state.pickPath ? renderPathPicker() : ""}`;

  $("view-files").querySelectorAll("[data-cd]").forEach((b) => (b.onclick = () => loadFiles(b.dataset.cd)));
  $("view-files").querySelectorAll("[data-open]").forEach((b) => {
    b.onclick = () => (b.dataset.dir === "true" ? loadFiles(b.dataset.open) : openFile(b.dataset.open));
  });
  $("view-files").querySelectorAll("[data-ren]").forEach((b) => (b.onclick = () => renameFile(b.dataset.ren, b.dataset.name)));
  $("view-files").querySelectorAll("[data-rm]").forEach((b) => (b.onclick = () => deleteRemoteFile(b.dataset.rm, b.dataset.name)));
  $("view-files").querySelectorAll("[data-dl]").forEach((b) => {
    b.onclick = async () => {
      try {
        await call(() => Backend.DownloadFile(p.id, b.dataset.dl));
        toast(t("files.download"), true);
      } catch (err) {
        toast(errMsg(err));
      }
    };
  });
  $("f-refresh").onclick = () => loadFiles(state.cwd);
  $("f-new-file").onclick = () => createItem("file");
  $("f-new-dir").onclick = () => createItem("dir");
  $("f-upload").onclick = () => stageUpload("file");
  $("f-upload-dir").onclick = () => stageUpload("folder");
  $("f-upload-zip").onclick = () => stageUpload("zip");
  $("f-change-path").onclick = () => openPathPicker();
  $("f-save-path").onclick = () => saveCurrentPath();
  $("f-test-path").onclick = () => testCurrentPath();
  bindPathPicker();
}

async function createItem(kind) {
  const name = await modal({ text: kind === "dir" ? t("files.new_folder") : t("files.new_file"), input: true });
  if (!name) return;
  const p = currentProject();
  const input = { projectId: p.id, dir: state.cwd, name };
  try {
    if (kind === "dir") await call(() => Backend.Mkdir(input));
    else await call(() => Backend.CreateRemoteFile(input));
    await loadFiles(state.cwd);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function renameFile(path, name) {
  const next = await modal({ text: t("files.rename") + " " + name, input: true, value: name });
  if (!next || next === name) return;
  try {
    await call(() => Backend.RenameFile({ projectId: currentProject().id, path, name: next }));
    await loadFiles(state.cwd);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function deleteRemoteFile(path, name) {
  const ok = await modal({ text: t("files.delete") + " " + name + "?", ok: t("files.delete") });
  if (!ok) return;
  try {
    await call(() => Backend.DeleteFile({ projectId: currentProject().id, path, confirm: true }));
    await loadFiles(state.cwd);
    toast(t("deleted"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

function setUploadBar(done, total) {
  const wrap = $("upload-wrap");
  const bar = $("upload-bar");
  if (!total) {
    wrap.classList.add("hidden");
    bar.style.width = "0";
    return;
  }
  wrap.classList.remove("hidden");
  bar.style.width = Math.min(100, Math.round((done / total) * 100)) + "%";
}

function hideUploadPanel() {
  const panel = $("upload-panel");
  if (panel) panel.classList.add("hidden");
  $("upload-arrows").classList.add("hidden");
  $("upload-status").classList.add("hidden");
  $("upload-actions").classList.remove("hidden");
  setUploadBar(0, 0);
}

function showUploadPreview(prev) {
  $("upload-title").textContent = t("files.upload_preview");
  $("upload-dest").textContent = t("files.upload_dest") + ": " + (prev.dest || activePath());
  $("upload-counts").textContent =
    (prev.files || 0) + " " + t("files.upload_files") + ", " + (prev.folders || 0) + " " + t("files.upload_folders") +
    (prev.bytes ? " · " + fmtSize(prev.bytes) : "");
  const items = (prev.items || []).slice(0, 6);
  $("upload-list").innerHTML = items.length
    ? items
        .map((it) => `<div class="px-2 py-1 border-b border-zinc-800 truncate">${esc(it.rel)}</div>`)
        .join("") +
      (prev.files > items.length ? `<div class="px-2 py-1 text-zinc-600">+ ${prev.files - items.length}</div>` : "")
    : "";
  $("upload-status").classList.add("hidden");
  $("upload-arrows").classList.add("hidden");
  $("upload-actions").classList.remove("hidden");
  $("upload-go").disabled = !prev.files;
  $("upload-panel").classList.remove("hidden");
}

async function stageUpload(kind) {
  const p = currentProject();
  if (!p) return;
  try {
    const dest = activePath();
    const fn =
      kind === "folder"
        ? () => Backend.StageUploadFolder(p.id, dest)
        : kind === "zip"
          ? () => Backend.StageUploadZip(p.id, dest)
          : () => Backend.StageUploadFile(p.id, dest);
    const prev = await call(fn);
    if (!prev || prev.skipped) return;
    if (prev.error) {
      toast(prev.error);
      return;
    }
    showUploadPreview(prev);
  } catch (err) {
    toast(errMsg(err));
  }
}

function onUploadProgress(ev) {
  if (!ev) return;
  setUploadBar(ev.done || 0, ev.total || 1);
  if (ev.path) {
    $("upload-status").textContent = t("files.uploading") + " " + (ev.done || 0) + "/" + (ev.total || 1) + "  " + ev.path;
  }
}

async function confirmStagedUpload() {
  const p = currentProject();
  $("upload-actions").classList.add("hidden");
  $("upload-arrows").classList.remove("hidden");
  $("upload-status").classList.remove("hidden");
  $("upload-status").textContent = t("files.uploading");
  try {
    EventsOn("upload:progress", onUploadProgress);
    const res = await call(() => Backend.ConfirmUpload());
    if (res && res.skipped) {
      hideUploadPanel();
      return;
    }
    state.uploadHistory = [
      {
        projectId: p && p.id,
        ok: !!(res && res.ok),
        label: (res && res.label) || "",
        size: (res && res.bytes) || 0,
        time: (res && res.time) || new Date().toISOString(),
        dest: (res && res.dest) || activePath(),
      },
      ...(state.uploadHistory || []),
    ];
    persistHistory();
    hideUploadPanel();
    await loadProjects();
    await loadFiles(state.cwd);
    if (res && res.failed) toast((res.failed || 0) + " failed");
    else toast(t("files.upload_done"), true);
  } catch (err) {
    toast(errMsg(err));
    $("upload-actions").classList.remove("hidden");
    $("upload-arrows").classList.add("hidden");
  } finally {
    EventsOff("upload:progress");
    setUploadBar(0, 0);
  }
}

if ($("upload-cancel")) {
  $("upload-cancel").onclick = async () => {
    try {
      await call(() => Backend.CancelUpload());
    } catch (_) {}
    hideUploadPanel();
  };
}
if ($("upload-go")) $("upload-go").onclick = () => confirmStagedUpload();

function modeFor(path) {
  const n = path.toLowerCase();
  if (n.endsWith(".blade.php") || n.endsWith(".php")) return "application/x-httpd-php";
  if (n.endsWith(".js") || n.endsWith(".mjs") || n.endsWith(".cjs")) return "javascript";
  if (n.endsWith(".json")) return { name: "javascript", json: true };
  if (n.endsWith(".html") || n.endsWith(".htm")) return "htmlmixed";
  if (n.endsWith(".css")) return "css";
  if (n.endsWith(".yml") || n.endsWith(".yaml")) return "yaml";
  if (n.endsWith(".sql")) return "sql";
  if (n.endsWith(".md") || n.endsWith(".markdown")) return "markdown";
  return null;
}

async function openFile(path) {
  try {
    const data = await call(() => Backend.ReadFile(currentProject().id, path));
    state.edit = { path, content: data.content };
    state.dirty = false;
    renderEditor();
  } catch (err) {
    toast(errMsg(err));
  }
}

function renderEditor() {
  const ed = state.edit;
  $("view-files").innerHTML = `
    <div class="flex items-center gap-2 px-4 py-2 border-b border-zinc-800 text-sm">
      <button id="ed-back" class="px-2 py-1 rounded bg-zinc-800 text-xs">${esc(t("nav.files"))}</button>
      <span class="font-mono text-xs text-zinc-400 truncate">${esc(relPath(ed.path))}</span>
      <span id="ed-dirty" class="text-amber-400 text-xs ${state.dirty ? "" : "hidden"}">unsaved</span>
      <button id="ed-save" class="ml-auto px-3 py-1 rounded bg-yellow-400 text-zinc-950 text-xs font-medium">${esc(t("notes.save"))}</button>
    </div>
    <div id="editor" class="flex-1 min-h-0"></div>`;
  $("ed-back").onclick = async () => {
    if (state.dirty) {
      const ok = await modal({ text: "Discard unsaved changes?", ok: "Discard" });
      if (!ok) return;
    }
    state.edit = null;
    renderFiles();
  };
  $("ed-save").onclick = saveEditor;
  if (typeof CodeMirror === "undefined") {
    $("editor").innerHTML = `<textarea id="ed-plain" class="w-full h-full bg-zinc-950 p-3 font-mono text-sm">${esc(ed.content)}</textarea>`;
    $("ed-plain").oninput = () => {
      state.dirty = true;
      $("ed-dirty").classList.remove("hidden");
    };
    return;
  }
  cm = CodeMirror($("editor"), {
    value: ed.content,
    mode: modeFor(ed.path),
    theme: "material-darker",
    lineNumbers: true,
    indentUnit: 4,
    lineWrapping: true,
  });
  cm.on("change", () => {
    state.dirty = true;
    $("ed-dirty").classList.remove("hidden");
  });
  requestAnimationFrame(() => cm.refresh());
}

async function saveEditor() {
  if (!state.edit) return;
  const content = cm ? cm.getValue() : $("ed-plain").value;
  try {
    await call(() => Backend.WriteFile({ projectId: currentProject().id, path: state.edit.path, content }));
    state.edit.content = content;
    state.dirty = false;
    $("ed-dirty").classList.add("hidden");
    toast(t("saved"), true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function renderCommands() {
  const p = currentProject();
  const el = $("view-commands");
  if (!p) {
    el.innerHTML = `<p class="text-sm text-zinc-500">${esc(t("commands.empty"))}</p>`;
    return;
  }
  try {
    state.commands = (await call(() => Backend.ListQuickCommands(p.id))) || [];
  } catch (err) {
    toast(errMsg(err));
    state.commands = [];
  }
  const groups = {};
  state.commands.forEach((c) => {
    if (!groups[c.group]) groups[c.group] = [];
    groups[c.group].push(c);
  });
  const panels = Object.keys(groups)
    .map((g) => {
      const rows = groups[g]
        .map(
          (c) => `
        <div class="flex items-center gap-2 py-1">
          <span class="text-sm flex-1">${esc(c.label)}</span>
          <button data-cmd="${esc(c.key)}" data-des="${c.destructive ? "1" : "0"}" class="px-2 py-1 text-xs rounded bg-yellow-400 text-zinc-950 font-medium">${esc(t("commands.run"))}</button>
        </div>`
        )
        .join("");
      return `
      <details class="border border-zinc-800 rounded-lg mb-2" open>
        <summary class="px-3 py-2 text-sm cursor-pointer select-none">${esc(g)}</summary>
        <div class="px-3 pb-2">${rows}</div>
      </details>`;
    })
    .join("");
  const out = state.cmdOut[p.id] || "";
  el.innerHTML = `
    <div class="max-w-3xl">
      <h1 class="text-lg font-medium mb-1">${esc(t("commands.title"))}</h1>
      <p class="text-xs text-zinc-500 mb-4 font-mono">${esc(p.name)} · ${esc(activePath() || p.remotePath)}</p>
      ${panels || `<p class="text-sm text-zinc-500">${esc(t("commands.empty"))}</p>`}
      <pre id="cmd-out" class="mt-4 p-3 bg-zinc-900 border border-zinc-800 rounded text-xs font-mono whitespace-pre-wrap max-h-64 overflow-auto">${esc(out)}</pre>
    </div>`;
  el.querySelectorAll("[data-cmd]").forEach((b) => {
    b.onclick = () => runCmd(b.dataset.cmd, b.dataset.des === "1");
  });
}

async function runCmd(key, destructive) {
  if (destructive) {
    const ok = await modal({ text: t("commands.confirm"), ok: t("commands.run") });
    if (!ok) return;
  }
  const p = currentProject();
  try {
    const res = await call(() => Backend.RunQuickCommand(p.id, key, !!destructive));
    const text = [res.output, res.error].filter(Boolean).join("\n") || (res.ok ? "ok" : "");
    state.cmdOut[p.id] = text;
    const out = $("cmd-out");
    if (out) out.textContent = text;
    if (res.error && !res.ok) toast(res.error);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function renderNotes() {
  const p = currentProject();
  const el = $("view-notes");
  try {
    state.notes = (await call(() => Backend.ListAllNotes())) || [];
  } catch (err) {
    toast(errMsg(err));
    state.notes = [];
  }
  const groups = {};
  state.notes.forEach((n) => {
    const cat = n.category || t("notes.general");
    if (!groups[cat]) groups[cat] = [];
    groups[cat].push(n);
  });
  const cats = Object.keys(groups).sort((a, b) => a.localeCompare(b));
  const open = state.noteOpen;
  const sections = cats
    .map((cat) => {
      const items = groups[cat]
        .map((n) => {
          const expanded = open === n.id;
          return `
      <div class="border border-zinc-800 rounded-lg p-3 mb-2">
        <div class="flex items-center gap-2">
          <div class="text-sm font-medium truncate flex-1">${esc(n.title)}</div>
          <button data-nexp="${n.id}" class="text-xs text-zinc-400">${esc(t("notes.expand"))}</button>
          <button data-ncopy="${n.id}" class="text-xs text-zinc-400">${esc(t("notes.copy"))}</button>
          <button data-nedit="${n.id}" class="text-xs text-zinc-400">${esc(t("notes.edit"))}</button>
          <button data-ndel="${n.id}" class="text-xs text-red-400">${esc(t("notes.delete"))}</button>
        </div>
        <pre class="mt-2 text-xs text-zinc-400 font-mono whitespace-pre-wrap max-h-32 overflow-auto ${expanded ? "max-h-96" : ""}">${esc(n.content)}</pre>
      </div>`;
        })
        .join("");
      return `<div class="mb-6"><h2 class="text-xs uppercase tracking-wider text-yellow-400 mb-2">${esc(cat)}</h2>${items}</div>`;
    })
    .join("");

  const hint = p ? p.name : t("notes.general");
  el.innerHTML = `
    <div class="max-w-3xl">
      <div class="flex items-center mb-3">
        <h1 class="text-lg font-medium">${esc(t("notes.title"))}</h1>
        <button id="note-new" class="ml-auto px-3 py-1.5 text-sm rounded bg-yellow-400 text-zinc-950 font-medium">${esc(t("notes.add"))}</button>
      </div>
      <p class="text-xs text-zinc-500 mb-3">${esc(t("notes.auto_category"))}: ${esc(hint)}</p>
      <form id="note-form" class="hidden border border-zinc-800 rounded-lg p-4 mb-4">
        <input type="hidden" id="note-id" value="" />
        ${field(t("notes.note_title"), inp("note-title", "text", "", "required"))}
        ${field(t("notes.category"), inp("note-cat", "text", hint))}
        <label class="block text-xs text-zinc-400 mb-3">${esc(t("notes.content"))}
          <textarea id="note-body" rows="8" class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm font-mono"></textarea>
        </label>
        <button type="submit" class="px-3 py-1.5 text-sm rounded bg-yellow-400 text-zinc-950 font-medium">${esc(t("notes.save"))}</button>
      </form>
      ${sections || `<p class="text-sm text-zinc-500">${esc(t("notes.empty"))}</p>`}
    </div>`;

  $("note-new").onclick = () => {
    $("note-form").classList.remove("hidden");
    $("note-id").value = "";
    $("note-title").value = "";
    $("note-cat").value = p ? p.name : t("notes.general");
    $("note-body").value = "";
  };
  $("note-form").onsubmit = (e) => {
    e.preventDefault();
    saveNote();
  };
  el.querySelectorAll("[data-nexp]").forEach((b) => {
    b.onclick = () => {
      state.noteOpen = state.noteOpen === Number(b.dataset.nexp) ? null : Number(b.dataset.nexp);
      renderNotes();
    };
  });
  el.querySelectorAll("[data-ncopy]").forEach((b) => {
    b.onclick = async () => {
      const n = state.notes.find((x) => x.id === Number(b.dataset.ncopy));
      if (!n) return;
      try {
        await navigator.clipboard.writeText(n.content || "");
        toast(t("copied"), true);
      } catch {
        toast(t("copied"));
      }
    };
  });
  el.querySelectorAll("[data-nedit]").forEach((b) => {
    b.onclick = () => {
      const n = state.notes.find((x) => x.id === Number(b.dataset.nedit));
      if (!n) return;
      $("note-form").classList.remove("hidden");
      $("note-id").value = n.id;
      $("note-title").value = n.title;
      $("note-cat").value = n.category || t("notes.general");
      $("note-body").value = n.content;
    };
  });
  el.querySelectorAll("[data-ndel]").forEach((b) => {
    b.onclick = () => deleteNote(Number(b.dataset.ndel));
  });
}

async function saveNote() {
  const p = currentProject();
  const id = Number($("note-id").value || 0);
  let category = ($("note-cat") && $("note-cat").value.trim()) || "";
  if (!category) category = p ? p.name : t("notes.general");
  const body = {
    projectId: p ? p.id : 0,
    category,
    title: $("note-title").value.trim(),
    content: $("note-body").value,
  };
  try {
    if (id) await call(() => Backend.UpdateNote(id, body));
    else await call(() => Backend.CreateNote(body));
    toast(t("saved"), true);
    renderNotes();
  } catch (err) {
    toast(errMsg(err));
  }
}

async function deleteNote(id) {
  const ok = await modal({ text: t("notes.delete") + "?", ok: t("notes.delete") });
  if (!ok) return;
  try {
    await call(() => Backend.DeleteNote(id, true));
    toast(t("deleted"), true);
    renderNotes();
  } catch (err) {
    toast(errMsg(err));
  }
}

async function renderSettings() {
  let audit = [];
  try {
    audit = (await call(() => Backend.ListAudit())) || [];
  } catch (err) {
    toast(errMsg(err));
  }
  const rows =
    audit
      .map((a) => {
        const ts = a.ts ? String(a.ts).replace("T", " ").replace("Z", "") : "";
        return `
    <tr class="border-t border-zinc-800">
      <td class="px-2 py-1 text-zinc-500 whitespace-nowrap">${esc(ts)}</td>
      <td class="px-2 py-1">${esc(a.action)}</td>
      <td class="px-2 py-1 text-zinc-400 font-mono text-xs">${esc(a.detail)}</td>
    </tr>`;
      })
      .join("") || `<tr><td class="px-2 py-4 text-zinc-500" colspan="3">No activity yet</td></tr>`;

  $("view-settings").innerHTML = `
    <div class="max-w-3xl space-y-6">
      <div>
        <h1 class="text-lg font-medium mb-2">${esc(t("settings.title"))}</h1>
        <p class="text-sm text-zinc-400">${esc(t("settings.data"))} <span class="font-mono">${esc(state.meta.dataDir || "")}</span></p>
      </div>
      <div>
        <h2 class="text-sm font-medium mb-2">${esc(t("theme.title"))}</h2>
        <div class="flex gap-2">
          <button id="theme-dark" class="px-3 py-1.5 text-sm rounded ${state.theme === "dark" ? "bg-yellow-400 text-zinc-950 font-medium" : "bg-zinc-800"}">${esc(t("theme.dark"))}</button>
          <button id="theme-light" class="px-3 py-1.5 text-sm rounded ${state.theme === "light" ? "bg-yellow-400 text-zinc-950 font-medium" : "bg-zinc-800"}">${esc(t("theme.light"))}</button>
        </div>
      </div>
      <div>
        <div class="flex items-center mb-2">
          <h2 class="text-sm font-medium">${esc(t("settings.audit"))}</h2>
          <button id="audit-clear" class="ml-auto text-xs px-2 py-1 rounded bg-zinc-800">${esc(t("settings.clear"))}</button>
        </div>
        <div class="border border-zinc-800 rounded overflow-auto max-h-80">
          <table class="w-full text-xs">${rows}</table>
        </div>
      </div>
    </div>`;
  $("theme-dark").onclick = () => {
    state.theme = "dark";
    applyTheme();
    renderSettings();
  };
  $("theme-light").onclick = () => {
    state.theme = "light";
    applyTheme();
    renderSettings();
  };
  $("audit-clear").onclick = async () => {
    const ok = await modal({ text: t("settings.clear") + "?", ok: t("settings.clear") });
    if (!ok) return;
    try {
      await call(() => Backend.ClearAudit(true));
      renderSettings();
    } catch (err) {
      toast(errMsg(err));
    }
  };
}

document.querySelectorAll(".nav-btn").forEach((b) => {
  b.onclick = () => showView(b.dataset.view);
});
if ($("sel-server")) {
  $("sel-server").onchange = async (e) => {
    state.serverId = Number(e.target.value);
    state.projectId = 0;
    persist();
    disconnectTerm();
    await loadProjects();
    if (state.view === "servers") renderServers();
    if (state.view === "files") loadFiles("");
  };
}
if ($("sel-project")) {
  $("sel-project").onchange = (e) => {
    const id = Number(e.target.value);
    if (id) openProject(id);
    else {
      state.projectId = 0;
      persist();
      disconnectTerm();
    }
  };
}
if ($("term-connect")) $("term-connect").onclick = connectTerm;
if ($("term-disconnect")) $("term-disconnect").onclick = disconnectTerm;
if ($("term-reconnect")) $("term-reconnect").onclick = connectTerm;
if ($("term-clear")) {
  $("term-clear").onclick = () => {
    if (term) term.clear();
  };
}

document.addEventListener("keydown", (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === "s") {
    if (state.view === "files" && state.edit) {
      e.preventDefault();
      saveEditor();
    }
  }
});

(async function init() {
  window.__staptPainted = false;
  try {
    const r = await fetch("en.json");
    state.i18n = await r.json();
  } catch (e) {
    state.i18n = {};
  }
  document.querySelectorAll("[data-i18n]").forEach((el) => {
    el.textContent = t(el.getAttribute("data-i18n"));
  });
  loadHistory();
  applyTheme();
  try {
    showView("servers");
    window.__staptPainted = true;
  } catch (err) {
    const el = $("view-servers");
    if (el) el.textContent = "UI error: " + errMsg(err);
    return;
  }
  try {
    state.meta = (await call(() => Backend.Meta())) || {};
  } catch (err) {
    toast(errMsg(err));
  }
  await loadServers();
  await loadProjects();
  try {
    showView("servers");
    window.__staptPainted = true;
  } catch (err) {
    const el = $("view-servers");
    if (el) el.textContent = "UI error: " + errMsg(err);
  }
})();
