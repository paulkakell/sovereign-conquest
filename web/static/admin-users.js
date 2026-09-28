(() => {
  "use strict";
  window.createAdminUserManager = ({ apiFetch, logout, currentUser }) => {
    const $ = id => document.getElementById(id);
    const controls = [
      ["credits", "Credits", "text", 0, "9000000000000000"],
      ["xp", "Experience (XP)", "text", 0, "9000000000000000"],
      ["ship_type", "Ship type", "select"],
      ["ship_cargo_upgrades", "Cargo upgrades", "number", 0, 20],
      ["ship_turn_upgrades", "Turn upgrades", "number", 0, 10],
      ["turns", "Turns", "number", 0, 1000000],
      ["turns_max", "Maximum turns", "number", 0, 1000000],
      ["sector_id", "Sector ID", "number", 1, 2147483647],
      ["cargo_max", "Cargo capacity", "number", 0, 1000000],
      ["cargo_ore", "Ore", "number", 0, 1000000],
      ["cargo_organics", "Organics", "number", 0, 1000000],
      ["cargo_equipment", "Equipment", "number", 0, 1000000],
      ["season_id", "Season ID", "number", 1, 2147483647],
    ];
    let selected = null;
    let cursor = "";
    let pageCursor = "";
    let listGeneration = 0;
    let editorGeneration = 0;
    let busy = false;
    let dirty = false;
    const time = value => value ? new Date(value).toLocaleString() : "Never";
    const message = (id, text, error = false) => {
      $(id).textContent = text;
      $(id).classList.toggle("bad", error);
    };
    for (const [name, label, type, min, max] of controls) {
      const container = document.createElement("label");
      container.textContent = label;
      const input = document.createElement(type === "select" ? "select" : "input");
      input.id = "editPlayer_" + name;
      if (type === "select") {
        for (const ship of ["SCOUT", "TRADER", "FREIGHTER", "INTERCEPTOR"]) {
          const option = document.createElement("option");
          option.value = ship; option.textContent = ship; input.append(option);
        }
      } else {
        input.type = type; input.required = true;
        if (type === "number") { input.min = min; input.max = max; input.step = "1"; }
        else { input.inputMode = "numeric"; input.pattern = "[0-9]+"; input.maxLength = 16; }
      }
      container.append(input); $("userPlayerInputs").append(container);
    }
    function setBusy(value) {
      busy = value;
      $("userEditFields").disabled = value;
      $("userModerationFields").disabled = value;
      $("userReload").disabled = value;
    }
    function clear() {
      selected = null; dirty = false; listGeneration++; editorGeneration++;
      $("userList").replaceChildren();
      $("userEditor").style.display = "none";
      $("userEditForm").reset(); $("userModerationForm").reset();
      for (const id of ["userListMsg", "userEditMsg", "userModerationMsg"]) message(id, "");
      setBusy(false);
    }
    async function load(after = "") {
      const generation = ++listGeneration;
      pageCursor = after;
      message("userListMsg", "Loading accounts...");
      $("usersNextPage").disabled = true;
      try {
        const params = new URLSearchParams({ q: $("userSearch").value.trim(), status: $("userStatusFilter").value, after });
        const data = await apiFetch("/api/admin/users?" + params);
        if (generation !== listGeneration) return;
        $("userList").replaceChildren();
        for (const user of data.users) {
          const row = document.createElement("tr");
          for (const value of [user.username, user.is_admin ? "Administrator" : "Player", user.effective_status]) {
            const cell = document.createElement("td"); cell.textContent = value; row.append(cell);
          }
          row.children[2].className = "user-status " + user.effective_status;
          const cell = document.createElement("td");
          const button = document.createElement("button");
          button.type = "button"; button.className = "ghost"; button.textContent = "Edit";
          button.setAttribute("aria-label", "Edit " + user.username);
          button.addEventListener("click", () => open(user.id));
          cell.append(button); row.append(cell); $("userList").append(row);
        }
        cursor = data.next_cursor;
        $("usersNextPage").disabled = !cursor;
        $("usersFirstPage").disabled = !after;
        message("userListMsg", data.users.length ? `${data.users.length} accounts shown.` : "No accounts match your search.");
      } catch (error) { if (generation === listGeneration) message("userListMsg", error.message, true); }
    }
    function render(user) {
      selected = user; dirty = false;
      $("userEditor").style.display = "";
      $("userEditorTitle").textContent = "Edit " + user.username;
      $("editUsername").value = user.username;
      $("editIsAdmin").checked = user.is_admin;
      $("editMustChange").checked = user.must_change_password;
      $("editPassword").value = ""; $("editPasswordConfirm").value = "";
      const self = currentUser()?.user_id === user.id;
      $("editIsAdmin").disabled = self;
      $("userSuspend").disabled = self; $("userBan").disabled = self;
      $("userActivate").disabled = user.effective_status === "active";
      $("userMetadata").textContent = `User ID: ${user.id}\nCreated: ${time(user.created_at)}\nPassword last changed: ${time(user.password_changed_at)}`;
      $("userPlayerFields").style.display = user.player ? "" : "none";
      $("userPlayerFields").disabled = !user.player;
      if (user.player) {
        for (const [name] of controls) $("editPlayer_" + name).value = user.player[name];
        $("userMetadata").textContent += `\nPlayer ID: ${user.player.id} | Level: ${user.player.level}\nPlayer created: ${time(user.player.created_at)} | Last turn regeneration: ${time(user.player.last_turn_regen)}`;
      }
      $("userCurrentStatus").textContent = `Status: ${user.effective_status}` +
        (user.status === "suspended" && user.suspended_until ? `\nSuspension ends: ${time(user.suspended_until)}` : "") +
        (user.moderated_at ? `\nLast access change: ${time(user.moderated_at)}\nReason: ${user.moderation_reason}` : "") +
        (self ? "\nYou cannot suspend, ban or remove your own administrator role." : "");
      $("userModerationForm").reset();
    }
    async function open(id) {
      if (busy || (dirty && !window.confirm("Discard unsaved account edits?"))) return;
      const generation = ++editorGeneration;
      // Never allow the previous account to be saved while another is loading.
      selected = null; $("userEditor").style.display = "none";
      message("userEditMsg", ""); message("userModerationMsg", "");
      try {
        const data = await apiFetch("/api/admin/users/" + encodeURIComponent(id));
        if (generation !== editorGeneration) return;
        render(data.user); $("userEditorTitle").focus();
      } catch (error) { if (generation === editorGeneration) message("userListMsg", error.message, true); }
    }
    function accountPatch() {
      const patch = { revision: selected.revision };
      const username = $("editUsername").value.trim();
      if (username !== selected.username) patch.username = username;
      if ($("editIsAdmin").checked !== selected.is_admin) patch.is_admin = $("editIsAdmin").checked;
      if ($("editMustChange").checked !== selected.must_change_password) patch.must_change_password = $("editMustChange").checked;
      const password = $("editPassword").value;
      if (password || $("editPasswordConfirm").value) {
        const bytes = new TextEncoder().encode(password).length;
        if (bytes < 8 || bytes > 72) throw new Error("New password must be 8-72 UTF-8 bytes.");
        if (password !== $("editPasswordConfirm").value) throw new Error("New passwords do not match.");
        patch.new_password = password; patch.must_change_password = $("editMustChange").checked;
      }
      const player = {};
      if (selected.player) {
        for (const [name, , type, min, max] of controls) {
          const value = $("editPlayer_" + name).value;
          if (String(selected.player[name]) === value) continue;
          if (type === "text") {
            if (!/^[0-9]+$/.test(value) || BigInt(value) > BigInt(max)) throw new Error(`${name} must be 0-${max}.`);
            player[name] = value;
          } else if (type === "number") {
            const number = Number(value);
            if (value === "" || !Number.isInteger(number) || number < min || number > max) throw new Error(`${name} must be a whole number between ${min} and ${max}.`);
            player[name] = number;
          } else player[name] = value;
        }
      }
      if (Object.keys(player).length) patch.player = player;
      return patch;
    }
    $("userEditForm").addEventListener("input", () => { dirty = true; });
    $("editPassword").addEventListener("input", () => {
      if ($("editPassword").value) $("editMustChange").checked = true;
    });
    $("userSearchForm").addEventListener("submit", event => { event.preventDefault(); void load(); });
    $("usersFirstPage").addEventListener("click", () => load());
    $("usersNextPage").addEventListener("click", () => { if (cursor) void load(cursor); });
    $("userReload").addEventListener("click", () => { if (selected) void open(selected.id); });
    $("userEditForm").addEventListener("submit", async event => {
      event.preventDefault();
      if (!selected || busy) return;
      const generation = editorGeneration;
      try {
        const patch = accountPatch();
        if (Object.keys(patch).length === 1) { message("userEditMsg", "No changes to save."); return; }
        if ((patch.is_admin !== undefined || patch.new_password) && !window.confirm(`Save account and access changes for ${selected.username}? Existing sessions will end.`)) return;
        setBusy(true); message("userEditMsg", "Saving...");
        const data = await apiFetch("/api/admin/users/" + encodeURIComponent(selected.id), { method: "PATCH", json: patch });
        if (generation !== editorGeneration) return;
        if (data.sign_in_again) { logout(); $("authMsg").textContent = "Account saved. Sign in with your updated credentials."; return; }
        render(data.user); message("userEditMsg", "Account saved."); await load(pageCursor);
      } catch (error) { if (generation === editorGeneration) message("userEditMsg", error.message, true); }
      finally { if (generation === editorGeneration) setBusy(false); }
    });
    $("userModerationForm").addEventListener("submit", async event => {
      event.preventDefault();
      if (!selected || busy) return;
      const action = event.submitter?.dataset.action;
      if (!["suspend", "ban", "activate"].includes(action)) return;
      const generation = editorGeneration;
      try {
        if (dirty) throw new Error("Save your account edits or reload the account before changing access.");
        const reason = $("userModerationReason").value.trim();
        if (!reason || new TextEncoder().encode(reason).length > 1000) throw new Error("Enter a reason of 1-1000 UTF-8 bytes.");
        const payload = { revision: selected.revision, action, reason };
        if (action === "suspend" && $("userSuspendedUntil").value) {
          const until = new Date($("userSuspendedUntil").value);
          if (!Number.isFinite(until.getTime()) || until <= new Date()) throw new Error("Choose a future suspension end time.");
          payload.suspended_until = until.toISOString();
        }
        const verb = action === "activate" ? "Restore access for" : action === "ban" ? "Ban" : "Suspend";
        if (!window.confirm(`${verb} ${selected.username}?\nReason: ${reason}\n${payload.suspended_until ? "Until: " + time(payload.suspended_until) : "This remains in effect until an administrator changes it."}`)) return;
        setBusy(true); message("userModerationMsg", "Saving access change...");
        const data = await apiFetch("/api/admin/users/" + encodeURIComponent(selected.id) + "/moderation", { method: "POST", json: payload });
        if (generation !== editorGeneration) return;
        render(data.user); message("userModerationMsg", "Account access updated. Previous sessions have ended."); await load(pageCursor);
      } catch (error) { if (generation === editorGeneration) message("userModerationMsg", error.message, true); }
      finally { if (generation === editorGeneration) setBusy(false); }
    });
    return { load, clear };
  };
})();
