(() => {
  const tokenKey = "sovereign_token";

  const authSection = document.getElementById("auth");
  const gameSection = document.getElementById("game");

	const versionBadge = document.getElementById("versionBadge");
	const authGrid = document.getElementById("authGrid");
	const pwCard = document.getElementById("pwCard");

  const authMsg = document.getElementById("authMsg");
  const cmdMsg = document.getElementById("cmdMsg");

  const pilotName = document.getElementById("pilotName");
  const corpName = document.getElementById("corpName");
  const seasonName = document.getElementById("seasonName");

  const sectorId = document.getElementById("sectorId");
  const sectorName = document.getElementById("sectorName");
  const warps = document.getElementById("warps");
  const credits = document.getElementById("credits");
  const turns = document.getElementById("turns");
  const cargo = document.getElementById("cargo");
  const cargoCap = document.getElementById("cargoCap");

  const eventBox = document.getElementById("eventBox");
  const eventTitle = document.getElementById("eventTitle");
  const eventEffect = document.getElementById("eventEffect");
  const eventEnds = document.getElementById("eventEnds");
  const eventDesc = document.getElementById("eventDesc");

  const noPort = document.getElementById("noPort");
  const portPanel = document.getElementById("portPanel");

  const pOreMode = document.getElementById("pOreMode");
  const pOreQty = document.getElementById("pOreQty");
  const pOrePrice = document.getElementById("pOrePrice");
  const pOrgMode = document.getElementById("pOrgMode");
  const pOrgQty = document.getElementById("pOrgQty");
  const pOrgPrice = document.getElementById("pOrgPrice");
  const pEqMode = document.getElementById("pEqMode");
  const pEqQty = document.getElementById("pEqQty");
  const pEqPrice = document.getElementById("pEqPrice");

  const logList = document.getElementById("logList");

  const scanBtn = document.getElementById("scanBtn");
  const moveBtn = document.getElementById("moveBtn");
  const moveTo = document.getElementById("moveTo");
  const tradeBtn = document.getElementById("tradeBtn");
  const tradeAction = document.getElementById("tradeAction");
  const tradeCommodity = document.getElementById("tradeCommodity");
  const tradeQty = document.getElementById("tradeQty");

  const refreshBtn = document.getElementById("refreshBtn");
  const logoutBtn = document.getElementById("logoutBtn");

  const marketBtn = document.getElementById("marketBtn");
  const routeBtn = document.getElementById("routeBtn");
  const eventsBtn = document.getElementById("eventsBtn");

  const cmdForm = document.getElementById("cmdForm");
  const cmdInput = document.getElementById("cmdInput");

  function getToken() {
    return localStorage.getItem(tokenKey) || "";
  }

  function setToken(t) {
    if (t) localStorage.setItem(tokenKey, t);
  }

  function clearToken() {
    localStorage.removeItem(tokenKey);
  }

  function setMsg(el, text, bad) {
    el.textContent = text || "";
    el.classList.toggle("bad", !!bad);
  }

  async function api(path, options) {
    const opts = options || {};
    opts.headers = opts.headers || {};
    opts.headers["Content-Type"] = "application/json";
    const t = getToken();
    if (t) opts.headers["Authorization"] = "Bearer " + t;

    const res = await fetch(path, opts);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      const msg = data && data.error ? data.error : "Request failed";
      throw new Error(msg);
    }
    return data;
  }

	async function loadVersionBadge() {
		if (!versionBadge) return;
		try {
			const res = await fetch("/api/healthz");
			const data = await res.json().catch(() => ({}));
			if (data && data.version) {
				versionBadge.textContent = "v" + data.version;
			} else {
				versionBadge.textContent = "v--.--.--";
			}
		} catch (e) {
			versionBadge.textContent = "v--.--.--";
		}
	}

  function showGame() {
    authSection.classList.add("hidden");
    gameSection.classList.remove("hidden");
  }

  function showAuth() {
    gameSection.classList.add("hidden");
    authSection.classList.remove("hidden");
  }

	function showPasswordChange() {
		if (authGrid) authGrid.classList.add("hidden");
		if (pwCard) pwCard.classList.remove("hidden");
	}

	function hidePasswordChange() {
		if (authGrid) authGrid.classList.remove("hidden");
		if (pwCard) pwCard.classList.add("hidden");
	}

  function renderLogs(logs) {
    logList.innerHTML = "";
    if (!logs || !logs.length) {
      logList.innerHTML = "<div class='muted'>No activity yet.</div>";
      return;
    }
    for (const e of logs) {
      const div = document.createElement("div");
      div.className = "log";
      const at = new Date(e.at).toLocaleString();
      div.innerHTML = "<div class='meta'>" + at + " | " + escapeHtml(e.kind) + "</div>" +
                      "<div class='text'>" + escapeHtml(e.message) + "</div>";
      logList.appendChild(div);
    }
  }

  function escapeHtml(s) {
    return (s || "").replace(/[&<>"']/g, (c) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#039;"
    }[c]));
  }

  function updateUI(state, sector, logs) {
    if (!state) return;

    pilotName.textContent = state.username + (state.is_admin ? " (ADMIN)" : "");
    corpName.textContent = state.corp_name ? state.corp_name : "-";
    seasonName.textContent = state.season_name ? state.season_name : (state.season_id ? ("Season " + state.season_id) : "-");

    sectorId.textContent = String(state.sector_id);
    credits.textContent = String(state.credits);
    turns.textContent = String(state.turns) + " / " + String(state.turns_max);

    const totalCargo = (state.cargo_ore || 0) + (state.cargo_organics || 0) + (state.cargo_equipment || 0);
    cargo.textContent = "Ore " + state.cargo_ore + ", Org " + state.cargo_organics + ", Eq " + state.cargo_equipment;
    cargoCap.textContent = String(totalCargo) + " / " + String(state.cargo_max);

    if (sector) {
      sectorName.textContent = sector.name || "";
      warps.textContent = (sector.warps || []).join(", ") || "None";

      if (sector.event) {
        eventBox.classList.remove("hidden");
        eventTitle.textContent = sector.event.title + " [" + sector.event.kind + "]";

        if (sector.event.kind === "ANOMALY" || sector.event.kind === "LIMITED") {
          eventEffect.textContent = "Prices " + sector.event.price_percent + "% (" + sector.event.commodity + ")";
        } else if (sector.event.kind === "INVASION") {
          eventEffect.textContent = "Severity " + sector.event.severity;
        } else {
          eventEffect.textContent = "";
        }

        eventEnds.textContent = new Date(sector.event.ends_at).toLocaleString();
        eventDesc.textContent = sector.event.description || "";
      } else {
        eventBox.classList.add("hidden");
        eventTitle.textContent = "";
        eventEffect.textContent = "";
        eventEnds.textContent = "";
        eventDesc.textContent = "";
      }

      if (sector.port) {
        noPort.classList.add("hidden");
        portPanel.classList.remove("hidden");

        pOreMode.textContent = sector.port.ore_mode;
        pOreQty.textContent = String(sector.port.ore_qty) + " / " + String(sector.port.ore_base_qty);
        pOrePrice.textContent = String(sector.port.ore_price);

        pOrgMode.textContent = sector.port.organics_mode;
        pOrgQty.textContent = String(sector.port.organics_qty) + " / " + String(sector.port.organics_base_qty);
        pOrgPrice.textContent = String(sector.port.organics_price);

        pEqMode.textContent = sector.port.equipment_mode;
        pEqQty.textContent = String(sector.port.equipment_qty) + " / " + String(sector.port.equipment_base_qty);
        pEqPrice.textContent = String(sector.port.equipment_price);
      } else {
        portPanel.classList.add("hidden");
        noPort.classList.remove("hidden");
      }
    }

    renderLogs(logs || []);
  }

  async function refreshState() {
    const data = await api("/api/state", { method: "GET" });
		if (data && data.state && data.state.must_change_password) {
			showAuth();
			showPasswordChange();
			setMsg(authMsg, "Password change required.", true);
			return;
		}

		hidePasswordChange();
		updateUI(data.state, data.sector, data.logs);
		showGame();
		setMsg(cmdMsg, "", false);
  }

  async function sendCommand(cmd) {
    const data = await api("/api/command", {
      method: "POST",
      body: JSON.stringify(cmd)
    });
    updateUI(data.state, data.sector, data.logs);
    setMsg(cmdMsg, data.ok ? data.message : data.message, !data.ok);
  }

  function parseCommandLine(line) {
    const t = (line || "").trim();
    if (!t) return null;

    const parts = t.split(/\s+/).filter(Boolean);
    const head = (parts[0] || "").toUpperCase();

    if (head === "SCAN") return { type: "SCAN" };
    if (head === "HELP") return { type: "HELP" };

    if (head === "MOVE") {
      const to = parseInt(parts[1] || "0", 10);
      return { type: "MOVE", to: to };
    }

    if (head === "TRADE") {
      const action = (parts[1] || "").toUpperCase();
      const commodity = (parts[2] || "").toUpperCase();
      const quantity = parseInt(parts[3] || "0", 10);
      return { type: "TRADE", action, commodity, quantity };
    }

    if (head === "PLANET") {
      const sub = (parts[1] || "INFO").toUpperCase();
      if (sub === "COLONIZE") {
        return { type: "PLANET", action: "COLONIZE", name: parts.slice(2).join(" ") };
      }
      if (sub === "LOAD" || sub === "UNLOAD") {
        const commodity = (parts[2] || "").toUpperCase();
        const quantity = parseInt(parts[3] || "0", 10);
        return { type: "PLANET", action: sub, commodity, quantity };
      }
      if (sub === "UPGRADE" && (parts[2] || "").toUpperCase() === "CITADEL") {
        return { type: "PLANET", action: "UPGRADE_CITADEL" };
      }
      if (sub === "INFO") return { type: "PLANET", action: "INFO" };
      // fallback: PLANET <anything>
      return { type: "PLANET", action: sub };
    }

    if (head === "CORP") {
      const sub = (parts[1] || "INFO").toUpperCase();
      if (sub === "CREATE") {
        return { type: "CORP", action: "CREATE", name: parts.slice(2).join(" ") };
      }
      if (sub === "JOIN") {
        return { type: "CORP", action: "JOIN", name: parts.slice(2).join(" ") };
      }
      if (sub === "LEAVE") {
        return { type: "CORP", action: "LEAVE" };
      }
      if (sub === "SAY") {
        return { type: "CORP", action: "SAY", text: parts.slice(2).join(" ") };
      }
      if (sub === "DEPOSIT" || sub === "WITHDRAW") {
        const quantity = parseInt(parts[2] || "0", 10);
        return { type: "CORP", action: sub, quantity };
      }
      return { type: "CORP", action: "INFO" };
    }

    if (head === "MINE") {
      const sub = (parts[1] || "INFO").toUpperCase();
      if (sub === "DEPLOY") {
        const quantity = parseInt(parts[2] || "0", 10);
        return { type: "MINE", action: "DEPLOY", quantity };
      }
      if (sub === "SWEEP") {
        return { type: "MINE", action: "SWEEP" };
      }
      return { type: "MINE", action: "INFO" };
    }

    if (head === "RANKINGS") return { type: "RANKINGS" };
    if (head === "SEASON") return { type: "SEASON" };

    if (head === "MARKET") {
      const commodity = (parts[1] || "").toUpperCase();
      return { type: "MARKET", commodity };
    }
    if (head === "ROUTE") {
      const commodity = (parts[1] || "").toUpperCase();
      return { type: "ROUTE", commodity };
    }
    if (head === "EVENTS") return { type: "EVENTS" };

    return { type: head };
  }

  // Register
  document.getElementById("registerForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    setMsg(authMsg, "", false);
    try {
      const username = document.getElementById("regUser").value;
      const password = document.getElementById("regPass").value;
      const data = await api("/api/register", {
        method: "POST",
        body: JSON.stringify({ username, password })
      });
      setToken(data.token);
			if (data && data.state && data.state.must_change_password) {
				showAuth();
				showPasswordChange();
				document.getElementById("oldPass").value = password;
				setMsg(authMsg, "Password change required.", true);
				return;
			}
			hidePasswordChange();
			updateUI(data.state, data.sector, data.logs);
			showGame();
			setMsg(authMsg, "", false);
    } catch (err) {
      setMsg(authMsg, err.message, true);
    }
  });

  // Login
  document.getElementById("loginForm").addEventListener("submit", async (e) => {
    e.preventDefault();
    setMsg(authMsg, "", false);
    try {
      const username = document.getElementById("logUser").value;
      const password = document.getElementById("logPass").value;
      const data = await api("/api/login", {
        method: "POST",
        body: JSON.stringify({ username, password })
      });
      setToken(data.token);
			if (data && data.state && data.state.must_change_password) {
				showAuth();
				showPasswordChange();
				document.getElementById("oldPass").value = password;
				setMsg(authMsg, "Password change required.", true);
				return;
			}
			hidePasswordChange();
			updateUI(data.state, data.sector, data.logs);
			showGame();
			setMsg(authMsg, "", false);
    } catch (err) {
      setMsg(authMsg, err.message, true);
    }
  });

	// Change password (used for initial admin first-login flow)
	document.getElementById("changePassForm").addEventListener("submit", async (e) => {
		e.preventDefault();
		setMsg(authMsg, "", false);
		try {
			const old_password = document.getElementById("oldPass").value;
			const new_password = document.getElementById("newPass").value;
			const new_password2 = document.getElementById("newPass2").value;
			if (new_password !== new_password2) {
				setMsg(authMsg, "New passwords do not match.", true);
				return;
			}
			await api("/api/change_password", {
				method: "POST",
				body: JSON.stringify({ old_password, new_password })
			});
			document.getElementById("newPass").value = "";
			document.getElementById("newPass2").value = "";
			setMsg(authMsg, "Password updated.", false);
			await refreshState();
		} catch (err) {
			setMsg(authMsg, err.message, true);
		}
	});

  // Buttons
  scanBtn.addEventListener("click", async () => {
    try { await sendCommand({ type: "SCAN" }); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });

  moveBtn.addEventListener("click", async () => {
    const to = parseInt(moveTo.value || "0", 10);
    try { await sendCommand({ type: "MOVE", to }); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });

  tradeBtn.addEventListener("click", async () => {
    const action = tradeAction.value;
    const commodity = tradeCommodity.value;
    const quantity = parseInt(tradeQty.value || "0", 10);
    try { await sendCommand({ type: "TRADE", action, commodity, quantity }); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });

  refreshBtn.addEventListener("click", async () => {
    try { await refreshState(); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });

  logoutBtn.addEventListener("click", () => {
    clearToken();
    showAuth();
		hidePasswordChange();
    setMsg(authMsg, "Signed out.", false);
    setMsg(cmdMsg, "", false);
  });

  // Command form
  cmdForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const cmd = parseCommandLine(cmdInput.value);
    if (!cmd) return;
    cmdInput.value = "";
    try {
      await sendCommand(cmd);
    } catch (err) {
      setMsg(cmdMsg, err.message, true);
    }
  });

  // Initial boot
  (async () => {
		loadVersionBadge();
    const t = getToken();
    if (!t) {
      showAuth();
			hidePasswordChange();
      return;
    }
    try {
      await refreshState();
    } catch (err) {
      clearToken();
      showAuth();
			hidePasswordChange();
      setMsg(authMsg, "Session expired. Login again.", true);
    }
  })();

  // Quick intel buttons
  marketBtn.addEventListener("click", async () => {
    try { await sendCommand({ type: "MARKET" }); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });
  routeBtn.addEventListener("click", async () => {
    try { await sendCommand({ type: "ROUTE" }); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });
  eventsBtn.addEventListener("click", async () => {
    try { await sendCommand({ type: "EVENTS" }); } catch (e) { setMsg(cmdMsg, e.message, true); }
  });
})();
