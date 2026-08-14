(() => {
  "use strict";

  const TOKEN_KEY = "sovereign_token";
  const LEGACY_TOKEN_KEY = "token";
  const nativeFetch = window.fetch.bind(window);
  const nativeSetItem = Storage.prototype.setItem;
  const nativeRemoveItem = Storage.prototype.removeItem;
  const seenLogs = new Set();
  let passwordChangeRequired = false;

  function storedToken() {
    return localStorage.getItem(TOKEN_KEY) || localStorage.getItem(LEGACY_TOKEN_KEY) || "";
  }

  function synchronizeStoredToken() {
    const value = storedToken();
    if (!value) return;
    nativeSetItem.call(localStorage, TOKEN_KEY, value);
    nativeSetItem.call(localStorage, LEGACY_TOKEN_KEY, value);
  }

  Storage.prototype.setItem = function setItem(key, value) {
    nativeSetItem.call(this, key, value);
    if (this === localStorage && (key === TOKEN_KEY || key === LEGACY_TOKEN_KEY)) {
      nativeSetItem.call(this, TOKEN_KEY, value);
      nativeSetItem.call(this, LEGACY_TOKEN_KEY, value);
    }
  };

  Storage.prototype.removeItem = function removeItem(key) {
    nativeRemoveItem.call(this, key);
    if (this === localStorage && (key === TOKEN_KEY || key === LEGACY_TOKEN_KEY)) {
      nativeRemoveItem.call(this, TOKEN_KEY);
      nativeRemoveItem.call(this, LEGACY_TOKEN_KEY);
    }
  };

  function requestPath(input) {
    try {
      const value = typeof input === "string" ? input : input.url;
      return new URL(value, window.location.href).pathname;
    } catch {
      return "";
    }
  }

  function positiveInteger(value) {
    const parsed = Number(value);
    return Number.isInteger(parsed) && parsed > 0 ? parsed : null;
  }

  function correctedCommandPayload(payload) {
    if (!payload || typeof payload !== "object") return payload;
    const raw = document.getElementById("commandInput")?.value?.trim() || "";
    const parts = raw.split(/\s+/);
    const type = String(parts[0] || payload.type || "").toUpperCase();
    const action = String(parts[1] || payload.action || "").toUpperCase();
    const corrected = { ...payload, type };

    if (type === "PLANET") {
      corrected.action = action || "INFO";
      if (action === "COLONIZE") {
        corrected.name = parts.slice(2).join(" ");
      } else if (action === "LOAD" || action === "UNLOAD") {
        corrected.commodity = String(parts[2] || "").toUpperCase();
        corrected.quantity = positiveInteger(parts[3]) || 0;
        delete corrected.name;
      } else if (action === "UPGRADE" && String(parts[2] || "").toUpperCase() === "CITADEL") {
        corrected.action = "UPGRADE_CITADEL";
        delete corrected.name;
      }
    }

    if (type === "CORP") {
      corrected.action = action || "INFO";
      if (action === "CREATE" || action === "JOIN") {
        corrected.name = parts.slice(2).join(" ");
      } else if (action === "SAY") {
        corrected.text = parts.slice(2).join(" ");
      } else if (action === "DEPOSIT" || action === "WITHDRAW") {
        corrected.quantity = positiveInteger(parts[2]) || 0;
        delete corrected.name;
        delete corrected.text;
      }
    }

    if (type === "MARKET" || type === "ROUTE") {
      const commodity = String(parts[1] || "").toUpperCase();
      if (commodity) corrected.commodity = commodity;
    }

    return corrected;
  }

  function correctedRequest(path, init) {
    if (path !== "/api/command" || typeof init?.body !== "string") return init;
    try {
      const payload = correctedCommandPayload(JSON.parse(init.body));
      return { ...init, body: JSON.stringify(payload) };
    } catch {
      return init;
    }
  }

  function logKey(entry) {
    return `${entry?.at || ""}|${entry?.kind || ""}|${entry?.message || entry?.msg || ""}`;
  }

  function normalizeLogs(path, logs) {
    if (!Array.isArray(logs)) return logs;
    if (path === "/api/state") seenLogs.clear();

    const normalized = [];
    for (const source of logs) {
      const entry = { ...source, msg: source.message || source.msg || "" };
      const key = logKey(entry);
      if (path === "/api/command" && seenLogs.has(key)) continue;
      seenLogs.add(key);
      normalized.push(entry);
    }
    return normalized.reverse();
  }

  function normalizeSector(sector) {
    if (!sector?.event) return sector;
    const event = sector.event;
    const details = [];
    if (event.description) details.push(event.description);
    if (event.commodity) details.push(`Commodity: ${event.commodity}`);
    if (event.price_percent) details.push(`Price effect: ${event.price_percent}%`);
    return {
      ...sector,
      event: {
        ...event,
        name: event.title || event.kind || "Active event",
        effect: details.join("\n"),
      },
    };
  }

  function showPasswordChange() {
    const auth = document.getElementById("auth");
    const game = document.getElementById("game");
    const topbar = document.getElementById("topbar");
    const login = document.getElementById("loginPanel");
    const register = document.getElementById("registerPanel");
    const panel = document.getElementById("pwChangePanel");
    const message = document.getElementById("authMsg");
    if (!auth || !panel) return;
    if (game) game.style.display = "none";
    if (topbar) topbar.style.display = "none";
    if (login) login.style.display = "none";
    if (register) register.style.display = "none";
    auth.style.display = "";
    panel.style.display = "";
    if (message) message.textContent = "Password change required before gameplay.";
  }

  function normalizePayload(path, payload) {
    if (!payload || typeof payload !== "object") return payload;
    const normalized = { ...payload };
    if (Array.isArray(payload.logs)) normalized.logs = normalizeLogs(path, payload.logs);
    if (payload.sector) normalized.sector = normalizeSector(payload.sector);
    if (payload.state?.must_change_password) {
      passwordChangeRequired = true;
      queueMicrotask(showPasswordChange);
    }
    return normalized;
  }

  window.fetch = async function compatibilityFetch(input, init = {}) {
    const path = requestPath(input);
    const response = await nativeFetch(input, correctedRequest(path, init));
    const contentType = response.headers.get("content-type") || "";
    if (!contentType.includes("application/json")) return response;

    try {
      const payload = normalizePayload(path, await response.clone().json());
      return new Response(JSON.stringify(payload), {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers,
      });
    } catch {
      return response;
    }
  };

  async function submitRequiredPasswordChange(event) {
    if (!passwordChangeRequired) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    const oldPassword = document.getElementById("oldPass")?.value || "";
    const newPassword = document.getElementById("newPass")?.value || "";
    const message = document.getElementById("authMsg");
    try {
      const response = await window.fetch("/api/change_password", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${storedToken()}`,
        },
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
      });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload.error || "Password change failed");
      passwordChangeRequired = false;
      window.location.reload();
    } catch (error) {
      if (message) message.textContent = error.message || "Password change failed";
    }
  }

  async function downloadProtectedAttachment(anchor, event) {
    const url = new URL(anchor.href, window.location.href);
    if (!url.pathname.startsWith("/api/messages/attachments/")) return;
    event.preventDefault();
    const response = await nativeFetch(url.pathname, {
      headers: { Authorization: `Bearer ${storedToken()}` },
    });
    if (!response.ok) {
      window.alert(`Download failed (HTTP ${response.status})`);
      return;
    }
    const blobURL = URL.createObjectURL(await response.blob());
    const link = document.createElement("a");
    link.href = blobURL;
    link.download = anchor.textContent?.trim() || "attachment";
    document.body.appendChild(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(blobURL), 1000);
  }

  function removeInvalidSentReportActions() {
    const sent = document.getElementById("dmSent");
    if (!sent) return;
    for (const button of sent.querySelectorAll("button")) {
      if (button.textContent?.trim() === "Report") button.remove();
    }
  }

  synchronizeStoredToken();
  document.addEventListener("DOMContentLoaded", () => {
    document.getElementById("pwChangeForm")?.addEventListener("submit", submitRequiredPasswordChange, true);
    document.addEventListener("click", (event) => {
      const anchor = event.target.closest?.('a[href^="/api/messages/attachments/"]');
      if (anchor) void downloadProtectedAttachment(anchor, event);
    }, true);

    const sent = document.getElementById("dmSent");
    if (sent) new MutationObserver(removeInvalidSentReportActions).observe(sent, { childList: true, subtree: true });
  });
})();
