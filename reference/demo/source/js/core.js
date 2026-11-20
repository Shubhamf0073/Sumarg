/* Runtime: state, router, shells, overlays and shared UI helpers. */
const KEY = 'previa-school-fleet-v3';
let S = loadState();
const UI = {};
const PAGES = {};
const ACT = {};
let CUR = { role: '', page: '', sub: [], params: {} };
let TIMERS = [];
const app = document.getElementById('app');

function loadState() {
  try {
    const o = JSON.parse(localStorage.getItem(KEY) || 'null');
    if (o && o.v === seedState().v) {
      const base = seedState();
      for (const k in base) if (base[k] && typeof base[k] === 'object' && !Array.isArray(base[k])) base[k] = { ...base[k], ...(o[k] || {}) }; else if (k in o) base[k] = o[k];
      return base;
    }
  } catch (e) { /* fall through to a fresh seed */ }
  return seedState();
}
function save() { try { localStorage.setItem(KEY, JSON.stringify(S)); } catch (e) { /* storage full or blocked */ } }
function resetDemo() { localStorage.removeItem(KEY); S = seedState(); for (const k in UI) delete UI[k]; toast('Demo data has been reset', 'ok'); go('#/'); }
function ui(key, def = {}) { return UI[key] || (UI[key] = { ...def }); }

/* ---------- formatting ---------- */
const inr = n => '₹' + Math.round(n).toLocaleString('en-IN');
const num = n => Math.round(n).toLocaleString('en-IN');
const esc = x => String(x ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const nowTime = () => new Intl.DateTimeFormat('en-IN', { hour: '2-digit', minute: '2-digit', hour12: true }).format(new Date()).toUpperCase();
const TODAY = 'Thu, 26 Sep 2026';

/* ---------- router ---------- */
function page(key, def) { PAGES[key] = def; }
const HOME = { admin: 'dashboard', maint: 'dashboard', portal: 'trip', driver: 'home', parent: 'home' };
const DESKTOP = new Set(['admin', 'maint', 'portal']);
function parseHash() {
  const h = location.hash.replace(/^#\/?/, '');
  const [path, q] = h.split('?');
  const parts = path.split('/').filter(Boolean);
  return { role: parts[0] || '', page: parts[1] || '', sub: parts.slice(2), params: Object.fromEntries(new URLSearchParams(q || '')) };
}
function go(h) { if (location.hash === h) render(true); else location.hash = h; }
function refresh() { render(false); }
let refreshTimer;
function softRefresh() { clearTimeout(refreshTimer); refreshTimer = setTimeout(refresh, 140); }

function render(isNav) {
  TIMERS.forEach(t => clearInterval(t)); TIMERS = [];
  const prev = CUR;
  const r = parseHash();
  if (r.role && !HOME[r.role]) { location.hash = '#/'; return; }
  if (r.role && !r.page) r.page = HOME[r.role];
  CUR = r;
  const key = r.role ? `${r.role}/${r.page}` : 'landing';
  const pg = PAGES[key] || PAGES[`${r.role}/${HOME[r.role]}`] || PAGES.landing;
  const samePage = prev.role === r.role && prev.page === r.page && JSON.stringify(prev.sub) === JSON.stringify(r.sub);
  const keep = !isNav || samePage;
  const scroll = { win: window.scrollY, mc: (document.querySelector('.mc') || {}).scrollTop || 0, keep: [...document.querySelectorAll('[data-keep]')].map(el => [el.dataset.keep, el.scrollTop]) };
  const ae = document.activeElement, focusKey = ae && ae.dataset ? ae.dataset.bind : null, selStart = ae && ae.selectionStart;
  document.querySelectorAll('.dd').forEach(d => d.remove());
  let html;
  if (!r.role) html = pg.render(r);
  else if (DESKTOP.has(r.role)) html = desktopShell(r, pg);
  else html = phoneShell(r, pg);
  app.innerHTML = html;
  document.title = 'Previa School Fleet' + (pg.title ? ' | ' + (typeof pg.title === 'function' ? pg.title(r) : pg.title) : '');
  if (keep) {
    window.scrollTo(0, scroll.win);
    const mc = document.querySelector('.mc'); if (mc) mc.scrollTop = scroll.mc;
    scroll.keep.forEach(([k, t]) => { const el = document.querySelector(`[data-keep="${k}"]`); if (el) el.scrollTop = t; });
  } else { window.scrollTo(0, 0); }
  if (focusKey) { const el = document.querySelector(`[data-bind="${focusKey}"]`); if (el) { el.focus(); try { el.setSelectionRange(selStart, selStart); } catch (e) { /* not a text input */ } } }
  if (pg.mount) pg.mount(app, r);
  startAnims();
}
window.addEventListener('hashchange', () => { closeModal(); render(true); });

/* ---------- binding & events ---------- */
function setPath(path, val) {
  const [scope, rest] = path.split(':');
  const root = scope === 's' ? S : UI;
  const ks = rest.split('.');
  let o = root;
  for (let i = 0; i < ks.length - 1; i++) o = o[ks[i]] || (o[ks[i]] = {});
  o[ks[ks.length - 1]] = val;
  if (scope === 's') save();
}
function getPath(path) {
  const [scope, rest] = path.split(':');
  let o = scope === 's' ? S : UI;
  for (const k of rest.split('.')) { if (o == null) return undefined; o = o[k]; }
  return o;
}
document.addEventListener('click', e => {
  const g = e.target.closest('[data-go]');
  if (g) { e.preventDefault(); closeDD(); if (g.closest('.ov')) closeModal(); go(g.dataset.go); return; }
  const a = e.target.closest('[data-act]');
  if (a) {
    const fn = ACT[a.dataset.act];
    if (fn) { e.preventDefault(); e.stopPropagation(); if (a.dataset.act !== 'dd' && a.dataset.keepdd === undefined) closeDD(); fn(a.dataset, a, e); }
    return;
  }
  if (!e.target.closest('.dd')) closeDD();
});
document.addEventListener('input', e => {
  const t = e.target;
  if (t.dataset && t.dataset.bind) {
    setPath(t.dataset.bind, t.type === 'checkbox' ? t.checked : t.value);
    if (t.dataset.live !== undefined) {
      if (t.dataset.pg) setPath(t.dataset.pg, 1);
      softRefresh();
    }
  }
});
document.addEventListener('change', e => {
  const t = e.target;
  if (t.tagName === 'SELECT' && t.dataset.bind) {
    setPath(t.dataset.bind, t.value);
    if (t.dataset.pg) setPath(t.dataset.pg, 1);
    if (t.dataset.norefresh === undefined) refresh();
  }
});
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') { closeDD(); closeModal(); closeSheet(); }
  if (e.key === 'Enter' && e.target.dataset && e.target.dataset.enter) { const fn = ACT[e.target.dataset.enter]; if (fn) { e.preventDefault(); fn(e.target.dataset, e.target, e); } }
});

/* generic actions */
ACT.set = d => { let v = d.v; if (d.num !== undefined) v = Number(v); if (d.toggle !== undefined) v = getPath(d.k) === v ? (d.off ?? null) : v; setPath(d.k, v); if (d.k2) setPath(d.k2, d.v2 === undefined ? null : (d.num2 !== undefined ? Number(d.v2) : d.v2)); if (d.msg) toast(d.msg, d.tt || 'ok'); refresh(); };
ACT.toast = d => toast(d.msg, d.tt || 'ok');
ACT.mclose = () => closeModal();
ACT.reset = () => resetDemo();
ACT.home = () => go('#/');
ACT.fullscreen = () => { if (!document.fullscreenElement) document.documentElement.requestFullscreen?.(); else document.exitFullscreen?.(); };
ACT.sidebar = () => document.querySelector('.sb')?.classList.toggle('open');
ACT.dl = d => toast(`${d.name || 'File'} downloaded`, 'ok');
ACT.call = d => toast(`Calling ${d.who || 'contact'}${d.phone ? ' (' + d.phone + ')' : ''}...`, 'ok');
ACT.sheetclose = () => closeSheet();

/* ---------- overlays ---------- */
function toast(msg, type = 'ok') {
  const icn = type === 'err' ? 'xc' : type === 'warn' ? 'warn' : 'checkc';
  const screen = document.querySelector('.screen');
  if (screen) {
    screen.querySelectorAll('.mtoast').forEach(t => t.remove());
    const el = document.createElement('div'); el.className = 'mtoast ' + type; el.innerHTML = ic(icn, 18) + `<span>${esc(msg)}</span>`;
    el.querySelector('.ic').style.color = type === 'err' ? '#f87171' : type === 'warn' ? '#fbbf24' : '#4ade80';
    screen.appendChild(el); setTimeout(() => el.remove(), 2600); return;
  }
  let w = document.querySelector('.toasts'); if (!w) { w = document.createElement('div'); w.className = 'toasts'; document.body.appendChild(w); }
  const el = document.createElement('div'); el.className = 'toast ' + type; el.innerHTML = ic(icn, 18) + `<span>${esc(msg)}</span>`;
  w.appendChild(el); setTimeout(() => el.remove(), 2800);
}
function openModal(title, body, foot = '', cls = '') {
  closeModal();
  const ov = document.createElement('div'); ov.className = 'ov';
  ov.innerHTML = `<div class="modal ${cls}" role="dialog"><div class="modal-h">${title}<button class="dp-close" data-act="mclose" aria-label="Close">${ic('x', 18)}</button></div><div class="modal-b">${body}</div>${foot ? `<div class="modal-f">${foot}</div>` : ''}</div>`;
  ov.addEventListener('mousedown', e => { if (e.target === ov) closeModal(); });
  document.body.appendChild(ov);
  const f = ov.querySelector('input,select,textarea'); if (f) setTimeout(() => f.focus(), 30);
  return ov;
}
function closeModal() { document.querySelectorAll('.ov').forEach(o => o.remove()); }
function formVals(root) { const o = {}; root.querySelectorAll('[name]').forEach(el => { o[el.name] = el.type === 'checkbox' ? el.checked : el.value.trim(); }); return o; }
function openDD(anchor, html, w = 320, align = 'right') {
  closeDD();
  const r = anchor.getBoundingClientRect();
  const dd = document.createElement('div'); dd.className = 'dd'; dd.style.width = w + 'px'; dd.setAttribute('data-ddroot', '');
  dd.innerHTML = html; document.body.appendChild(dd);
  const left = align === 'right' ? Math.max(8, r.right - w) : Math.min(r.left, window.innerWidth - w - 8);
  dd.style.left = left + 'px';
  const h = dd.offsetHeight; const top = r.bottom + 6 + h > window.innerHeight && r.top - h - 6 > 0 ? r.top - h - 6 : r.bottom + 6;
  dd.style.top = top + 'px';
  return dd;
}
function closeDD() { document.querySelectorAll('.dd').forEach(d => d.remove()); }
function openSheet(html) {
  closeSheet();
  const scr = document.querySelector('.screen'); if (!scr) return;
  const ov = document.createElement('div'); ov.className = 'sheet-ov';
  ov.innerHTML = `<div class="sheet"><div class="grab"></div>${html}</div>`;
  ov.addEventListener('mousedown', e => { if (e.target === ov) closeSheet(); });
  scr.appendChild(ov);
}
function closeSheet() { document.querySelectorAll('.sheet-ov').forEach(o => o.remove()); }
function pushNote(o) {
  const scr = document.querySelector('.screen'); if (!scr) return;
  scr.querySelectorAll('.push').forEach(p => p.remove());
  const el = document.createElement('div'); el.className = 'push';
  el.innerHTML = `<span class="mic sm" style="background:${o.bg || '#f59e0b'};color:#fff">${ic(o.icon || 'bus', 18)}</span><div style="flex:1;min-width:0"><div class="row" style="font-size:11.5px;color:#475569"><span>School Transport</span><span class="sp"></span><span>now</span></div><div class="b fs14 mt4">${esc(o.title)}</div><div class="fs12" style="color:#334155;margin-top:2px">${esc(o.body)}</div></div>`;
  el.onclick = () => { el.remove(); if (o.go) go(o.go); };
  scr.appendChild(el); setTimeout(() => el.remove(), 4500);
}
ACT.dd = (d, el) => { const fn = DDS[d.dd]; if (fn) openDD(el, fn(d, el), Number(d.w || 320), d.align || 'right'); };
const DDS = {};

/* ---------- map animation ---------- */
let animRAF = null;
function startAnims() {
  cancelAnimationFrame(animRAF);
  const els = [...document.querySelectorAll('[data-anim]')];
  if (!els.length) return;
  const items = els.map(g => {
    const p = document.getElementById(g.dataset.anim); if (!p) return null;
    const m = /scale\(([\d.]+)\)/.exec(g.getAttribute('transform') || ''); return { g, p, L: p.getTotalLength(), dur: Number(g.dataset.dur || 30), off: Number(g.dataset.off || 0), s: m ? m[1] : 1 };
  }).filter(Boolean);
  const t0 = performance.now();
  const tick = t => {
    if (!items[0] || !document.body.contains(items[0].g)) return;
    for (const it of items) {
      const f = ((t - t0) / 1000 / it.dur + it.off) % 1, at = f * it.L;
      const a = it.p.getPointAtLength(at), b = it.p.getPointAtLength(Math.min(it.L, at + 2));
      const ang = Math.atan2(b.y - a.y, b.x - a.x) * 57.3;
      it.g.setAttribute('transform', `translate(${a.x.toFixed(1)},${a.y.toFixed(1)}) rotate(${ang.toFixed(1)}) scale(${it.s})`);
    }
    animRAF = requestAnimationFrame(tick);
  };
  animRAF = requestAnimationFrame(tick);
}
function every(ms, fn) { const t = setInterval(fn, ms); TIMERS.push(t); return t; }

/* ---------- small UI builders ---------- */
const BADGE = {
  green: ['On Road', 'Active', 'On Duty', 'Completed', 'Paid', 'Pass', 'Passed', 'Resolved', 'Scheduled', 'In Stock', 'On Time', 'Boarded', 'Verified', 'Compliant', 'Normal', 'Good', 'Valid', 'Present', 'Efficient', 'Running', 'On Trip', 'Assigned', 'Arrived', 'Enabled', 'Approved', 'Live', 'Income', 'OK', 'Ready for Inspection', 'All Clear', 'Primary'],
  orange: ['Delayed', 'Pending', 'Partially Paid', 'Due Soon', 'Idle', 'In Progress', 'Low Stock', 'Acknowledged', 'On Leave', 'Due Today', 'Deviation', 'Draft', 'Check', 'Secondary', 'Minor', 'Under Review', 'Near Miss'],
  red: ['Overdue', 'Absent', 'Failed', 'Fail', 'Open', 'Out of Stock', 'Critical', 'No Show', 'Not Boarded', 'SOS Alert', 'Out of Service', 'In Service', 'Expense', 'Rejected', 'Breakdown', 'Major', 'Inactive', 'Delayed Bus', 'Medical', 'Harassment', 'Accident', 'Minor Injury', 'Missed'],
  gray: ['In Maintenance', 'Off Duty', 'Not Assigned', 'Closed', 'Disabled', 'Not Checked', 'Not Started', 'Cancelled'],
  blue: ['Info', 'Upcoming', 'Sent'],
};
function bcls(s) { for (const k in BADGE) if (BADGE[k].includes(s)) return k; return 'gray'; }
function badge(s, dot = true, cls) { return `<span class="bd bd-${cls || bcls(s)} ${dot ? 'dot' : ''}">${esc(s)}</span>`; }
function rc(code, small) { return `<span class="rc" style="background:${DATA.rcol(code)};${code === 'R-10' ? 'color:#1f2937' : ''}${small ? ';min-width:34px;height:22px;font-size:11px' : ''}">${code}</span>`; }
function kpi(o) {
  return `<div class="kpi k-${o.c} ${o.go || o.act ? 'clk' : ''}" ${o.go ? `data-go="${o.go}"` : ''} ${o.act ? `data-act="${o.act}" ${o.data || ''}` : ''}><span class="kpi-ic">${o.iconHtml || ic(o.i, 24)}</span><div style="min-width:0"><div class="kpi-v">${o.v}</div><div class="kpi-l">${o.l}</div></div><div class="kpi-f">${o.f || ''}</div></div>`;
}
const fsep = '<span class="sep"></span>';
function trend(up, txt, good, suffix = 'vs last month') { const c = good ? 'c-green' : 'c-red'; return `<span class="${c} b row" style="gap:3px">${ic(up ? 'arrowu' : 'arrowd', 14)}${txt}</span><span class="mut">${suffix}</span>`; }
function sel(bind, opts, val, attrs = '') { return `<div class="sel"><select data-bind="${bind}" ${attrs}>${opts.map(o => { const [v, t] = Array.isArray(o) ? o : [o, o]; return `<option value="${esc(v)}" ${String(val) === String(v) ? 'selected' : ''}>${esc(t)}</option>`; }).join('')}</select>${ic('down', 15)}</div>`; }
function searchBox(bind, ph, val, pgKey, extra = '') { return `<label class="search" ${extra}>${ic('search', 16)}<input class="inp" placeholder="${esc(ph)}" data-bind="${bind}" data-live ${pgKey ? `data-pg="${pgKey}"` : ''} value="${esc(val || '')}"/></label>`; }
function paginate(list, pg, per) { const pages = Math.max(1, Math.ceil(list.length / per)); const p = Math.min(Math.max(1, pg || 1), pages); return { rows: list.slice((p - 1) * per, p * per), p, pages, from: list.length ? (p - 1) * per + 1 : 0, to: Math.min(p * per, list.length) }; }
function pager(key, pg, total, noun, perKey, per = 10, perOpts = [10, 25, 50]) {
  const btn = (n, t = n, dis) => `<button ${dis ? 'disabled' : ''} class="${n === pg.p && t === n ? 'on' : ''}" data-act="set" data-k="${key}" data-v="${n}" data-num>${t}</button>`;
  let nums = [];
  if (pg.pages <= 7) nums = Array.from({ length: pg.pages }, (_, i) => i + 1);
  else { const s = Math.min(Math.max(1, pg.p - 2), pg.pages - 5); nums = [s, s + 1, s + 2, s + 3, s + 4]; }
  let mid = nums.map(n => btn(n)).join('');
  if (pg.pages > 7) { if (nums[nums.length - 1] < pg.pages - 1) mid += `<button disabled>...</button>`; if (nums[nums.length - 1] < pg.pages) mid += btn(pg.pages); }
  return `<div class="pager"><span>Showing ${pg.from} to ${pg.to} of ${num(total)} ${noun}</span><div class="pg">${btn(Math.max(1, pg.p - 1), ic('left', 14), pg.p === 1)}${mid}${btn(Math.min(pg.pages, pg.p + 1), ic('right', 14), pg.p === pg.pages)}</div>${perKey ? sel(perKey, perOpts.map(n => [n, n + ' per page']), per, `data-pg="${key}"`) : ''}</div>`;
}
function chk(on, act = '', data = '') { return `<span class="chk ${on ? 'on' : ''}" ${act ? `data-act="${act}" ${data}` : ''}>${on ? ic('check', 12, 'stroke-width="3"') : ''}</span>`; }
function legendRow(c, label, v, p) { return `<div class="lg-r"><span class="dotc" style="background:${c}"></span><span>${label}</span><span class="v">${v ?? ''}</span>${p !== undefined ? `<span class="p">${p}</span>` : ''}</div>`; }
function cardH(t, right = '') { return `<div class="card-h"><div class="card-t">${t}</div>${right}</div>`; }
function viewAll(go, t = 'View All') { return `<span class="link" style="margin-left:auto" ${go.startsWith('#') ? `data-go="${go}"` : `data-act="${go}"`}>${t}</span>`; }
function emptyRow(cols, msg = 'No matching records') { return `<tr><td colspan="${cols}" class="tc mut" style="padding:28px">${ic('search', 18)} ${msg}</td></tr>`; }
function field(label, inner, req) { return `<div style="min-width:0"><label class="fl">${label}${req ? ' <span class="req">*</span>' : ''}</label>${inner}</div>`; }
function audit(action, detail) { S.admin.audit.unshift({ t: nowTime(), d: '26 Sep 2026', user: CUR.role === 'maint' ? 'Maintenance Manager' : 'Super Admin', action, detail }); save(); }

/* ---------- desktop shell ---------- */
const NAV = {
  admin: [
    { k: 'dashboard', t: 'Dashboard', i: 'home' }, { k: 'fleet', t: 'Fleet', i: 'bus' }, { k: 'routes', t: 'Routes', i: 'routes' }, { k: 'students', t: 'Students', i: 'users' },
    { k: 'people', t: 'People', i: 'people', sub: ['Drivers', 'Maintenance Managers', 'Parents', 'Students', 'Administrators'] },
    { k: 'operations', t: 'Operations', i: 'ops', sub: ['Live Dispatch', 'Active Trips', 'Trip Assignments', 'Driver Reassignment', 'Route Changes', 'Delays & Alerts', 'Geofence & Alerts', 'Parent Notifications', 'Emergency Response'] },
    { k: 'maintenance', t: 'Maintenance', i: 'tools', sub: ['Overview', 'Preventive Maintenance', 'Service Requests', 'Repair & Work Orders', 'Inspections', 'Spare Parts & Inventory', 'Vendors & Garages', 'Documents', 'Maintenance Reports'] },
    { k: 'finance', t: 'Finance', i: 'finance', sub: ['Overview', 'Transport Fees', 'Payments', 'Expenses', 'Fuel Costs', 'Maintenance Costs', 'Vendor Payments', 'Outstanding Payments', 'Invoices & Receipts', 'Financial Reports'] },
    { k: 'safety', t: 'Safety', i: 'shield', sub: ['Overview', 'Incidents', 'SOS', 'Student Safety', 'Boarding Verification', 'Driver Compliance', 'Vehicle Compliance', 'Safety Inspections', 'Emergency Records'] },
    { k: 'sustainability', t: 'Sustainability', i: 'leaf', sub: ['Overview', 'Fuel', 'CO₂ Emissions', 'Scope 1', 'Scope 2', 'Scope 3', 'Vehicle Emissions', 'Route Emissions', 'Student Emissions', 'Reports'] },
    { k: 'reports', t: 'Reports', i: 'file', sub: ['Fleet Reports', 'Trip Reports', 'Student Reports', 'Finance Reports', 'Safety Reports', 'Scheduled Reports'] },
    { k: 'audit', t: 'Audit Trail', i: 'clip' },
  ],
  maint: [
    { k: 'dashboard', t: 'Dashboard', i: 'home' }, { k: 'vehicles', t: 'Vehicle Maintenance', i: 'tools' }, { k: 'workorders', t: 'Work Orders', i: 'wrench' },
    { k: 'schedule', t: 'Service Schedule', i: 'calcheck' }, { k: 'parts', t: 'Parts Inventory', i: 'box' }, { k: 'inspections', t: 'Inspections', i: 'shieldcheck' },
    { k: 'reports', t: 'Reports', i: 'barsc' }, { k: 'documents', t: 'Documents', i: 'file' },
  ],
  portal: [
    { k: 'dash', t: 'Dashboard', i: 'home' }, { k: 'trip', t: 'My Trip', i: 'cal' }, { k: 'route', t: 'Route Details', i: 'gear' }, { k: 'students', t: 'My Students', i: 'users' },
    { k: 'attendance', t: 'Attendance', i: 'clip' }, { k: 'inspection', t: 'Inspection', i: 'clipcheck' }, { k: 'history', t: 'History', i: 'history' },
    { sep: true }, { k: 'messages', t: 'Messages', i: 'msg', badge: 3 }, { k: 'help', t: 'Help & Support', i: 'help' },
  ],
};
const USER = {
  admin: { av: 'SA', n: 'Super Admin', r: 'Full Administrator', brand: 'Super Admin' },
  maint: { av: 'MM', n: 'Maintenance Manager', r: '', brand: 'Maintenance Manager' },
  portal: { av: 'RP', n: 'Ramesh Patel', r: 'Driver', brand: 'Driver Portal' },
};
const SEARCH_PH = { dashboard: 'Search vehicles, students, routes, drivers...', fleet: 'Search vehicles, drivers, routes...', routes: 'Search routes, vehicles, stops, students...', students: 'Search students, parents, routes, stops...', people: 'Search people, drivers, students, parents...', operations: 'Search vehicles, drivers, routes, students, stops...', finance: 'Search students, parents, routes, vehicles, invoices...', safety: 'Search students, drivers, vehicles, incidents...', sustainability: 'Search vehicles, routes, students, reports...', maintenance: 'Search vehicles, work orders, parts, vendors...' };
const slug = s => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
function desktopShell(r, pg) {
  const role = r.role, u = USER[role];
  const navKey = pg.nav || r.page;
  const items = NAV[role].map(it => {
    if (it.sep) return '<div style="height:1px;background:rgba(255,255,255,.08);margin:10px 8px"></div>';
    const on = navKey === it.k;
    let h = `<button class="sb-item ${on ? 'active' : ''}" data-go="#/${role}/${it.k}">${ic(it.i, 20)}<span>${it.t}</span>${it.badge ? `<span class="sb-badge">${it.badge}</span>` : ''}${it.sub ? `<span class="chev">${ic(on ? 'down' : 'right', 16)}</span>` : ''}</button>`;
    if (it.sub && on) {
      const cur = r.params.s || slug(it.sub[0]);
      h += `<div class="sb-sub">${it.sub.map(s => `<button class="sb-subitem ${cur === slug(s) ? 'active' : ''}" data-go="#/${role}/${it.k}?s=${slug(s)}">${ic('right', 13)}<span>${s}</span></button>`).join('')}</div>`;
    }
    return h;
  }).join('');
  const foot = role === 'portal' ? `<button class="sb-item" data-act="logout">${ic('logout', 20)}<span>Logout</span></button>` : `<button class="sb-item ${r.page === 'settings' ? 'active' : ''}" data-go="#/${role}/settings">${ic('settings', 20)}<span>Settings</span></button>`;
  const nb = role === 'portal' ? 2 : 3;
  const ph = role === 'portal' ? 'Search routes, students, stop, ...' : (SEARCH_PH[navKey] || 'Search vehicles, work orders, parts, vendors...');
  return `<div class="dk"><aside class="sb"><div class="sb-brand"><span class="sb-logo" style="${role === 'maint' ? 'background:#1463ff;color:#fff' : ''}">${ic(role === 'maint' ? 'tools' : 'busf', role === 'maint' ? 24 : 34, role === 'maint' ? '' : 'stroke-width="2.2"')}</span><div><div class="t1">School Transport</div><div class="t2">${u.brand}</div></div></div><nav class="sb-nav">${items}</nav><div class="sb-foot">${foot}</div></aside>
  <div class="dk-main"><header class="tb"><button class="tb-ic menu-btn" data-act="sidebar">${ic('menu', 20)}</button>${`<label class="search" data-act="gsearch">${ic('search', 16)}<input class="inp" placeholder="${ph}" data-bind="u:g.q" data-enter="gsearch" value="${esc((UI.g || {}).q || '')}" readonly/></label>`}<span class="sp"></span>
  <button class="tb-ic" data-act="dd" data-dd="notifs" data-w="360" title="Notifications">${ic('bell', 21)}${S.notifRead[role] ? '' : `<span class="n">${nb}</span>`}</button>
  <button class="tb-ic" data-act="dd" data-dd="mails" data-w="340" title="Messages">${ic('mail', 21)}<span class="n">5</span></button>
  <button class="tb-ic" data-act="fullscreen" title="Full screen">${ic('fullscreen', 19)}</button><span class="tb-sep"></span>
  <button class="tb-user" data-act="dd" data-dd="user" data-w="240"><span class="av">${u.av}</span><span style="text-align:left"><div class="n1">${u.n}</div>${u.r ? `<div class="n2">${u.r}</div>` : ''}</span>${ic('down', 16)}</button></header>
  <main class="content">${pg.render(r)}</main></div></div>`;
}

/* top bar dropdowns */
const NOTIFS = {
  admin: [['warn', '#ef4444', 'Bus GJ01AB1234 delayed by 45 minutes', '08:12 AM', '#/admin/operations'], ['tools', '#f59e0b', 'Maintenance due for Bus GJ01CD5678', '07:45 AM', '#/admin/maintenance'], ['shieldalert', '#ef4444', 'SOS alert from Diya Patel (R-05)', '07:32 AM', '#/admin/safety']],
  maint: [['warn', '#ef4444', 'GJ01CD5678 tire inspection overdue by 16 days', '09:10 AM', '#/maint/schedule'], ['box', '#f59e0b', '27 parts are below reorder level', '08:30 AM', '#/maint/parts'], ['wrench', '#2563eb', 'WO-2026-014 waiting for approval', '08:05 AM', '#/maint/workorders']],
  portal: [['route', '#2563eb', 'Route R-01 updated for tomorrow', '10:24 AM', '#/portal/route'], ['checkc', '#16a34a', 'Attendance marked successfully', '06:45 AM', '#/portal/attendance']],
};
DDS.notifs = () => {
  const role = CUR.role, list = NOTIFS[role];
  return `<div class="dd-h">Notifications<span class="sp"></span><span class="link" data-act="readall">Mark all as read</span></div><div class="dd-sep"></div>${list.map(n => `<button class="dd-i" data-go="${n[4]}"><span class="mic sm" style="background:${n[1]}1a;color:${n[1]}">${ic(n[0], 17)}</span><span style="flex:1"><div class="sb6 fs13">${n[2]}</div><div class="fs11 mut mt4">${n[3]} | Today</div></span></button>`).join('')}`;
};
ACT.readall = () => { S.notifRead[CUR.role] = true; save(); closeDD(); refresh(); toast('All notifications marked as read'); };
const MAILS = [['Transport Coordinator', 'R-03 route change approved for tomorrow.', '10:24 AM'], ['City Plus School', 'Please share the revised drop timings.', '09:40 AM'], ['Ramesh Patel', 'Bus GJ01AB1234 refuelled, 28 L.', '09:12 AM'], ['Finance Team', 'September vendor payments processed.', 'Yesterday'], ['Parent Desk', '3 new pickup change requests.', 'Yesterday']];
DDS.mails = () => `<div class="dd-h">Messages<span class="sp"></span><span class="bd bd-blue">5 new</span></div><div class="dd-sep"></div>${MAILS.map(m => `<button class="dd-i" data-act="openmail" data-who="${esc(m[0])}" data-msg="${esc(m[1])}">${av(m[0], 34)}<span style="flex:1;min-width:0"><div class="row"><span class="sb6 fs13">${m[0]}</span><span class="sp"></span><span class="fs11 mut">${m[2]}</span></div><div class="fs12 mut ellip mt4">${m[1]}</div></span></button>`).join('')}`;
ACT.openmail = d => openModal(esc(d.who), `<div class="row mt12" style="align-items:flex-start">${av(d.who, 40)}<div class="bubble in" style="max-width:100%">${esc(d.msg)}</div></div><label class="fl">Reply</label><textarea class="inp" name="reply" placeholder="Type your reply..."></textarea>`, `<button class="btn btn-outline" data-act="mclose">Cancel</button><button class="btn btn-primary" data-act="sendmail">${ic('send', 16)}Send</button>`);
ACT.sendmail = () => { closeModal(); toast('Reply sent'); };
DDS.user = () => {
  const u = USER[CUR.role];
  return `<div class="dd-h">${u.n}</div><div class="fs12 mut" style="padding:0 10px 8px">${u.r || 'Previa School Fleet'}</div><div class="dd-sep"></div>
  ${CUR.role !== 'portal' ? `<button class="dd-i" data-go="#/${CUR.role}/settings">${ic('settings', 17)}Settings</button>` : `<button class="dd-i" data-go="#/driver/home">${ic('phonem', 17)}Open Driver Mobile App</button>`}
  <button class="dd-i" data-go="#/">${ic('swap', 17)}Switch Role</button><button class="dd-i" data-act="reset">${ic('refresh', 17)}Reset Demo Data</button><div class="dd-sep"></div><button class="dd-i" data-act="logout" style="color:#dc2626">${ic('logout', 17)}Logout</button>`;
};
ACT.logout = () => { toast('Signed out'); go('#/'); };

/* global search */
ACT.gsearch = (d, el) => {
  const q = ((UI.g || {}).q || '');
  openModal('Search', `<label class="search mt12">${ic('search', 16)}<input class="inp" id="gsq" placeholder="Search vehicles, students, routes, drivers..." value="${esc(q)}"/></label><div id="gsr" class="mt12" style="max-height:420px;overflow:auto"></div>`, '', '');
  const inp = document.getElementById('gsq'), out = document.getElementById('gsr');
  const base = CUR.role === 'portal' ? 'admin' : CUR.role === 'maint' ? 'maint' : 'admin';
  const run = () => {
    const t = inp.value.trim().toLowerCase(); setPath('u:g.q', inp.value);
    if (!t) { out.innerHTML = '<div class="mut fs12" style="padding:10px">Try "GJ01AB1234", "Aarav", "R-03" or "Ramesh".</div>'; return; }
    const res = [];
    allVehicles().filter(v => (v.no + ' ' + v.model + ' ' + (v.driver || '')).toLowerCase().includes(t)).slice(0, 5).forEach(v => res.push(['bus', v.no, v.model + ' | ' + v.status, base === 'maint' ? `#/maint/vehicles?v=${v.no}` : `#/admin/fleet?v=${v.no}`]));
    allStudents().filter(s => (s.name + ' ' + s.adm).toLowerCase().includes(t)).slice(0, 5).forEach(s => res.push(['users', s.name, s.adm + ' | ' + s.cls + ' | ' + s.route, `#/admin/students?id=${s.adm}`]));
    allRoutes().filter(r => (r.code + ' ' + r.name + ' ' + r.area).toLowerCase().includes(t)).slice(0, 5).forEach(r => res.push(['routes', r.code + ' - ' + r.name, r.area, `#/admin/routes?r=${r.code}`]));
    DATA.drivers.filter(x => (x.name + ' ' + x.id).toLowerCase().includes(t)).slice(0, 5).forEach(x => res.push(['user', x.name, x.id + ' | ' + x.status, `#/admin/people?s=drivers&d=${x.id}`]));
    out.innerHTML = res.length ? res.map(x => `<button class="dd-i" data-go="${x[3]}">${tile(x[0], '#1463ff', '#eaf1ff', 34, 17, 9)}<span><div class="sb6">${esc(x[1])}</div><div class="fs12 mut">${esc(x[2])}</div></span></button>`).join('') : '<div class="mut" style="padding:14px">No results found.</div>';
  };
  inp.addEventListener('input', run); run(); inp.focus();
};

/* data accessors that include records added during the demo */
const allVehicles = () => [...S.added.vehicles, ...DATA.vehicles];
const allStudents = () => [...S.added.students, ...DATA.students];
const allRoutes = () => [...S.added.routes, ...DATA.routes];
const allDrivers = () => [...S.added.drivers, ...DATA.drivers];

/* ---------- phone shell ---------- */
function phoneShell(r, pg) {
  const role = r.role;
  const head = pg.header ? pg.header(r) : mhead(typeof pg.title === 'function' ? pg.title(r) : pg.title, pg.back, pg.right ? pg.right(r) : '', pg.center);
  const nav = pg.nav ? bottomNav(role, pg.nav, r) : '';
  const side = role === 'driver'
    ? `<h4>Driver App</h4><p>Imran Shaikh | R-03 East Zone. Run the full trip: photo check-in, safety inspection, boarding by stop, incidents and trip completion.</p><div class="mt12">${[['Driver Dashboard', '#/driver/home'], ['Trip Details', '#/driver/trip'], ['Safety Inspection', '#/driver/inspect'], ['Student Boarding', '#/driver/stops'], ['Report Incident', '#/driver/incidents'], ['Messages', '#/driver/messages'], ['Trip History', '#/driver/history']].map(x => `<button class="btn btn-outline" data-go="${x[1]}">${ic('right', 14)}${x[0]}</button>`).join('')}<button class="btn btn-soft" data-go="#/portal/trip">${ic('monitor', 16)}Open Driver Web Portal</button></div>`
    : `<h4>Parent App</h4><p>Priya Sharma | Parent of Aarav and Ananya. Track the bus live, see boarding photos, confirm safe arrival home and raise requests.</p><div class="mt12">${[['Home', '#/parent/home'], ['My Children', '#/parent/children'], ['Live Tracking', '#/parent/tracking'], ['Boarding Status', '#/parent/boarding'], ['Notifications', '#/parent/notifications'], ['Pickup / Drop Requests', '#/parent/requests'], ['Emergency', '#/parent/emergency'], ['Messages', '#/parent/messages']].map(x => `<button class="btn btn-outline" data-go="${x[1]}">${ic('right', 14)}${x[0]}</button>`).join('')}</div>`;
  return `<div class="stage"><div class="stage-side">${side}<div class="mt16"><button class="btn btn-outline" data-go="#/">${ic('swap', 16)}Switch Role</button><button class="btn btn-outline" data-act="reset">${ic('refresh', 16)}Reset Demo Data</button></div></div>
  <div class="phone"><div class="screen"><div class="sbar"><span>9:41</span>${statusIcons()}</div>${head}<div class="mc ${pg.pad === false ? '' : 'pad'}">${pg.render(r)}</div>${nav}</div></div></div>`;
}
function mhead(title, back, right = '', center = false) {
  const b = back ? `<button class="mh-b" ${back === true ? 'data-act="back"' : `data-go="${back}"`} aria-label="Back">${ic('arrowl', 22)}</button>` : '';
  return `<div class="mh ${center ? 'center' : ''}">${b}<div class="mh-t">${title}</div>${right}</div>`;
}
ACT.back = () => history.back();
function mhb(icon, attrs, n) { return `<button class="mh-b" ${attrs}>${ic(icon, 21)}${n ? `<span class="n">${n}</span>` : ''}</button>`; }
const BNAV = {
  driver: [['home', 'Home', 'home', '#/driver/home'], ['trip', 'My Trip', 'msglines', '#/driver/trip'], ['incidents', 'Incidents', 'warn', '#/driver/incidents'], ['messages', 'Messages', 'msg', '#/driver/messages'], ['more', 'More', 'grid9', '#/driver/profile']],
  parentA: [['home', 'Home', 'home', '#/parent/home'], ['children', 'My Children', 'users', '#/parent/children'], ['tracking', 'Tracking', 'pin', '#/parent/tracking'], ['notifications', 'Notifications', 'bell', '#/parent/notifications'], ['more', 'More', 'grid9', 'more']],
  parentB: [['home', 'Home', 'home', '#/parent/home'], ['tracking', 'Tracking', 'pin', '#/parent/tracking'], ['notifications', 'Notifications', 'bell', '#/parent/notifications'], ['messages', 'Messages', 'mail', '#/parent/messages'], ['more', 'More', 'grid9', 'more']],
};
function bottomNav(role, spec, r) {
  const [set, onKey] = spec.split(':');
  const unread = role === 'parent' ? PARENT_NOTIFS.filter((n, i) => !S.parent.notifRead.includes(i)).length : 0;
  return `<nav class="bn">${BNAV[set].map(([k, t, i, to]) => `<button class="bn-i ${k === onKey ? 'on' : ''}" ${to === 'more' ? 'data-act="pmore"' : `data-go="${to}"`}>${ic(i, 22)}<span>${t}</span>${k === 'notifications' && unread ? `<span class="n">${unread}</span>` : ''}${k === 'messages' && role === 'parent' ? '<span class="n">3</span>' : ''}</button>`).join('')}</nav>`;
}
const PARENT_NOTIFS = [];

/* start */
window.addEventListener('DOMContentLoaded', () => render(true));
