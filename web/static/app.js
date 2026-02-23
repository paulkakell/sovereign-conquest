(() => {
  const $ = (id) => document.getElementById(id);
  const show = (el, on = true) => {
    if (!el) return;
    el.style.display = on ? "" : "none";
  };

  let token = localStorage.getItem("token") || "";
  let currentPlayer = null;
  let currentSector = null;
  let unreadTimer = null;
  let activePage = "game"; // game | messages | adminMap
  let activeMsgTab = "inbox"; // inbox | sent

  // Auth UI
  const auth = $("auth");
  const tabLogin = $("tabLogin");
  const tabRegister = $("tabRegister");
  const loginPanel = $("loginPanel");
  const registerPanel = $("registerPanel");
  const pwChangePanel = $("pwChangePanel");
  const authMsg = $("authMsg");

  // Game UI
  const game = $("game");
  const topbar = $("topbar");
  const pilotName = $("pilotName");
  const rankName = $("rankName");
  const xpValue = $("xpValue");
  const corpName = $("corpName");
  const seasonName = $("seasonName");
  const sectorName = $("sectorName");
  const credits = $("credits");
  const turns = $("turns");
  const cargo = $("cargo");
  const cargoCap = $("cargoCap");

  const messagesNavBtn = $("messagesNavBtn");
  const msgBadge = $("msgBadge");
  const adminMapBtn = $("adminMapBtn");
  const refreshBtn = $("refreshBtn");
  const logoutBtn = $("logoutBtn");

  const discCount = $("discCount");
  const playerCount = $("playerCount");
  const corpInfo = $("corpInfo");
  const planetInfo = $("planetInfo");

  const sectorDetails = $("sectorDetails");
  const portDetails = $("portDetails");
  const eventDetails = $("eventDetails");
  const logBox = $("log");

  const commandInput = $("commandInput");
  const sendCmdBtn = $("sendCmdBtn");
  const cmdMsg = $("cmdMsg");
  const helpText = $("helpText");

  // Pages
  const pageGame = $("pageGame");
  const pageMessages = $("pageMessages");
  const pageAdminMap = $("pageAdminMap");

  // Messaging page
  const inboxTabBtn = $("inboxTabBtn");
  const sentTabBtn = $("sentTabBtn");
  const inboxPanel = $("inboxPanel");
  const sentPanel = $("sentPanel");
  const refreshInboxBtn = $("refreshInboxBtn");
  const refreshSentBtn = $("refreshSentBtn");
  const dmInbox = $("dmInbox");
  const dmSent = $("dmSent");

  const sendMsgForm = $("sendMsgForm");
  const msgTo = $("msgTo");
  const msgSubject = $("msgSubject");
  const msgBody = $("msgBody");
  const msgAttachment = $("msgAttachment");
  const relatedMessageId = $("relatedMessageId");
  const replyContext = $("replyContext");
  const replyActions = $("replyActions");
  const cancelReplyBtn = $("cancelReplyBtn");
  const msgSendStatus = $("msgSendStatus");

  // Admin map page
  const refreshAdminMapBtn = $("refreshAdminMapBtn");
  const adminMapPre = $("adminMapPre");
  const adminMapMsg = $("adminMapMsg");

  const ver = $("ver");

  function setAuthTab(which) {
    if (which === "login") {
      tabLogin.classList.add("active");
      tabRegister.classList.remove("active");
      show(loginPanel, true);
      show(registerPanel, false);
    } else {
      tabRegister.classList.add("active");
      tabLogin.classList.remove("active");
      show(registerPanel, true);
      show(loginPanel, false);
    }
    show(pwChangePanel, false);
    authMsg.textContent = "";
  }

  function escapeHtml(s) {
    return (s || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;");
  }

  function fmtTime(ts) {
    if (!ts) return "";
    try {
      return new Date(ts).toLocaleString();
    } catch {
      return String(ts);
    }
  }

  async function apiFetch(path, opts = {}) {
    const headers = opts.headers ? { ...opts.headers } : {};
    if (token) headers["Authorization"] = `Bearer ${token}`;

    let body = opts.body;
    if (opts.json !== undefined) {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify(opts.json);
    }

    const res = await fetch(path, {
      method: opts.method || "GET",
      headers,
      body,
    });

    const ct = res.headers.get("content-type") || "";
    const isJSON = ct.includes("application/json");
    const data = isJSON ? await res.json() : await res.text();

    if (!res.ok) {
      const msg = isJSON ? (data?.error || JSON.stringify(data)) : data;
      const err = new Error(msg || `HTTP ${res.status}`);
      err.status = res.status;
      err.data = data;
      throw err;
    }

    return data;
  }

  async function loadVersion() {
    try {
      const data = await apiFetch("/api/version");
      if (data?.version) ver.textContent = data.version;
    } catch {
      // ignore
    }
  }

  function showGameUI() {
    show(auth, false);
    show(game, true);
    show(topbar, true);
  }

  function showAuthUI() {
    show(game, false);
    show(topbar, false);
    show(auth, true);
  }

  function setPage(page) {
    activePage = page;
    show(pageGame, page === "game");
    show(pageMessages, page === "messages");
    show(pageAdminMap, page === "adminMap");
  }

  function updateBadge(unread) {
    const n = Number(unread || 0);
    if (n > 0) {
      msgBadge.textContent = String(n);
      show(msgBadge, true);
    } else {
      show(msgBadge, false);
    }
  }

  async function refreshUnreadCount() {
    if (!token) return;
    try {
      const data = await apiFetch("/api/messages/unread_count");
      updateBadge(data?.unread || 0);
    } catch {
      // ignore
    }
  }

  function startUnreadPolling() {
    stopUnreadPolling();
    refreshUnreadCount();
    unreadTimer = setInterval(refreshUnreadCount, 20000);
  }

  function stopUnreadPolling() {
    if (unreadTimer) {
      clearInterval(unreadTimer);
      unreadTimer = null;
    }
  }

  function renderPlayer(p) {
    currentPlayer = p;
    pilotName.textContent = p.username;

    // Rank display
    const lvl = p.level || 1;
    const rank = p.rank || "";
    rankName.textContent = `L${lvl}${rank ? " • " + rank : ""}`;

    const xp = p.xp || 0;
    const next = p.next_level_xp || 0;
    if (next > 0 && next > xp) {
      xpValue.textContent = `${xp}/${next}`;
    } else {
      xpValue.textContent = String(xp);
    }

    corpName.textContent = p.corp_name || "-";
    seasonName.textContent = p.season_name || "-";
    credits.textContent = String(p.credits ?? 0);
    turns.textContent = `${p.turns ?? 0}/${p.turns_max ?? 0}`;
    cargo.textContent = `${(p.cargo_ore ?? 0) + (p.cargo_organics ?? 0) + (p.cargo_equipment ?? 0)}`;
    cargoCap.textContent = String(p.cargo_max ?? 0);

    // Optional status placeholders
    discCount.textContent = "-";
    playerCount.textContent = "-";
    corpInfo.textContent = p.corp_name ? `${p.corp_name} (${p.corp_role || ""})` : "-";

    // Admin-only UI
    if (p.is_admin) {
      show(adminMapBtn, true);
    } else {
      show(adminMapBtn, false);
    }
  }

  function renderSector(s) {
    currentSector = s;
    sectorName.textContent = s?.name ? `${s.name} (#${s.id})` : `#${s?.id ?? "?"}`;

    const lines = [];
    lines.push(`Sector: ${s.id} ${s.name}`);
    if (s.is_protectorate) {
      lines.push(`Protectorate space: ${s.protectorate_fighters ?? 0} fighters on patrol.`);
      lines.push(`Shipyard: ${s.has_shipyard ? "available" : "-"}`);
    }
    lines.push(`Warps: ${(s.warps || []).join(", ") || "(none)"}`);
    lines.push(`Mines: ${s.mines ?? 0}`);

    if (s.planet) {
      const owner = s.planet.owner || "(unowned)";
      lines.push(`Planet: ${s.planet.name} | Owner: ${owner} | Citadel: ${s.planet.citadel_level}`);
      planetInfo.textContent = `${s.planet.name} (${owner})`;
    } else {
      lines.push("Planet: (none)");
      planetInfo.textContent = "-";
    }

    sectorDetails.textContent = lines.join("\n");

    // Port
    renderPort(s.port);

    // Event
    renderEvent(s.event);
  }

  function renderPort(p) {
    if (!p) {
      portDetails.textContent = "No port in this sector.";
      return;
    }

    const lines = [];
    lines.push(`Port: ${p.name || "(spaceport)"}`);
    lines.push(`Ore: ${p.ore_mode} Qty=${p.ore_qty} Price=${p.ore_price}`);
    lines.push(`Organics: ${p.organics_mode} Qty=${p.organics_qty} Price=${p.organics_price}`);
    lines.push(`Equipment: ${p.equipment_mode} Qty=${p.equipment_qty} Price=${p.equipment_price}`);
    portDetails.textContent = lines.join("\n");
  }

  function renderEvent(e) {
    if (!e) {
      eventDetails.textContent = "No active event.";
      return;
    }
    const lines = [];
    lines.push(`Event: ${e.name}`);
    lines.push(`Effect: ${e.effect}`);
    lines.push(`Ends: ${fmtTime(e.ends_at)}`);
    eventDetails.textContent = lines.join("\n");
  }

  function appendLogs(logs) {
    if (!Array.isArray(logs)) return;
    for (const l of logs) {
      const div = document.createElement("div");
      div.className = `logline kind-${(l.kind || "").toLowerCase()}`;
      div.textContent = `[${l.kind}] ${l.msg}`;
      logBox.prepend(div);
    }
  }

  async function refreshState() {
    const data = await apiFetch("/api/state");
    if (data?.state) renderPlayer(data.state);
    if (data?.sector) renderSector(data.sector);
    if (Array.isArray(data?.logs)) {
      logBox.innerHTML = "";
      appendLogs(data.logs);
    }
  }

  async function refreshHelp() {
    try {
      const data = await apiFetch("/api/help");
      if (data?.help) {
        helpText.textContent = data.help.join("\n");
      }
    } catch {
      // ignore
    }
  }

  function parseCommandLine(line) {
    const raw = (line || "").trim();
    if (!raw) return null;

    const parts = raw.split(/\s+/);
    const type = (parts[0] || "").toUpperCase();

    if (type === "SCAN") return { type: "SCAN" };

    if (type === "MOVE") {
      const to = Number(parts[1]);
      if (!Number.isFinite(to)) return null;
      return { type: "MOVE", to };
    }

    if (type === "TRADE") {
      const action = (parts[1] || "").toUpperCase();
      const commodity = (parts[2] || "").toUpperCase();
      const qty = Number(parts[3]);
      if (!action || !commodity || !Number.isFinite(qty)) return null;
      return { type: "TRADE", action, commodity, quantity: qty };
    }

    if (type === "PLANET") {
      const action = (parts[1] || "INFO").toUpperCase();
      const name = parts.slice(2).join(" ");
      return { type: "PLANET", action, name };
    }

    if (type === "CORP") {
      const action = (parts[1] || "INFO").toUpperCase();
      const name = parts.slice(2).join(" ");
      return { type: "CORP", action, name, text: parts.slice(2).join(" ") };
    }

    if (type === "MINE") {
      const action = (parts[1] || "INFO").toUpperCase();
      const qty = parts[2] ? Number(parts[2]) : 0;
      return { type: "MINE", action, quantity: Number.isFinite(qty) ? qty : 0 };
    }

    if (type === "SHIPYARD") {
      const action = (parts[1] || "").toUpperCase();
      const name = (parts[2] || "").toUpperCase();
      return { type: "SHIPYARD", action, name };
    }

    if (type === "HELP") return { type: "HELP" };
    if (type === "RANKINGS") return { type: "RANKINGS" };
    if (type === "SEASON") return { type: "SEASON" };
    if (type === "MARKET") return { type: "MARKET" };
    if (type === "ROUTE") return { type: "ROUTE" };
    if (type === "EVENTS") return { type: "EVENTS" };

    return null;
  }

  async function sendCommand() {
    cmdMsg.textContent = "";
    const cmd = parseCommandLine(commandInput.value);
    if (!cmd) {
      cmdMsg.textContent = "Invalid command.";
      return;
    }

    try {
      const resp = await apiFetch("/api/command", { method: "POST", json: cmd });
      if (resp?.message) {
        cmdMsg.textContent = resp.message;
      }
      if (resp?.state) renderPlayer(resp.state);
      if (resp?.sector) renderSector(resp.sector);
      if (Array.isArray(resp?.logs)) {
        appendLogs(resp.logs);
      }
      refreshUnreadCount();
    } catch (e) {
      cmdMsg.textContent = e.message || "Command failed";
      if (e.status === 401) {
        logout();
      }
    }
  }

  function setMsgTab(tab) {
    activeMsgTab = tab;
    if (tab === "inbox") {
      inboxTabBtn.classList.add("active");
      sentTabBtn.classList.remove("active");
      show(inboxPanel, true);
      show(sentPanel, false);
    } else {
      sentTabBtn.classList.add("active");
      inboxTabBtn.classList.remove("active");
      show(sentPanel, true);
      show(inboxPanel, false);
    }
  }

  async function refreshInbox() {
    dmInbox.textContent = "";
    try {
      const data = await apiFetch("/api/messages/inbox");
      const msgs = data?.messages || [];
      renderMessageList(dmInbox, msgs, "inbox");

      // Mark unread as read.
      const unread = msgs.filter((m) => !m.read_at).map((m) => m.id);
      if (unread.length > 0) {
        await apiFetch("/api/messages/mark_read", { method: "POST", json: { message_ids: unread } });
        refreshUnreadCount();
      }
    } catch (e) {
      dmInbox.textContent = e.message || "Failed to load inbox";
      if (e.status === 401) logout();
    }
  }

  async function refreshSent() {
    dmSent.textContent = "";
    try {
      const data = await apiFetch("/api/messages/sent");
      const msgs = data?.messages || [];
      renderMessageList(dmSent, msgs, "sent");
    } catch (e) {
      dmSent.textContent = e.message || "Failed to load sent";
      if (e.status === 401) logout();
    }
  }

  function setReplyContext(msg) {
    if (!msg) {
      relatedMessageId.value = "";
      replyContext.textContent = "";
      show(replyContext, false);
      show(replyActions, false);
      return;
    }

    relatedMessageId.value = String(msg.id);
    replyContext.textContent = `Replying to message #${msg.id} from ${msg.from}`;
    show(replyContext, true);
    show(replyActions, true);
  }

  function quoteBody(msg) {
    const header = `On ${fmtTime(msg.created_at)}, ${msg.from} wrote:`;
    const quoted = (msg.body || "")
      .split("\n")
      .map((l) => "> " + l)
      .join("\n");
    return `\n\n${header}\n${quoted}`;
  }

  function renderMessageList(container, messages, mode) {
    if (!messages || messages.length === 0) {
      container.textContent = "(no messages)";
      return;
    }

    container.innerHTML = "";
    for (const m of messages) {
      const item = document.createElement("div");
      item.className = "msgitem" + (!m.read_at && mode === "inbox" ? " unread" : "");

      const meta = document.createElement("div");
      meta.className = "msgmeta";
      meta.innerHTML =
        `<span><strong>${escapeHtml(m.subject || "(no subject)")}</strong></span>` +
        `<span class="small">${escapeHtml(m.kind || "")}</span>` +
        `<span class="small">${escapeHtml(m.from)} → ${escapeHtml(m.to)}</span>` +
        `<span class="small">${escapeHtml(fmtTime(m.created_at))}</span>` +
        (m.read_at ? `<span class="small">Read</span>` : mode === "inbox" ? `<span class="small">Unread</span>` : "");

      const body = document.createElement("div");
      body.className = "pre";
      body.textContent = m.body || "";

      const att = document.createElement("div");
      if (Array.isArray(m.attachments) && m.attachments.length > 0) {
        const links = m.attachments
          .map((a) => `<a href="/api/messages/attachments/${a.id}" target="_blank">${escapeHtml(a.filename)}</a>`)
          .join(" | ");
        att.innerHTML = `<div class="muted">Attachments: ${links}</div>`;
      }

      const actions = document.createElement("div");
      actions.className = "msgactions";

      const replyBtn = document.createElement("button");
      replyBtn.className = "ghost";
      replyBtn.textContent = "Reply";
      replyBtn.addEventListener("click", () => {
        setMsgTab("inbox");
        setReplyContext(m);
        msgTo.value = m.from || "";
        const subj = m.subject || "";
        msgSubject.value = subj.toLowerCase().startsWith("re:") ? subj : (subj ? `Re: ${subj}` : "Re:");
        msgBody.value = (msgBody.value || "") + quoteBody(m);
        msgBody.focus();
      });

      const delBtn = document.createElement("button");
      delBtn.className = "ghost";
      delBtn.textContent = "Delete";
      delBtn.addEventListener("click", async () => {
        if (!confirm("Delete this message?")) return;
        try {
          await apiFetch("/api/messages/delete", { method: "POST", json: { message_id: m.id } });
          if (mode === "inbox") await refreshInbox();
          else await refreshSent();
          refreshUnreadCount();
        } catch (e) {
          alert(e.message || "Delete failed");
        }
      });

      const reportBtn = document.createElement("button");
      reportBtn.className = "ghost";
      reportBtn.textContent = "Report";
      reportBtn.addEventListener("click", async () => {
        if (!confirm("Report this message as abusive/spam?")) return;
        try {
          await apiFetch("/api/messages/report", { method: "POST", json: { message_id: m.id } });
          alert("Reported.");
        } catch (e) {
          alert(e.message || "Report failed");
        }
      });

      // Only allow reply from inbox; for sent we allow delete.
      if (mode === "inbox") actions.appendChild(replyBtn);
      actions.appendChild(delBtn);
      actions.appendChild(reportBtn);

      item.appendChild(meta);
      item.appendChild(body);
      if (att.innerHTML) item.appendChild(att);
      item.appendChild(actions);

      container.appendChild(item);
    }
  }

  async function sendMessage(e) {
    e.preventDefault();
    msgSendStatus.textContent = "";

    const to = (msgTo.value || "").trim();
    const subject = (msgSubject.value || "").trim();
    const body = (msgBody.value || "").trim();
    const rel = (relatedMessageId.value || "").trim();

    if (!to || !body) {
      msgSendStatus.textContent = "To and Body are required.";
      return;
    }

    try {
      const file = msgAttachment?.files?.[0];
      if (file) {
        const fd = new FormData();
        fd.append("to_username", to);
        fd.append("subject", subject);
        fd.append("body", body);
        if (rel) fd.append("related_message_id", rel);
        fd.append("attachment", file, file.name);
        await apiFetch("/api/messages/send", { method: "POST", body: fd });
      } else {
        const payload = { to_username: to, subject, body };
        if (rel) payload.related_message_id = Number(rel);
        await apiFetch("/api/messages/send", { method: "POST", json: payload });
      }

      msgSendStatus.textContent = "Sent.";
      msgBody.value = "";
      msgAttachment.value = "";
      setReplyContext(null);

      // Refresh lists
      await refreshSent();
      await refreshUnreadCount();
    } catch (e2) {
      msgSendStatus.textContent = e2.message || "Send failed";
      if (e2.status === 401) logout();
    }
  }

  async function refreshAdminMap() {
    adminMapMsg.textContent = "";
    adminMapPre.textContent = "";
    try {
      const data = await apiFetch("/api/admin/ansi_map");
      adminMapPre.textContent = data?.map || "(empty)";
    } catch (e) {
      adminMapMsg.textContent = e.message || "Failed to load map";
      if (e.status === 401) logout();
    }
  }

  async function login(username, password) {
    authMsg.textContent = "";
    const data = await apiFetch("/api/login", { method: "POST", json: { username, password } });
    if (!data?.token) throw new Error("missing token");
    token = data.token;
    localStorage.setItem("token", token);

    // Force state load.
    await enterGame();
  }

  async function register(username, password) {
    authMsg.textContent = "";
    await apiFetch("/api/register", { method: "POST", json: { username, password } });
    await login(username, password);
  }

  async function enterGame() {
    showGameUI();
    setPage("game");

    await refreshHelp();
    await refreshState();

    startUnreadPolling();
  }

  function logout() {
    token = "";
    localStorage.removeItem("token");
    stopUnreadPolling();
    showAuthUI();
    setAuthTab("login");
  }

  // Event wiring
  tabLogin.addEventListener("click", () => setAuthTab("login"));
  tabRegister.addEventListener("click", () => setAuthTab("register"));

  $("loginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      await login($("loginUser").value, $("loginPass").value);
    } catch (err) {
      authMsg.textContent = err.message || "Login failed";
    }
  });

  $("registerForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      await register($("regUser").value, $("regPass").value);
    } catch (err) {
      authMsg.textContent = err.message || "Register failed";
    }
  });

  $("pwChangeForm")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    try {
      await apiFetch("/api/change_password", {
        method: "POST",
        json: { old_password: $("oldPass").value, new_password: $("newPass").value },
      });
      authMsg.textContent = "Password updated.";
      setAuthTab("login");
    } catch (err) {
      authMsg.textContent = err.message || "Password change failed";
    }
  });

  refreshBtn.addEventListener("click", async () => {
    try {
      await refreshState();
      refreshUnreadCount();
    } catch (e) {
      if (e.status === 401) logout();
    }
  });

  logoutBtn.addEventListener("click", () => logout());

  sendCmdBtn.addEventListener("click", sendCommand);
  commandInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      sendCommand();
    }
  });

  messagesNavBtn.addEventListener("click", async () => {
    if (activePage !== "messages") {
      setPage("messages");
      setMsgTab("inbox");
      await refreshInbox();
      await refreshSent();
    } else {
      setPage("game");
    }
  });

  adminMapBtn.addEventListener("click", async () => {
    if (activePage !== "adminMap") {
      setPage("adminMap");
      await refreshAdminMap();
    } else {
      setPage("game");
    }
  });

  inboxTabBtn.addEventListener("click", async () => {
    setMsgTab("inbox");
    await refreshInbox();
  });
  sentTabBtn.addEventListener("click", async () => {
    setMsgTab("sent");
    await refreshSent();
  });
  refreshInboxBtn.addEventListener("click", refreshInbox);
  refreshSentBtn.addEventListener("click", refreshSent);

  cancelReplyBtn.addEventListener("click", () => setReplyContext(null));

  sendMsgForm.addEventListener("submit", sendMessage);

  refreshAdminMapBtn.addEventListener("click", refreshAdminMap);

  // Init
  setAuthTab("login");
  loadVersion();

  // Resume session if token exists
  (async () => {
    if (!token) return;
    try {
      await enterGame();
    } catch {
      logout();
    }
  })();
})();
