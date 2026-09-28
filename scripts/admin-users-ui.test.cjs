'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');

// Small DOM test double for component events; no browser or runtime dependency.
class Element {
  constructor() { this.value = ''; this.checked = false; this.disabled = false; this.children = []; this.style = {}; this.events = {}; this.dataset = {}; this.textContent = ''; this.classList = { toggle() {} }; }
  get value() { return this._value; }
  set value(value) { this._value = String(value); }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) { this.children = children; }
  setAttribute(key, value) { this[key] = value; }
  addEventListener(name, fn) { (this.events[name] ||= []).push(fn); }
  async fire(name, extra = {}) { for (const fn of this.events[name] || []) await fn({ preventDefault() {}, ...extra }); }
  focus() {}
  reset() {}
}
function fixture(options = {}) {
  const elements = new Map();
  const element = id => { if (!elements.has(id)) elements.set(id, new Element()); return elements.get(id); };
  const document = { getElementById: element, createElement: () => new Element() };
  const user = { id: 'pilot', username: '<pilot>', is_admin: false, must_change_password: false, revision: 3,
    effective_status: 'active', status: 'active', created_at: '2026-09-01T00:00:00Z',
    player: { id: 'player', credits: '8999999999999999', xp: '0', level: 1, ship_type: 'SCOUT',
      ship_cargo_upgrades: 0, ship_turn_upgrades: 0, turns: 50, turns_max: 100, sector_id: 1,
      cargo_max: 30, cargo_ore: 0, cargo_organics: 0, cargo_equipment: 0, season_id: 1 } };
  const calls = []; let logout = false;
  const apiFetch = async (path, opts = {}) => {
    calls.push({ path, ...opts });
    if (options.apiFetch) return options.apiFetch(path, opts);
    if (opts.method) { user.revision++; if (opts.json.player) Object.assign(user.player, opts.json.player); return { user: structuredClone(user), sign_in_again: options.signInAgain }; }
    if (path.startsWith('/api/admin/users?')) return { users: [structuredClone(user)], next_cursor: '' };
    return { user: structuredClone(user) };
  };
  const window = { confirm: () => options.confirm !== false };
  vm.runInNewContext(fs.readFileSync('web/static/admin-users.js', 'utf8'), { window, document, URLSearchParams, TextEncoder, Date, BigInt });
  const manager = window.createAdminUserManager({ apiFetch, logout: () => { logout = true; manager.clear(); }, currentUser: () => ({ user_id: options.self ? 'pilot' : 'admin' }) });
  const open = async () => { await manager.load(); await element('userList').children[0].children[3].children[0].fire('click'); await new Promise(resolve => setImmediate(resolve)); };
  return { element, user, calls, manager, open, loggedOut: () => logout };
}

test('loads untrusted usernames as text and preserves large balances exactly', async () => {
  const f = fixture(); await f.open();
  assert.equal(f.element('userList').children[0].children[0].textContent, '<pilot>');
  assert.equal(f.element('editPlayer_credits').value, '8999999999999999');
  assert.equal(f.element('editPassword').value, '');
});
test('only changed player fields are sent and zero balances are supported', async () => {
  const f = fixture(); await f.open(); f.element('editPlayer_credits').value = '0';
  await f.element('userEditForm').fire('submit');
  const request = f.calls.find(call => call.method === 'PATCH');
  assert.equal(JSON.stringify(request.json), JSON.stringify({ revision: 3, player: { credits: '0' } }));
  assert.equal(f.element('userEditMsg').textContent, 'Account saved.');
});
test('rejects invalid password bytes before any request', async () => {
  const f = fixture(); await f.open(); f.element('editPassword').value = 'é'.repeat(37); f.element('editPasswordConfirm').value = 'é'.repeat(37);
  await f.element('userEditForm').fire('submit');
  assert.match(f.element('userEditMsg').textContent, /8-72/);
  assert.equal(f.calls.some(call => call.method), false);
});
test('password reset defaults to required change and confirmation can cancel', async () => {
  const f = fixture({ confirm: false }); await f.open(); f.element('editPassword').value = 'Valid-Fixture-Password';
  await f.element('editPassword').fire('input'); assert.equal(f.element('editMustChange').checked, true);
  f.element('editPasswordConfirm').value = f.element('editPassword').value;
  await f.element('userEditForm').fire('submit'); assert.equal(f.calls.some(call => call.method), false);
});
test('moderation requires a reason and preserves the selected account ID', async () => {
  const f = fixture(); await f.open(); const submitter = { dataset: { action: 'ban' } };
  await f.element('userModerationForm').fire('submit', { submitter });
  assert.equal(f.calls.some(call => call.method), false);
  f.element('userModerationReason').value = 'Repeated abuse';
  await f.element('userModerationForm').fire('submit', { submitter });
  const call = f.calls.find(call => call.method === 'POST');
  assert.equal(call.path, '/api/admin/users/pilot/moderation');
  assert.equal(call.json.action, 'ban'); assert.equal(call.json.revision, 3);
});
test('unsaved edits prevent accidental moderation', async () => {
  const f = fixture(); await f.open(); await f.element('userEditForm').fire('input');
  f.element('userModerationReason').value = 'Reason';
  await f.element('userModerationForm').fire('submit', { submitter: { dataset: { action: 'suspend' } } });
  assert.match(f.element('userModerationMsg').textContent, /Save your account edits/);
  assert.equal(f.calls.some(call => call.method), false);
});
test('self restriction controls are disabled', async () => {
  const f = fixture({ self: true }); await f.open();
  for (const id of ['editIsAdmin', 'userSuspend', 'userBan']) assert.equal(f.element(id).disabled, true);
});
test('late account responses cannot repopulate data after logout', async () => {
  let resolve;
  const f = fixture({ apiFetch: () => new Promise(r => { resolve = r; }) });
  const pending = f.manager.load(); f.manager.clear(); resolve({ users: [f.user], next_cursor: '' }); await pending;
  assert.equal(f.element('userList').children.length, 0);
  assert.equal(f.element('userEditor').style.display, 'none');
});
test('editing own credentials ends the editor session', async () => {
  const f = fixture({ signInAgain: true }); await f.open(); f.element('editUsername').value = 'renamed';
  await f.element('userEditForm').fire('submit'); assert.equal(f.loggedOut(), true);
  assert.match(f.element('authMsg').textContent, /Sign in/);
});
