import * as Backend from "./wailsjs/go/main/App.js";
import { EventsOn, EventsOff } from "./wailsjs/runtime/runtime.js";

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
};

let term, fit, cm;
let termSessionId = null;

const $ = (id) => document.getElementById(id);

function errMsg(err) {
  if (!err) return "unknown error";
  if (typeof err === "string") return err;
  return err.message || String(err);
}

async function call(fn) {
  try {
    return await fn();
  } catch (err) {
    const e = new Error(errMsg(err));
    throw e;
  }
}

function toast(msg, ok) {
  const el = $("toast");
  el.textContent = msg;
  el.classList.remove("hidden", "border-red-800", "border-yellow-800");
  el.classList.add(ok ? "border-yellow-800" : "border-red-800");
  clearTimeout(toast._t);
  toast._t = setTimeout(() => el.classList.add("hidden"), 3500);
}

function modal({ text, input, value = "", ok = "OK" }) {
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
  return String(s ?? "")
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
  ss.innerHTML =
    `<option value="0">— select —</option>` +
    state.servers
      .map(
        (s) =>
          `<option value="${s.id}" ${s.id === state.serverId ? "selected" : ""}>${esc(s.name)} (${esc(s.host)})</option>`
      )
      .join("");
  const ps = $("sel-project");
  ps.innerHTML =
    `<option value="0">— select —</option>` +
    state.projects
      .map(
        (p) =>
          `<option value="${p.id}" ${p.id === state.projectId ? "selected" : ""}>${esc(p.name)}</option>`
      )
      .join("");
}

async function loadServers() {
  state.servers = (await call(() => Backend.ListServers())) || [];
  if (state.serverId && !state.servers.some((s) => s.id === state.serverId)) state.serverId = 0;
  fillSelects();
}

async function loadProjects() {
  if (!state.serverId) {
    state.projects = [];
    state.projectId = 0;
    fillSelects();
    return;
  }
  state.projects = (await call(() => Backend.ListProjects(state.serverId))) || [];
  if (state.projectId && !state.projects.some((p) => p.id === state.projectId)) state.projectId = 0;
  fillSelects();
  persist();
}

function showView(name) {
  state.view = name;
  document.querySelectorAll(".nav-btn").forEach((b) => b.classList.toggle("active", b.dataset.view === name));
  ["servers", "terminal", "files", "settings"].forEach((v) => {
    const el = $("view-" + v);
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
  }
  if (name === "files") loadFiles(state.cwd || (currentProject() && currentProject().remotePath) || "");
  if (name === "settings") renderSettings();
}

function field(label, inner) {
  return `<label class="block text-xs text-zinc-400 mb-3">${esc(label)}${inner}</label>`;
}
function inp(id, type, value, extra = "") {
  return `<input id="${id}" type="${type}" value="${esc(value || "")}" ${extra} class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm" />`;
}

function renderProjectsPanel() {
  const sv = currentServer();
  if (!sv) {
    return `<p class="text-sm text-zinc-500 mt-4">Select a server to configure allowed root paths.</p>`;
  }
  const list =
    state.projects
      .map(
        (p) => `
    <button data-pj="${p.id}" class="w-full text-left px-3 py-2 rounded border border-zinc-800 hover:border-zinc-600 mb-1 ${p.id === state.projectId ? "border-yellow-800 bg-yellow-950/40" : ""}">
      <div class="text-sm">${esc(p.name)}</div>
      <div class="text-[11px] text-zinc-500 font-mono">${esc(p.remotePath)}</div>
    </button>`
      )
      .join("") || `<p class="text-sm text-zinc-500">No root paths yet.</p>`;

  return `
    <div class="mt-6 pt-6 border-t border-zinc-800">
      <h2 class="text-sm font-medium mb-1">Allowed root paths</h2>
      <p class="text-xs text-zinc-500 mb-3">Terminal and file manager stay inside these directories.</p>
      ${list}
      <form id="proj-form" class="mt-4 border border-zinc-800 rounded-lg p-4">
        <h3 class="text-xs font-medium mb-3 text-zinc-400" id="proj-form-title">New root path</h3>
        <input type="hidden" id="pj-id" value="" />
        ${field("Label", inp("pj-name", "text", "", "required"))}
        ${field("Remote path", inp("pj-path", "text", "", 'required placeholder="/var/www/myproject"'))}
        <div class="flex gap-2">
          <button type="submit" class="px-3 py-1.5 text-sm rounded bg-yellow-700">Save</button>
          <button type="button" id="pj-new" class="px-3 py-1.5 text-sm rounded bg-zinc-800">Clear</button>
          <button type="button" id="pj-del" class="hidden ml-auto px-3 py-1.5 text-sm rounded bg-red-900">Delete</button>
        </div>
      </form>
    </div>`;
}

function renderServers() {
  const list =
    state.servers
      .map(
        (s) => `
    <button data-edit="${s.id}" class="w-full text-left px-3 py-2 rounded border border-zinc-800 hover:border-zinc-600 mb-1 ${s.id === state.serverId ? "border-yellow-800 bg-yellow-950/40" : ""}">
      <div class="text-sm">${esc(s.name)}</div>
      <div class="text-[11px] text-zinc-500">${esc(s.username)}@${esc(s.host)}:${s.port} · ${esc(s.authType)}</div>
    </button>`
      )
      .join("") || `<p class="text-sm text-zinc-500">No servers yet.</p>`;

  $("view-servers").innerHTML = `
    <div class="max-w-4xl grid lg:grid-cols-2 gap-6">
      <div>
        <h1 class="text-lg font-medium mb-3">Servers</h1>
        ${list}
      </div>
      <div>
        <form id="server-form" class="border border-zinc-800 rounded-lg p-4">
          <h2 class="text-sm font-medium mb-3" id="server-form-title">New server</h2>
          <input type="hidden" id="sv-id" value="" />
          ${field("Name", inp("sv-name", "text", "", "required"))}
          ${field("Host / domain / IP", inp("sv-host", "text", "", 'required autocomplete="off"'))}
          ${field("Port", inp("sv-port", "number", "22", 'min="1" max="65535"'))}
          ${field("Username", inp("sv-user", "text", "", 'required autocomplete="off"'))}
          <label class="block text-xs text-zinc-400 mb-3">Auth
            <select id="sv-auth" class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm">
              <option value="password">Password</option>
              <option value="key">SSH private key</option>
            </select>
          </label>
          <div id="sv-pass-wrap">${field("Password", inp("sv-pass", "password", "", 'autocomplete="new-password"'))}</div>
          <div id="sv-key-wrap" class="hidden">
            ${field("Private key", `<textarea id="sv-key" rows="6" class="mt-1 w-full bg-zinc-900 border border-zinc-800 rounded px-2 py-1.5 text-sm font-mono" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"></textarea>`)}
            ${field("Passphrase (optional)", inp("sv-phrase", "password", "", 'autocomplete="new-password"'))}
          </div>
          <p id="sv-keep" class="hidden text-[11px] text-zinc-500 mb-3">Leave password/key blank to keep stored credentials.</p>
          <label id="sv-forget-wrap" class="hidden text-xs text-zinc-400 mb-3">
            <input id="sv-forget" type="checkbox" class="mr-1 align-middle" /> Forget saved host key
          </label>
          <div class="flex gap-2 flex-wrap">
            <button type="button" id="sv-test" class="px-3 py-1.5 text-sm rounded bg-zinc-800">Test</button>
            <button type="submit" class="px-3 py-1.5 text-sm rounded bg-yellow-700">Save</button>
            <button type="button" id="sv-new" class="px-3 py-1.5 text-sm rounded bg-zinc-800">Clear</button>
            <button type="button" id="sv-del" class="hidden ml-auto px-3 py-1.5 text-sm rounded bg-red-900">Delete</button>
          </div>
        </form>
        ${renderProjectsPanel()}
      </div>
    </div>`;

  const auth = $("sv-auth");
  auth.onchange = () => {
    const key = auth.value === "key";
    $("sv-pass-wrap").classList.toggle("hidden", key);
    $("sv-key-wrap").classList.toggle("hidden", !key);
  };

  $("view-servers").querySelectorAll("[data-edit]").forEach((btn) => {
    btn.onclick = async () => {
      state.serverId = Number(btn.dataset.edit);
      persist();
      fillSelects();
      await loadProjects();
      fillServerForm(state.serverId);
      renderServers();
    };
  });
  $("view-servers").querySelectorAll("[data-pj]").forEach((btn) => {
    btn.onclick = () => fillProjectForm(Number(btn.dataset.pj));
  });
  $("sv-new").onclick = () => fillServerForm(0);
  $("sv-test").onclick = () => testServer();
  $("sv-del").onclick = () => deleteServer();
  $("server-form").onsubmit = (e) => {
    e.preventDefault();
    saveServer();
  };
  const pf = $("proj-form");
  if (pf) {
    $("pj-new").onclick = () => fillProjectForm(0);
    $("pj-del").onclick = () => deleteProject();
    pf.onsubmit = (e) => {
      e.preventDefault();
      saveProject();
    };
  }
  auth.dispatchEvent(new Event("change"));
}

function fillServerForm(id) {
  state.hostKey = "";
  const s = state.servers.find((x) => x.id === id);
  $("sv-id").value = id || "";
  $("server-form-title").textContent = s ? "Edit server" : "New server";
  $("sv-name").value = s ? s.name : "";
  $("sv-host").value = s ? s.host : "";
  $("sv-port").value = s ? s.port : 22;
  $("sv-user").value = s ? s.username : "";
  $("sv-auth").value = s ? s.authType : "password";
  $("sv-pass").value = "";
  $("sv-key").value = "";
  $("sv-phrase").value = "";
  $("sv-keep").classList.toggle("hidden", !s);
  $("sv-del").classList.toggle("hidden", !s);
  $("sv-forget-wrap").classList.toggle("hidden", !(s && s.hasHostKey));
  $("sv-forget").checked = false;
  $("sv-auth").dispatchEvent(new Event("change"));
  if (id) {
    state.serverId = id;
    persist();
    fillSelects();
    loadProjects();
  }
}

function serverPayload(forTest) {
  const id = Number($("sv-id").value || 0);
  const auth = $("sv-auth").value;
  const body = {
    name: $("sv-name").value.trim(),
    host: $("sv-host").value.trim(),
    port: Number($("sv-port").value || 22),
    username: $("sv-user").value.trim(),
    authType: auth,
  };
  const pass = $("sv-pass").value;
  const key = $("sv-key").value;
  const phrase = $("sv-phrase").value;
  if (forTest) {
    if (pass) body.password = pass;
    if (key) body.privateKey = key;
    if (phrase) body.passphrase = phrase;
  } else {
    if (pass) body.password = pass;
    if (key) body.privateKey = key;
    if (phrase || key) body.passphrase = phrase;
    if (state.hostKey) body.hostKey = state.hostKey;
    if ($("sv-forget") && $("sv-forget").checked) body.clearHostKey = true;
  }
  return { id, body };
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

async function testServer() {
  try {
    let res = await runTest(false);
    if (res.needHostKey) {
      const ok = await modal({
        text: "Unrecognized host key:\n" + (res.fingerprintUI || "") + "\n\nTrust and continue?",
        ok: "Trust",
      });
      if (!ok) return;
      res = await runTest(true);
    }
    if (res.error && !res.ok) {
      toast(res.error);
      return;
    }
    if (res.hostKey) state.hostKey = res.hostKey;
    toast("Connected · " + (res.fingerprint || "ok"), true);
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
    toast("Server saved", true);
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
  const ok = await modal({ text: "Delete this server and its root paths?", ok: "Delete" });
  if (!ok) return;
  try {
    await call(() => Backend.DeleteServer(id, true));
    if (state.serverId === id) state.serverId = 0;
    state.projectId = 0;
    persist();
    await loadServers();
    await loadProjects();
    renderServers();
    toast("Server deleted", true);
  } catch (err) {
    toast(errMsg(err));
  }
}

function fillProjectForm(id) {
  const p = state.projects.find((x) => x.id === id);
  if (!$("pj-id")) return;
  $("pj-id").value = id || "";
  $("proj-form-title").textContent = p ? "Edit root path" : "New root path";
  $("pj-name").value = p ? p.name : "";
  $("pj-path").value = p ? p.remotePath : "";
  $("pj-del").classList.toggle("hidden", !p);
  if (p) {
    state.projectId = p.id;
    persist();
    fillSelects();
  }
}

async function saveProject() {
  try {
    const id = Number($("pj-id").value || 0);
    const body = {
      serverId: state.serverId,
      name: $("pj-name").value.trim(),
      remotePath: $("pj-path").value.trim(),
    };
    if (id) await call(() => Backend.UpdateProject(id, body));
    else {
      const newId = await call(() => Backend.CreateProject(body));
      state.projectId = newId;
    }
    persist();
    await loadProjects();
    renderServers();
    toast("Root path saved", true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function deleteProject() {
  const id = Number($("pj-id").value || 0);
  if (!id) return;
  const ok = await modal({ text: "Delete this root path profile? Remote files are not deleted.", ok: "Delete" });
  if (!ok) return;
  try {
    await call(() => Backend.DeleteProject(id, true));
    if (state.projectId === id) state.projectId = 0;
    persist();
    await loadProjects();
    renderServers();
    toast("Deleted", true);
  } catch (err) {
    toast(errMsg(err));
  }
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
      background: "#09090b",
      foreground: "#e4e4e7",
      cursor: "#facc15",
      selectionBackground: "#713f12",
    },
  });
  fit = typeof FitAddon !== "undefined" ? new FitAddon.FitAddon() : null;
  if (fit) term.loadAddon(fit);
  term.open($("term"));
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
}

function teardownTermEvents() {
  EventsOff("terminal:output", "terminal:closed");
}

function disconnectTerm() {
  if (termSessionId) {
    Backend.StopTerminal(termSessionId);
    termSessionId = null;
  }
  teardownTermEvents();
  setTermButtons(false);
}

async function connectTerm() {
  const p = currentProject();
  if (!p) {
    toast("Select a server and root path");
    return;
  }
  ensureTerm();
  if (!term) return;
  disconnectTerm();
  term.reset();
  if (fit) fit.fit();
  $("term-title").textContent = p.name + " · " + p.remotePath;

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
      setTermButtons(false);
      teardownTermEvents();
    }
  });

  try {
    termSessionId = await call(() => Backend.StartTerminal(p.id, term.cols, term.rows));
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

async function loadFiles(path) {
  const p = currentProject();
  const el = $("view-files");
  if (!p) {
    el.innerHTML = `<div class="p-6 text-sm text-zinc-500">Select a server and root path.</div>`;
    return;
  }
  try {
    const data = await call(() => Backend.ListFiles(p.id, path || p.remotePath));
    state.cwd = path || p.remotePath;
    state.files = data.files || [];
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
        <button data-ren="${esc(f.path)}" data-name="${esc(f.name)}" class="text-zinc-500 hover:text-zinc-200 mr-2">rename</button>
        ${f.dir ? "" : `<button data-dl="${esc(f.path)}" class="text-zinc-500 hover:text-zinc-200 mr-2">download</button>`}
        <button data-rm="${esc(f.path)}" data-name="${esc(f.name)}" class="text-red-400/80 hover:text-red-300">delete</button>
      </td>
    </tr>`
    )
    .join("");

  $("view-files").innerHTML = `
    <div class="flex items-center gap-2 px-4 py-2 border-b border-zinc-800 text-sm">
      <div class="truncate text-zinc-400 font-mono text-xs">${crumbs(state.cwd)}</div>
      <div class="ml-auto flex gap-2">
        <button id="f-new-file" class="px-2 py-1 rounded bg-zinc-800 text-xs">New file</button>
        <button id="f-new-dir" class="px-2 py-1 rounded bg-zinc-800 text-xs">New folder</button>
        <button id="f-upload" class="px-2 py-1 rounded bg-zinc-800 text-xs">Upload</button>
        <button id="f-refresh" class="px-2 py-1 rounded bg-zinc-800 text-xs">Refresh</button>
      </div>
    </div>
    <div class="flex-1 overflow-auto">
      <table class="w-full text-sm">
        <tbody>${parentRow}${rows || `<tr><td class="px-3 py-6 text-zinc-500">Empty folder</td></tr>`}</tbody>
      </table>
    </div>`;

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
        toast("Downloaded", true);
      } catch (err) {
        toast(errMsg(err));
      }
    };
  });
  $("f-refresh").onclick = () => loadFiles(state.cwd);
  $("f-new-file").onclick = () => createItem("file");
  $("f-new-dir").onclick = () => createItem("dir");
  $("f-upload").onclick = () => uploadRemote();
}

async function createItem(kind) {
  const name = await modal({ text: kind === "dir" ? "Folder name" : "File name", input: true });
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
  const next = await modal({ text: "Rename " + name, input: true, value: name });
  if (!next || next === name) return;
  try {
    await call(() =>
      Backend.RenameFile({ projectId: currentProject().id, path, name: next })
    );
    await loadFiles(state.cwd);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function deleteRemoteFile(path, name) {
  const ok = await modal({ text: "Delete " + name + "?", ok: "Delete" });
  if (!ok) return;
  try {
    await call(() =>
      Backend.DeleteFile({ projectId: currentProject().id, path, confirm: true })
    );
    await loadFiles(state.cwd);
    toast("Deleted", true);
  } catch (err) {
    toast(errMsg(err));
  }
}

async function uploadRemote() {
  try {
    await call(() => Backend.UploadFile(currentProject().id, state.cwd));
    await loadFiles(state.cwd);
    toast("Uploaded", true);
  } catch (err) {
    toast(errMsg(err));
  }
}

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
      <button id="ed-back" class="px-2 py-1 rounded bg-zinc-800 text-xs">Files</button>
      <span class="font-mono text-xs text-zinc-400 truncate">${esc(relPath(ed.path))}</span>
      <span id="ed-dirty" class="text-amber-400 text-xs ${state.dirty ? "" : "hidden"}">unsaved</span>
      <button id="ed-save" class="ml-auto px-3 py-1 rounded bg-yellow-700 text-xs">Save</button>
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
    await call(() =>
      Backend.WriteFile({ projectId: currentProject().id, path: state.edit.path, content })
    );
    state.edit.content = content;
    state.dirty = false;
    $("ed-dirty").classList.add("hidden");
    toast("Saved", true);
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
        <h1 class="text-lg font-medium mb-2">Settings</h1>
        <p class="text-sm text-zinc-400">Native desktop app (Wails). No browser or public HTTP UI.</p>
        <p class="text-sm text-zinc-400 mt-1">Data directory <span class="font-mono">${esc(state.meta.dataDir || "")}</span></p>
        <p class="text-xs text-zinc-500 mt-2">Credentials are encrypted locally and never exposed to the UI after save. Terminal keystrokes are not logged.</p>
      </div>
      <div>
        <div class="flex items-center mb-2">
          <h2 class="text-sm font-medium">Audit</h2>
          <button id="audit-clear" class="ml-auto text-xs px-2 py-1 rounded bg-zinc-800">Clear</button>
        </div>
        <div class="border border-zinc-800 rounded overflow-auto max-h-80">
          <table class="w-full text-xs">${rows}</table>
        </div>
      </div>
    </div>`;
  $("audit-clear").onclick = async () => {
    const ok = await modal({ text: "Clear local audit history?", ok: "Clear" });
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
$("sel-server").onchange = async (e) => {
  state.serverId = Number(e.target.value);
  state.projectId = 0;
  persist();
  disconnectTerm();
  await loadProjects();
  if (state.view === "servers") renderServers();
  if (state.view === "files") loadFiles("");
};
$("sel-project").onchange = (e) => {
  state.projectId = Number(e.target.value);
  persist();
  disconnectTerm();
  const p = currentProject();
  state.cwd = p ? p.remotePath : "";
  $("term-title").textContent = p ? p.name + " · " + p.remotePath : "Select a root path";
  if (state.view === "files") loadFiles(state.cwd);
};
$("term-connect").onclick = connectTerm;
$("term-disconnect").onclick = disconnectTerm;

document.addEventListener("keydown", (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === "s") {
    if (state.view === "files" && state.edit) {
      e.preventDefault();
      saveEditor();
    }
  }
});

(async function init() {
  try {
    state.meta = (await call(() => Backend.Meta())) || {};
  } catch (err) {
    toast(errMsg(err));
  }
  await loadServers();
  await loadProjects();
  showView("servers");
})();
