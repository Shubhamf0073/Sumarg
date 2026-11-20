/* Driver: mobile app (#/driver/...) and web portal (#/portal/...) */
const D = () => S.driver;
const dp = (k, def) => page('driver/' + k, def);
const dBell = () => mhb('bell', 'data-go="#/driver/messages"');
const INC_TYPES = {
  accident: { t: 'Report Accident', n: 'Accident', s: 'Vehicle accident or collision', ic: 'crash', c: '#ef4444', bg: '#fdeeee', lead: 'Provide details about the accident and your current location.', desc: 'Minor collision with another vehicle.\nNo major injuries. Vehicle is safe.', photos: ['boarding', 'brake', 'engine'] },
  breakdown: { t: 'Report Breakdown', n: 'Breakdown', s: 'Vehicle breakdown or mechanical issue', ic: 'tools', c: '#f97316', bg: '#fff3e6', lead: 'Report vehicle breakdown or mechanical issue.', desc: 'Engine overheated. Vehicle stopped on the side of the road.', photos: ['boarding', 'interior', 'engine'], issue: ['Engine Issue', 'Tyre Puncture', 'Battery', 'Brake Issue', 'Other'] },
  student: { t: 'Report Student Issue', n: 'Student Issue', s: 'Student behavior or safety issue', ic: 'users', c: '#2563eb', bg: '#eaf1ff', lead: 'Report any student related issue or safety concern.', desc: 'Student was not following instructions at the stop.', photos: ['interior'], issue: ['Behavior Issue', 'Injury', 'Unwell', 'Lost Item', 'Other'] },
  route: { t: 'Report Route Issue', n: 'Route Issue', s: 'Traffic, road closure, delay', ic: 'routes', c: '#16a34a', bg: '#ecf9f0', lead: 'Report traffic, road closure or delay related issues.', desc: 'Heavy traffic on NH-8.  Expected delay of 20 minutes.', photos: ['traffic'], issue: ['Traffic', 'Road Closure', 'Diversion', 'Weather', 'Other'] },
  emergency: { t: 'Emergency', n: 'Emergency', s: 'Medical emergency or SOS', ic: 'siren', c: '#ef4444', bg: '#fdeeee' },
};
function stopKids(i) {
  const st = DATA.drvStops[i]; if (!st || !st.exp) return [];
  if (i === 1) return DATA.boardKids;
  const off = [0, 0, 20, 30, 40][i] || 50;
  return DATA.students.slice(off, off + st.exp).map((s, k) => ({ id: `S${String(12 + off + k).padStart(3, '0')}`, name: s.name, cls: s.cls }));
}
const kidStatus = id => D().board[id] || 'Pending';
const stopCounts = i => { const ks = stopKids(i); const b = ks.filter(k => kidStatus(k.id) === 'Boarded').length, n = ks.filter(k => kidStatus(k.id) === 'No Show').length; return { total: ks.length, b, n, p: ks.length - b - n }; };
const curStop = () => DATA.drvStops[D().stop] ? D().stop : 1;
function dCard(inner, st = '') { return `<div class="mcard" style="${st}">${inner}</div>`; }
function tripHead() { return `<div class="mcard row" style="align-items:flex-start">${tile('busf', '#fff', '#8b5cf6', 42, 22, 11)}<div style="flex:1"><div class="b fs16">R-03 &nbsp;East Zone</div><div class="fs12 mut mt4">GJ01EF9012 | Tata Starbus</div></div>${D().phase === 'completed' ? badge('Completed', false) : badge('On Time', false)}</div>`; }

/* ---------- Dashboard ---------- */
dp('home', {
  title: 'Driver', nav: 'driver:home', pad: false,
  header: () => `<div class="mh" style="flex-direction:column;align-items:stretch;padding-bottom:22px"><div class="row">${ic('steering', 22)}<span class="mh-t" style="margin-left:6px">Driver</span>${dBell()}</div><div class="row" style="margin-top:10px">${av('Imran Shaikh', 50, 'mb')}<div><div class="b fs16">Imran Shaikh</div><div class="fs12" style="opacity:.85">Driver ID: DRV001</div></div></div></div>`,
  render: () => {
    const d = D(), ph = d.phase;
    const status = ph === 'completed' ? 'Completed' : ph === 'inprogress' ? 'In Progress' : 'Assigned';
    const cta = ph === 'completed' ? `<button class="mbtn g mt12" data-go="#/driver/history">${ic('checkc', 18)}Trip Completed - View History</button>` : ph === 'inprogress' ? `<button class="mbtn p mt12" data-go="#/driver/stops">Continue Boarding ${ic('arrowr', 18)}</button>` : `<button class="mbtn p mt12" data-go="#/driver/trip">View Trip Details ${ic('arrowr', 18)}</button>`;
    return `<div style="padding:12px"><div class="row fs13 sb6" style="padding:2px 4px 10px">${ic('cal', 17)}${TODAY}</div>
    ${dCard(`<div class="row"><span class="b fs18" style="flex:1">Today's Trip</span>${badge(status, false, status === 'In Progress' ? 'orange' : 'green')}</div>
      <div class="row mt12"><div style="flex:1"><div class="fs13 sb6">Bus</div><div class="b fs18 mt4">GJ01EF9012</div><div class="fs12 mut mt4">Tata Starbus (52 Seater)</div></div>${busSide(96)}</div>
      <div style="border-top:1px solid #eef1f6;margin-top:12px;padding-top:10px"><div class="fs13 sb6">Route</div><div class="b fs16 mt4">R-03 &nbsp;East Zone</div></div>
      <div class="row mt12" style="border-top:1px solid #eef1f6;padding-top:12px">${[['pin', '18', 'Stops'], ['users', '24', 'Students'], ['clock', '07:05 AM', 'Start Time']].map(x => `<div class="row" style="flex:1;gap:6px">${ic(x[0], 20, 'style="color:#1e3a8a"')}<div><div class="b fs15">${x[1]}</div><div class="fs11 mut">${x[2]}</div></div></div>`).join('')}</div>${cta}`)}
    ${ph === 'inprogress' ? dCard(`<div class="row">${ic('pin', 22, 'style="color:#2563eb"')}<div style="flex:1"><div class="fs12 mut">Current Stop</div><div class="b fs15">${DATA.drvStops[curStop()].name}</div></div><div class="tr"><div class="b fs15">${DATA.boardKids.filter(k => kidStatus(k.id) === 'Boarded').length + [2, 3, 4].reduce((a, i) => a + stopCounts(i).b, 0)}/24</div><div class="fs11 mut">Boarded</div></div></div>`) : ''}
    <div class="g mt12" style="grid-template-columns:repeat(4,1fr);gap:8px">${[['users', 'Student Boarding', '#/driver/stops', '#2563eb', '#eaf1ff'], ['clipcheck', 'Safety Inspection', '#/driver/inspect', '#16a34a', '#e8f8ee'], ['warn', 'Report Incident', '#/driver/incidents', '#ef4444', '#fdeeee'], ['msg', 'Messages', '#/driver/messages', '#2563eb', '#eaf1ff']].map(x => `<button class="mcard" style="margin:0;display:flex;flex-direction:column;align-items:center;gap:6px;padding:10px 2px" data-go="${x[2]}">${tile(x[0], x[3], x[4], 40, 22, 10)}<span class="fs11 sb6 tc">${x[1]}</span></button>`).join('')}</div>
    <div class="mbanner mt12" style="background:#fff;border:1px solid #e9eef5">${ic('info', 20, 'style="color:#2563eb"')}<div class="s" style="margin:0">Trip flow: Trip Details, Driver Photo, Safety Inspection, Student Boarding, Complete Trip.</div></div></div>`;
  },
});

/* ---------- Trip details ---------- */
dp('trip', {
  title: 'My Details', back: '#/driver/home', right: dBell, nav: 'driver:trip',
  render: () => {
    const t = UI.dtt || 'Stops', ph = D().phase;
    const tabs = `<div class="mtabs" style="margin:12px -12px 0">${[['Route', 'Route'], ['Stops', 'Stops (18)'], ['Students', 'Students (24)']].map(x => `<button class="mtab ${t === x[0] ? 'on' : ''}" data-act="set" data-k="u:dtt" data-v="${x[0]}">${x[1]}</button>`).join('')}</div>`;
    let body;
    if (t === 'Route') body = `<div class="map mt12" style="height:300px">${drvMap(300)}</div><div class="mcard mt12">${infoRow('route', '#2563eb', 'Distance', '28.5 km')}${infoRow('clock', '#16a34a', 'Duration', '55 mins')}${infoRow('school', '#8b5cf6', 'Destination', 'City Plus School')}</div>`;
    else if (t === 'Students') body = `<div class="mcard mt12" style="padding:2px 10px">${[1, 2, 3, 4].flatMap(i => stopKids(i).slice(0, i === 1 ? 12 : 4).map(k => ({ ...k, stop: DATA.drvStops[i].name }))).slice(0, 24).map(k => `<div class="mrow" style="padding:8px 0">${av(k.name, 36)}<div style="flex:1"><div class="t fs13">${k.name}</div><div class="s">${k.id} | ${k.cls} | ${k.stop}</div></div>${badge(kidStatus(k.id), false)}</div>`).join('')}</div>`;
    else body = `<div class="mcard mt12" style="padding:6px 10px">${DATA.drvStops.map((s, i) => { const c = stopCounts(i), done = ph === 'completed' || (ph === 'inprogress' && (D().stopsDone.includes(i) || i === 0)); const cur = i === curStop() && ph === 'inprogress'; const tot = i === 5 ? 24 : s.exp; return `<div class="row" style="padding:9px 6px;${i === 1 && !ph.startsWith('a') ? '' : ''}${cur ? 'background:#eef4ff;border-radius:10px;border:1px solid #d6e4ff' : ''}"><span class="tl-dot ${done ? '' : ''}" style="width:16px;height:16px;border-color:${i === 0 || i === 5 ? '#16a34a' : i % 2 ? '#8b5cf6' : '#16a34a'};${cur ? 'background:#2563eb;border-color:#2563eb' : ''}"></span><span class="fs13 mut" style="width:68px">${s.time}</span><span class="b fs14" style="flex:1">${s.name}</span><span class="fs13 ${done ? 'c-green b' : 'mut'}" style="width:48px">${i === 0 ? '0/24' : i === 5 ? (ph === 'completed' ? '24/24' : '0/24') : (ph === 'assigned' ? tot + '/' + tot : c.b + '/' + c.total)}</span>${ph === 'assigned' ? (i > 0 ? `<span class="ck">${ic('check', 13, 'stroke-width="3"')}</span>` : '<span style="width:22px"></span>') : done ? `<span class="ck">${ic('check', 13, 'stroke-width="3"')}</span>` : '<span class="ck off"></span>'}</div>`; }).join('')}</div>`;
    return `${dCard(`<div class="row">${tile('busf', '#fff', '#8b5cf6', 44, 22, 11)}<div style="flex:1"><div class="b fs16">R-03 &nbsp;East Zone</div><div class="fs12 mut mt4">GJ01EF9012 | Tata Starbus</div></div>${badge('On Time', false)}</div><div class="row fs13 sb6 mt12">${ic('cal', 17)}${TODAY}</div><div class="row mt12" style="background:#f7f9fc;border-radius:10px;padding:10px"><div class="row" style="flex:1">${ic('cal', 20)}<div><div class="fs11 mut">Start Time</div><div class="b fs16">07:05 AM</div></div></div><div class="row" style="flex:1;border-left:1px solid #e2e8f0;padding-left:12px">${ic('clock', 20)}<div><div class="fs11 mut">End Time</div><div class="b fs16">08:00 AM</div></div></div></div>`)}${tabs}${body}
    ${ph === 'assigned' ? `<button class="mbtn p mt12" data-go="#/driver/start">${ic('playc', 20)}Start Trip</button>` : ph === 'inprogress' ? `<button class="mbtn p mt12" data-go="#/driver/stops">Continue Boarding ${ic('arrowr', 18)}</button>` : `<button class="mbtn g mt12" data-go="#/driver/history">${ic('checkc', 18)}Trip Completed</button>`}`;
  },
});
function drvMap(h) {
  const pts = [[60, 250], [80, 210], [130, 190], [170, 150], [210, 110], [250, 70], [290, 40]];
  return mapSVG({ w: 360, h, seed: 61, labels: [[70, 120, 'Infocity', 13], [250, 140, 'Raysan', 13], [290, 30, 'Koba', 13], [70, 280, 'Kudasan', 13]], routes: [{ id: 'drvroute', pts, c: '#2563eb', w: 4 }], markers: [{ t: 'num', x: 60, y: 250, n: 5 }, { t: 'ring', x: 80, y: 210, c: '#2563eb' }, { t: 'num', x: 130, y: 190, n: 3 }, { t: 'ring', x: 170, y: 150, c: '#2563eb' }, { t: 'num', x: 210, y: 110, n: 3 }, { t: 'ring', x: 250, y: 70, c: '#2563eb' }, { t: 'num', x: 290, y: 40, n: 3 }, { t: 'tag', x: 110, y: 64, text: 'R-03', bg: '#8b5cf6', c: '#fff', w: 42 }, { t: 'bus', x: 130, y: 92, a: 0, anim: 'drvroute', dur: 40, s: 1.1 }, { t: 'call', x: 140, y: h - 60, lines: [['City Plus School', 600]] }] });
}

/* ---------- Start trip: driver photo ---------- */
dp('start', {
  title: 'Start Trip', back: '#/driver/trip', pad: false,
  render: () => `<div class="map" style="height:300px;border-radius:0">${drvMap(300)}</div><div style="padding:14px;background:#fff;min-height:420px"><div class="b fs15 tc">Please take your photo to start trip</div>
  <div style="position:relative;width:170px;height:170px;margin:14px auto;border-radius:12px;overflow:hidden;background:#cbd5e1">${UI.dcam ? `<div style="position:absolute;inset:0;display:flex;align-items:center;justify-content:center;background:#1f2937;color:#fff" class="col">${ic('refresh', 30, 'class="spin"')}<div class="fs12 mt8">Verifying face...</div></div>` : avatarSVG('Imran Shaikh', 'mb')}${D().photo ? `<span style="position:absolute;right:8px;top:8px" class="ck">${ic('check', 14, 'stroke-width="3"')}</span>` : ''}</div>
  <div class="tc"><button class="mic" style="background:#1463ff;color:#fff;border-radius:50%;width:56px;height:56px;margin:-38px auto 0;border:4px solid #fff;position:relative" data-act="dphoto">${ic('camera', 24)}</button></div>
  ${D().photo ? `<div class="mbanner mt12" style="background:#ecf9f0">${ic('checkc', 22, 'style="color:#16a34a"')}<div><div class="t fs14">Photo verified</div><div class="s">Checked in at ${D().photoAt || '06:58 AM'}. Next: complete the safety inspection.</div></div></div><button class="mbtn p mt12" data-go="#/driver/inspect">Continue to Safety Inspection ${ic('arrowr', 18)}</button>` : `<div class="fs12 mut tc mt8">Your face is matched with your driver profile before the trip begins.</div>`}</div>`,
});
ACT.dphoto = () => { UI.dcam = true; refresh(); setTimeout(() => { UI.dcam = false; D().photo = true; D().photoAt = nowTime(); save(); if (CUR.page === 'start') refresh(); toast('Photo captured and verified'); }, 1300); };

/* ---------- Safety inspection ---------- */
const inspState = k => D().insp[k] || null;
function inspStatusBadge(k) { const s = inspState(k); if (!s || !s.status) return `<span class="bd bd-gray" style="height:20px;font-size:10.5px">Not Checked</span>`; return s.status === 'pass' ? badge('Pass') : `<span class="bd bd-red dot">Failed</span>`; }
dp('inspect', {
  title: 'Safety Inspection', back: '#/driver/home',
  render: () => {
    const items = DATA.inspectionItems, done = items.filter(i => inspState(i.k) && inspState(i.k).status).length;
    return `<div class="mcard"><div class="row">${busSide(110)}<div><div class="b fs16">GJ01EF9012</div><div class="fs12 mut">Tata Starbus (52 Seater)</div><span class="bd bd-green mt4" style="margin-top:6px">${done === items.length ? 'Inspection Complete' : 'Ready for Inspection'}</span></div></div>
    <div class="row mt12 fs13 sb6">${ic('bus', 16)}R-03 &nbsp;East Zone</div><div class="row mt8 fs13">${ic('cal', 16)}<span style="flex:1">${TODAY}</span>${ic('clock', 16)}<span class="sb6">07:05 AM</span></div>
    <div class="mbanner mt12" style="background:#eef4fd">${ic('info', 20, 'style="color:#2563eb"')}<div class="s" style="margin:0;color:#1e3a8a">Complete all checks before starting the trip.<br>This ensures the safety of all students.</div></div>
    <div class="mt8">${items.map(i => `<button class="mrow" style="padding:9px 2px" data-go="#/driver/check/${i.k}">${ic(i.ic, 22, `style="color:${i.c}"`)}<span class="fs14" style="flex:1">${i.t}</span>${inspStatusBadge(i.k)}${ic('right', 16, 'style="color:#94a3b8"')}</button>`).join('')}</div>
    <div class="row mt8"><div class="prog" style="flex:1"><i style="width:${done / items.length * 100}%;background:#16a34a"></i></div><span class="fs12 sb6">${done}/${items.length}</span></div>
    <button class="mbtn p mt12" data-go="${done === items.length ? '#/driver/inspsummary' : '#/driver/check/' + (items.find(i => !(inspState(i.k) || {}).status) || items[0]).k}">${ic('play', 16)}${done === 0 ? 'Start Inspection' : done === items.length ? 'View Summary' : 'Continue Inspection'}</button></div>`;
  },
});
page('driver/check', {
  back: '#/driver/inspect', right: () => mhb('bell', 'data-go="#/driver/messages"'), pad: false,
  title: r => { const it = DATA.inspectionItems.find(i => i.k === r.sub[0]); return it ? (it.k === 'other' ? 'Other Checks' : it.t + ' Inspection') : 'Inspection'; },
  render: r => {
    const it = DATA.inspectionItems.find(i => i.k === r.sub[0]) || DATA.inspectionItems[0];
    if (it.k === 'other') return otherChecksBody();
    const st = D().insp[it.k] || {}, checks = st.checks || it.checks.map(() => true), photos = st.photos ?? it.photos.length;
    return `<div class="mhero" style="height:150px">${scene(it.scene)}</div><div style="padding:12px 14px;background:#fff;min-height:560px"><div class="row" style="align-items:flex-start"><div style="flex:1"><div class="b fs16">${it.t}</div><div class="fs12 mut mt4">${it.lead}</div></div>${ic('info', 20, 'style="color:#475569"')}</div>
    <div class="mt8">${it.checks.map((c, i) => { const [label, extra] = c.split('|'); return `<div class="chkrow"><span class="ck ${checks[i] ? '' : 'off'}" data-act="dck" data-k="${it.k}" data-i="${i}">${ic('check', 13, 'stroke-width="3"')}</span><span style="flex:1">${label}</span>${extra ? `<span class="c-green sb6 fs12">${extra}</span>` : ''}</div>`; }).join('')}</div>
    <div class="mlbl">Add Photos (Optional)</div><div class="mph">${Array.from({ length: photos }, (_, i) => `<div class="p">${scene(it.photos[i % (it.photos.length || 1)] || it.scene).replace('<svg ', '<svg width="100%" height="100%" ')}</div>`).join('')}<button class="add" data-act="dphotoadd" data-k="${it.k}">${ic('camera', 20)}</button></div>
    <div class="mlbl">Notes (Optional)</div><input class="minp" placeholder="Enter any observations..." data-bind="s:driver.insp.${it.k}.notes" value="${esc(st.notes || '')}"/>
    <div class="row mt16" style="gap:10px"><button class="mbtn rsoft" style="flex:1" data-go="#/driver/issue/${it.k}">${ic('warn', 18)}Mark as Issue</button><button class="mbtn p" style="flex:1" data-act="dnext" data-k="${it.k}">Next ${ic('arrowr', 18)}</button></div></div>`;
  },
});
function ensureInsp(k) { const it = DATA.inspectionItems.find(i => i.k === k); if (!D().insp[k]) D().insp[k] = { checks: it.checks.map(() => true), photos: it.photos.length, notes: '' }; return D().insp[k]; }
ACT.dck = d => { const s = ensureInsp(d.k); s.checks[d.i] = !s.checks[d.i]; save(); refresh(); };
ACT.dphotoadd = d => { const s = ensureInsp(d.k); s.photos = Math.min(6, (s.photos || 0) + 1); save(); refresh(); toast('Photo added'); };
ACT.dnext = d => {
  const s = ensureInsp(d.k);
  if (s.checks && s.checks.some(c => !c)) { toast('Some checks are not ticked. Tick them or mark as issue.', 'warn'); return; }
  s.status = 'pass'; s.issue = null; save();
  const items = DATA.inspectionItems, i = items.findIndex(x => x.k === d.k);
  go(i < items.length - 1 ? '#/driver/check/' + items[i + 1].k : '#/driver/inspsummary');
};
function otherChecksBody() {
  const st = D().other;
  return `<div style="padding:10px 12px;background:#fff;min-height:100%">${DATA.otherChecks.map((c, i) => `<div class="row" style="padding:7px 0;border-bottom:1px solid #f0f2f6"><span style="width:60px;height:44px;border-radius:7px;overflow:hidden;flex:none">${thumbIcon(c[2]).replace('<svg ', '<svg width="100%" height="100%" ')}</span><div style="flex:1"><div class="sb6 fs13">${c[0]}</div><div class="fs11 mut">${c[1]}</div></div>${ic('info', 16, 'style="color:#94a3b8"')}<span class="ck ${st[i] === false ? 'off' : ''}" data-act="dother" data-i="${i}">${ic('check', 13, 'stroke-width="3"')}</span></div>`).join('')}
  ${(D().customChecks || []).map((c, i) => `<div class="row" style="padding:9px 0;border-bottom:1px solid #f0f2f6">${tile('clip', '#2563eb', '#eaf1ff', 44, 20, 8)}<div style="flex:1"><div class="sb6 fs13">${esc(c)}</div><div class="fs11 mut">Custom check</div></div><span class="ck">${ic('check', 13, 'stroke-width="3"')}</span></div>`).join('')}
  <button class="mbtn soft sm mt12" data-act="dcustom">${ic('pluscircle', 18)}Add Custom Check</button>
  <div class="row mt16" style="gap:10px"><button class="mbtn rsoft" style="flex:1" data-go="#/driver/issue/other">${ic('warn', 18)}Mark as Issue</button><button class="mbtn p" style="flex:1" data-act="dothernext">Next ${ic('arrowr', 18)}</button></div></div>`;
}
ACT.dother = d => { D().other[d.i] = D().other[d.i] === false ? true : false; save(); refresh(); };
ACT.dcustom = () => openSheet(`<div class="b fs16">Add Custom Check</div><div class="mlbl">Check name</div><input class="minp" id="dcc" placeholder="e.g. Wheelchair ramp"/><button class="mbtn p mt16" data-act="dcustom2">Add Check</button>`);
ACT.dcustom2 = () => { const v = document.getElementById('dcc').value.trim(); if (!v) return; D().customChecks = [...(D().customChecks || []), v]; save(); closeSheet(); refresh(); };
ACT.dothernext = () => { if (Object.values(D().other).some(v => v === false)) { toast('Some checks are not ticked. Tick them or mark as issue.', 'warn'); return; } D().insp.other = { status: 'pass' }; save(); go('#/driver/inspsummary'); };
page('driver/issue', {
  title: r => { const it = DATA.inspectionItems.find(i => i.k === r.sub[0]); return (it ? it.t : 'Item') + ' Inspection'; }, back: true, right: () => mhb('bell', 'data-go="#/driver/messages"'), pad: false,
  render: r => {
    const it = DATA.inspectionItems.find(i => i.k === r.sub[0]) || DATA.inspectionItems[0];
    const f = UI.dissue || { desc: it.k === 'tyres' ? 'Left rear tyre has visible cut on sidewall.' : '', sev: 'major', act: 'Reported to Maintenance' };
    UI.dissue = f;
    return `<div class="mhero" style="height:110px">${scene(it.scene)}</div><div style="padding:12px 14px;background:#fff;min-height:620px"><div class="row"><span class="b fs14" style="flex:1">${(it.checks[0] || it.t).split('|')[0]}</span><span class="bd bd-red dot">Failed</span>${ic('info', 18, 'style="color:#475569;margin-left:6px"')}</div>
    <div class="mlbl">Issue Details</div><textarea class="minp" style="height:60px" data-bind="u:dissue.desc" placeholder="Describe the issue">${esc(f.desc)}</textarea>
    <div class="mlbl">Add Photos</div><div class="mph"><div class="p">${scene(it.scene).replace('<svg ', '<svg width="100%" height="100%" ')}</div><div class="p">${scene(it.photos[1] || it.scene).replace('<svg ', '<svg width="100%" height="100%" ')}</div><button class="add" data-act="toast" data-msg="Photo added">${ic('camera', 20)}</button></div>
    <div class="mlbl">Mark as</div><div class="row fs13" style="gap:14px">${[['minor', 'Minor (Can continue)'], ['major', 'Major (Do not start trip)']].map(x => `<label class="row" style="gap:6px;cursor:pointer;${x[0] === 'major' ? 'color:#dc2626' : ''}"><input type="radio" name="sev" ${f.sev === x[0] ? 'checked' : ''} data-act="set" data-k="u:dissue.sev" data-v="${x[0]}" style="accent-color:#ef4444;width:16px;height:16px"/>${x[1]}</label>`).join('')}</div>
    <div class="mlbl">Action Taken</div><div class="msel"><select data-bind="u:dissue.act" data-norefresh>${['Reported to Maintenance', 'Fixed on the spot', 'Vehicle replaced', 'Supervisor informed'].map(o => `<option ${f.act === o ? 'selected' : ''}>${o}</option>`).join('')}</select>${ic('down', 16)}</div>
    <button class="mbtn r mt16" data-act="dissuesave" data-k="${it.k}">${ic('xc', 18)}Save Issue & Continue</button></div>`;
  },
});
ACT.dissuesave = d => {
  const f = UI.dissue; if (!f.desc) { toast('Describe the issue', 'warn'); return; }
  const s = d.k === 'other' ? (D().insp.other = D().insp.other || {}) : ensureInsp(d.k);
  s.status = 'fail'; s.issue = { ...f }; save(); UI.dissue = null;
  toast(f.sev === 'major' ? 'Major issue logged. Maintenance has been alerted.' : 'Issue logged. You can continue.', f.sev === 'major' ? 'warn' : 'ok');
  const items = DATA.inspectionItems, i = items.findIndex(x => x.k === d.k);
  go(i < items.length - 1 ? '#/driver/check/' + items[i + 1].k : '#/driver/inspsummary');
};
dp('inspsummary', {
  title: 'Inspection Summary', back: '#/driver/inspect',
  render: () => {
    const items = DATA.inspectionItems, fails = items.filter(i => (inspState(i.k) || {}).status === 'fail'), major = fails.some(i => inspState(i.k).issue && inspState(i.k).issue.sev === 'major');
    const pending = items.filter(i => !(inspState(i.k) || {}).status);
    return `<div class="mcard"><div class="row">${busSide(100)}<div style="flex:1"><div class="b fs16">GJ01EF9012</div><div class="fs12 mut">Tata Starbus (52 Seater)</div></div>${pending.length ? badge('Pending') : fails.length ? `<span class="bd bd-red">${major ? 'Failed' : 'Passed with Issues'}</span>` : '<span class="bd bd-green">Passed</span>'}</div>
    <div class="row mt12 fs13 sb6">${ic('bus', 16)}R-03 &nbsp;East Zone</div><div class="row mt8 fs13">${ic('cal', 16)}<span style="flex:1">${TODAY}</span>${ic('clock', 16)}<span class="sb6">07:05 AM</span></div>
    <div class="mt8">${items.map(i => `<button class="mrow" style="padding:8px 2px" data-go="#/driver/check/${i.k}">${ic(i.ic, 20, `style="color:${i.c}"`)}<span class="fs13" style="flex:1">${i.t}</span>${inspStatusBadge(i.k)}${ic('right', 16, 'style="color:#94a3b8"')}</button>`).join('')}</div>
    <div class="mlbl">Inspector Notes</div><textarea class="minp" data-bind="s:driver.tripNotes" placeholder="Enter any additional notes...">${esc(D().tripNotes || '')}</textarea>
    ${major ? `<div class="mbanner mt12" style="background:#fdeeee">${ic('warn', 20, 'style="color:#dc2626"')}<div class="s" style="margin:0;color:#991b1b">A major issue was found. The trip cannot start until maintenance clears the vehicle or a replacement bus is assigned.</div></div>` : ''}
    <div class="row mt12" style="gap:10px"><button class="mbtn o" style="flex:1" data-act="dinspdraft">Save as Draft</button><button class="mbtn g" style="flex:1.4" data-act="dinspdone">${ic('checkc', 18)}Complete Inspection</button></div></div>`;
  },
});
ACT.dinspdraft = () => { save(); toast('Inspection saved as draft'); go('#/driver/home'); };
ACT.dinspdone = () => {
  const items = DATA.inspectionItems; const pending = items.filter(i => !(inspState(i.k) || {}).status);
  if (pending.length) { toast(`Complete ${pending.length} remaining check(s) first`, 'warn'); go('#/driver/check/' + pending[0].k); return; }
  const major = items.some(i => (inspState(i.k) || {}).issue && inspState(i.k).issue.sev === 'major');
  if (major) { openSheet(`<div class="b fs16">Major issue found</div><div class="fs13 mut mt8">Maintenance has been notified. For this demo you can assign the standby bus and continue.</div><button class="mbtn p mt16" data-act="dstandby">Use Standby Bus and Continue</button><button class="mbtn o mt8" data-act="sheetclose">Wait for Maintenance</button>`); return; }
  D().inspDone = true; if (D().phase === 'assigned') D().phase = 'inprogress'; save(); toast('Inspection completed. Trip started.'); go('#/driver/stops');
};
ACT.dstandby = () => { closeSheet(); D().inspDone = true; D().phase = 'inprogress'; save(); toast('Standby bus assigned. Trip started.'); go('#/driver/stops'); };

/* ---------- Boarding ---------- */
dp('stops', {
  title: 'My Trip - Stops', back: '#/driver/home', right: dBell,
  render: () => {
    const ph = D().phase, cur = curStop();
    return `<div class="mcard"><div class="row">${tile('busf', '#fff', '#8b5cf6', 42, 22, 11)}<div style="flex:1"><div class="b fs16">R-03 &nbsp;East Zone</div><div class="fs12 mut mt4">GJ01EF9012 | Tata Starbus</div></div>${busSide(70)}</div><div class="row fs13 sb6 mt12">${ic('cal', 17)}${TODAY}</div><div class="row mt12" style="padding:6px 0"><div class="row" style="flex:1">${ic('cal', 20)}<div><div class="fs11 mut">Start Time</div><div class="b fs16">07:05 AM</div></div></div><div class="row" style="flex:1">${ic('clock', 20)}<div><div class="fs11 mut">End Time</div><div class="b fs16">08:00 AM</div></div></div></div>
    ${ph === 'assigned' ? `<div class="mbanner" style="background:#fff7e8;margin:6px 0">${ic('info', 20, 'style="color:#b45309"')}<div class="s" style="margin:0">Complete the driver photo and safety inspection to start boarding.</div></div>` : ''}
    <div class="mt8">${DATA.drvStops.map((s, i) => { const c = stopCounts(i), done = D().stopsDone.includes(i), isCur = i === cur && ph === 'inprogress'; return `<button class="row" style="width:100%;text-align:left;padding:10px 8px;margin:4px 0;border-radius:10px;${isCur || i <= 1 ? 'background:#eef4ff;' : ''}${isCur ? 'border:1px solid #c9dbff' : ''}" data-act="dstop" data-i="${i}"><span class="mic sm" style="border-radius:50%;width:26px;height:26px;font-size:12px;font-weight:700;background:${done ? '#16a34a' : isCur || i <= 1 ? '#2563eb' : '#94a3b8'};color:#fff">${done ? ic('check', 14, 'stroke-width="3"') : i + 1}</span><div style="flex:1"><div class="b fs14">${s.name}</div><div class="fs12 mut">${s.time}</div></div><span class="fs13 sb6">${c.b}/${c.total}</span>${ic('right', 16, 'style="color:#64748b"')}</button>`; }).join('')}</div>
    <button class="mbtn p mt12" data-act="dstartboard">${ph === 'completed' ? 'Trip Completed' : 'Start Boarding'} ${ic('arrowr', 18)}</button></div>
    <button class="mbtn soft mt12" data-go="#/driver/allstops">${ic('list', 18)}All Stops Overview</button>`;
  },
});
ACT.dstop = d => { const i = Number(d.i); if (D().phase === 'assigned') { toast('Start the trip first: photo and safety inspection', 'warn'); go('#/driver/start'); return; } if (i === 0) { toast('Depot: no pickups at this stop'); return; } if (i === 5) { go('#/driver/complete'); return; } D().stop = i; save(); go('#/driver/board'); };
ACT.dstartboard = () => { if (D().phase === 'assigned') { toast('Take your photo and complete the inspection first', 'warn'); go(D().photo ? '#/driver/inspect' : '#/driver/start'); return; } if (D().phase === 'completed') { go('#/driver/history'); return; } go('#/driver/board'); };
dp('board', {
  title: 'Student Boarding', back: '#/driver/stops', right: dBell,
  render: () => {
    const i = curStop(), s = DATA.drvStops[i], c = stopCounts(i), t = UI.dbt || 'Total', q = (UI.dbq || '').toLowerCase();
    const kids = stopKids(i).filter(k => (t === 'Total' || kidStatus(k.id) === (t === 'Boarded' ? 'Boarded' : t === 'No Show' ? 'No Show' : 'Pending')) && (!q || (k.name + k.id).toLowerCase().includes(q)));
    return `<div class="mcard row"><span class="mic" style="background:#2563eb;color:#fff;border-radius:50%;width:34px;height:34px;font-weight:700">${i + 1}</span><div style="flex:1"><div class="b fs15">${s.name}</div><div class="fs12 mut">${s.time}</div></div><div class="tr"><div class="b fs15">${c.total}</div><div class="fs11 mut">Expected</div></div></div>
    <div class="row mt8" style="background:#fff;border-radius:12px;border:1px solid #e9eef5">${[['Total', c.total, '#0f172a'], ['Boarded', c.b, '#0f172a'], ['No Show', c.n, c.n ? '#dc2626' : '#0f172a'], ['Pending', c.p, '#0f172a']].map(x => `<button style="flex:1;padding:8px 0;text-align:center;border-bottom:2.5px solid ${t === x[0] ? '#2563eb' : 'transparent'}" data-act="set" data-k="u:dbt" data-v="${x[0]}"><div class="b fs16" style="color:${t === x[0] ? '#2563eb' : x[2]}">${x[1]}</div><div class="fs11" style="color:${t === x[0] ? '#2563eb' : x[0] === 'No Show' && c.n ? '#dc2626' : '#64748b'}">${x[0]}</div></button>`).join('')}</div>
    <div class="row mt8" style="gap:8px"><label class="search" style="flex:1">${ic('search', 16)}<input class="inp" style="height:40px" placeholder="Search student by name or ID..." data-bind="u:dbq" data-live value="${esc(UI.dbq || '')}"/></label><button class="ibtn blue" style="width:40px;height:40px" data-go="#/driver/scan">${ic('scan', 18)}</button></div>
    <div class="mcard mt8" style="padding:2px 10px">${kids.length ? kids.map(k => { const st = kidStatus(k.id); return `<button class="mrow" style="padding:8px 0" data-go="#/driver/kid/${k.id}">${av(k.name, 38)}<div style="flex:1"><div class="t fs13">${k.name}</div><div class="s">ID: ${k.id} | ${k.cls}</div></div><div class="tr">${st === 'Boarded' ? `<span class="bd bd-green">Boarded</span><div class="fs11 mut mt4">${D().boardTime[k.id] || '07:12 AM'}</div>` : st === 'No Show' ? '<span class="bd bd-red">No Show</span>' : '<span class="bd bd-orange">Pending</span>'}</div>${ic('right', 16, 'style="color:#64748b"')}</button>`; }).join('') : '<div class="tc mut fs13" style="padding:20px">No students in this list</div>'}</div>
    <div class="row mt12" style="gap:10px"><button class="mbtn o" style="flex:1;font-size:14px" data-go="#/driver/scan">${ic('scan', 18)}Scan RFID/QR</button><button class="mbtn p" style="flex:1;font-size:14px" data-go="#/driver/manual">${ic('usercheck', 18)}Manual Verify</button></div>
    <div class="row mt8" style="gap:10px"><button class="mbtn osoft sm" style="flex:1" data-act="dnoshowpick">${ic('dotfill', 14, 'style="color:#f97316"')}Mark No Show</button><button class="mbtn g sm" style="flex:1" data-go="#/driver/stopstatus">${ic('flag', 16)}Stop Status</button></div>`;
  },
});
ACT.dnoshowpick = () => { const p = stopKids(curStop()).filter(k => kidStatus(k.id) === 'Pending'); if (!p.length) { toast('No pending students at this stop'); return; } openSheet(`<div class="b fs16 mb8">Mark No Show</div>${p.map(k => `<button class="mrow" data-go="#/driver/noshow/${k.id}">${av(k.name, 36)}<div style="flex:1"><div class="t fs13">${k.name}</div><div class="s">ID: ${k.id} | ${k.cls}</div></div>${ic('right', 16)}</button>`).join('')}`); };
const allKids = () => [1, 2, 3, 4].flatMap(i => stopKids(i).map(k => ({ ...k, stopI: i })));
const kidById = id => allKids().find(k => k.id === id);
function boardKid(id, how) { D().board[id] = 'Boarded'; D().boardTime[id] = nowTime(); D().how = D().how || {}; D().how[id] = how; save(); }
dp('scan', {
  title: 'Scan Student', back: '#/driver/board', right: dBell, pad: false,
  render: () => {
    const t = UI.dst || 'QR Code', next = stopKids(curStop()).find(k => kidStatus(k.id) === 'Pending');
    return `<div class="mtabs">${['QR Code', 'RFID Card', 'Manual Entry'].map(x => `<button class="mtab ${t === x ? 'on' : ''}" ${x === 'Manual Entry' ? 'data-go="#/driver/manual"' : `data-act="set" data-k="u:dst" data-v="${x}"`}>${x}</button>`).join('')}</div>
    <div class="scanbox" style="height:660px;background:linear-gradient(#3c4352,#262b35)">
      ${next ? `<button style="position:absolute;left:60px;right:60px;top:40px;height:330px;border-radius:12px;background:#e6ebf2;box-shadow:0 10px 30px rgba(0,0,0,.4);overflow:hidden" data-act="dscan">
        <div style="height:200px">${avatarSVG(next.name, DATA.kindOf(next.name)).replace('<svg ', '<svg width="100%" height="100%" preserveAspectRatio="xMidYMid slice" ')}</div>
        <div class="row" style="padding:10px 12px;align-items:flex-start"><div style="flex:1;text-align:left"><div class="b fs16">${next.name}</div><div class="fs12 mut">ID: ${next.id}</div></div>${ic('shieldcheck', 30, 'style="color:#1e3a8a"')}</div>
        <div style="width:${t === 'RFID Card' ? 80 : 90}px;height:${t === 'RFID Card' ? 50 : 90}px;margin:-8px auto 0">${t === 'RFID Card' ? `<div style="width:80px;height:50px;border-radius:8px;background:linear-gradient(135deg,#1e3a8a,#2563eb);display:flex;align-items:center;justify-content:center;color:#fff">${ic('radio', 26)}</div>` : qrArt(hashStr(next.id))}</div></button>
        <div style="position:absolute;left:52px;right:52px;top:32px;height:346px;pointer-events:none"><span style="position:absolute;left:0;top:0;width:40px;height:40px;border-left:4px solid #22c55e;border-top:4px solid #22c55e;border-radius:10px 0 0 0"></span><span style="position:absolute;right:0;top:0;width:40px;height:40px;border-right:4px solid #22c55e;border-top:4px solid #22c55e;border-radius:0 10px 0 0"></span><span style="position:absolute;left:0;bottom:0;width:40px;height:40px;border-left:4px solid #22c55e;border-bottom:4px solid #22c55e;border-radius:0 0 0 10px"></span><span style="position:absolute;right:0;bottom:0;width:40px;height:40px;border-right:4px solid #22c55e;border-bottom:4px solid #22c55e;border-radius:0 0 10px 0"></span><span class="scanline" style="position:absolute;left:8px;right:8px;height:2px;background:#22c55e;box-shadow:0 0 12px #22c55e;animation:scanl 2s ease-in-out infinite alternate"></span></div>
        <div style="position:absolute;left:0;right:0;top:396px;text-align:center;color:#fff;font-size:15px">${t === 'RFID Card' ? 'Tap RFID card on the reader' : 'Align QR code within the frame'}<div class="fs12" style="opacity:.7;margin-top:4px">Tap the card to simulate a scan</div></div>`
        : `<div style="position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;color:#fff;gap:10px">${ic('checkc', 48, 'style="color:#22c55e"')}<div class="b fs16">All students at this stop are verified</div><button class="mbtn p sm" style="width:auto;padding:0 18px" data-go="#/driver/stopstatus">View Stop Status</button></div>`}
      <button class="mic" style="position:absolute;left:24px;bottom:110px;width:58px;height:58px;border-radius:50%;background:rgba(0,0,0,.35);border:2px solid #fff;color:#fff" data-act="set" data-k="u:torch" data-v="1" data-toggle data-msg="Flashlight toggled">${ic('torch', 24)}</button>
      <button class="mic" style="position:absolute;right:24px;bottom:110px;width:58px;height:58px;border-radius:50%;background:rgba(0,0,0,.35);border:2px solid #fff;color:#fff" data-act="toast" data-msg="Gallery opened. Pick a QR image.">${ic('image', 24)}</button>
    </div><style>@keyframes scanl{from{top:12px}to{top:330px}}</style>`;
  },
});
ACT.dscan = () => { const next = stopKids(curStop()).find(k => kidStatus(k.id) === 'Pending'); if (!next) return; boardKid(next.id, UI.dst === 'RFID Card' ? 'RFID Card' : 'QR Code'); go('#/driver/verified/' + next.id); };
page('driver/verified', {
  title: 'Student Verified', back: '#/driver/board', right: dBell,
  render: r => {
    const k = kidById(r.sub[0]) || DATA.boardKids[0];
    const conf = Array.from({ length: 14 }, (_, i) => `<i style="left:${10 + (i * 37) % 80}%;top:${(i * 23) % 60}px;background:${['#f59e0b', '#2563eb', '#ef4444', '#22c55e', '#8b5cf6'][i % 5]};animation-delay:${(i % 5) * .08}s;transform:rotate(${i * 40}deg)"></i>`).join('');
    return `<div class="mcard tc" style="padding:20px 14px;position:relative;overflow:hidden"><div class="confetti" style="position:absolute;left:0;right:0;top:6px;height:100px">${conf}</div><div style="position:relative;width:110px;margin:10px auto 0">${av(k.name, 110)}<span class="ck" style="position:absolute;right:0;bottom:4px;width:32px;height:32px;border:3px solid #fff">${ic('check', 18, 'stroke-width="3"')}</span></div>
    <div class="b mt12" style="font-size:24px">${k.name}</div><div class="fs15 mut mt4">ID: ${k.id} | ${k.cls}</div><span class="bd bd-green" style="height:34px;padding:0 22px;font-size:15px;margin-top:12px">Boarded Successfully</span>
    <div class="mt16" style="text-align:left">${[['clock', 'Boarding Time', D().boardTime[k.id] || '07:12 AM'], ['pin', 'Stop', DATA.drvStops[k.stopI || 1].name], ['bus', 'Bus', 'GJ01EF9012 (R-03)'], ['idcard', 'Verified by', (D().how || {})[k.id] || 'QR Code']].map(x => `<div class="row" style="padding:10px 0;border-bottom:1px solid #f0f2f6">${ic(x[0], 18)}<span class="fs13" style="flex:1">${x[1]}</span><span class="fs13 sb6">${x[2]}</span></div>`).join('')}</div>
    <button class="mbtn p mt16" data-act="dnextkid">Next Student ${ic('arrowr', 18)}</button><button class="link mt12" style="width:100%;justify-content:center" data-go="#/driver/kid/${k.id}">View Student Details</button></div>`;
  },
});
ACT.dnextkid = () => { const next = stopKids(curStop()).find(k => kidStatus(k.id) === 'Pending'); go(next ? '#/driver/scan' : '#/driver/stopstatus'); };
dp('manual', {
  title: 'Manual Verification', back: '#/driver/board',
  render: () => {
    const q = (UI.dmq || '').toLowerCase(), list = stopKids(curStop()).filter(k => kidStatus(k.id) === 'Pending' && (!q || (k.name + k.id).toLowerCase().includes(q))), selId = UI.dmsel;
    return `<div class="mcard"><div class="mlbl" style="margin-top:0">Search by Name or ID</div><label class="search" style="flex-direction:row-reverse"><input class="minp" placeholder="Enter student name or ID" data-bind="u:dmq" data-live value="${esc(UI.dmq || '')}"/>${ic('search', 18, 'style="left:auto;right:12px"')}</label>
    <div class="mlbl">Select Student</div>${list.length ? list.map(k => `<button class="mrow" style="padding:9px 4px;${selId === k.id ? 'background:#eef4ff;border-radius:10px' : ''}" data-act="set" data-k="u:dmsel" data-v="${k.id}">${av(k.name, 40)}<div style="flex:1"><div class="t fs13">${k.name}</div><div class="s">ID: ${k.id} | ${k.cls}</div></div>${selId === k.id ? ic('checkc', 20, 'style="color:#2563eb"') : ic('right', 16, 'style="color:#64748b"')}</button>`).join('') : '<div class="mut fs13 tc" style="padding:16px">No pending students match</div>'}
    <button class="mbtn p mt16 ${selId ? '' : 'dis'}" data-act="dmanual">${ic('scan', 18)}Mark as Boarded</button></div>`;
  },
});
ACT.dmanual = () => { const id = UI.dmsel; if (!id) return; boardKid(id, 'Manual (Driver)'); UI.dmsel = null; go('#/driver/verified/' + id); };
page('driver/noshow', {
  title: 'Mark No Show', back: '#/driver/board',
  render: r => { const k = kidById(r.sub[0]) || DATA.boardKids[4]; const f = UI.dns || { reason: 'Not at stop', notes: `Not present at stop at ${DATA.drvStops[k.stopI || 1].time.replace('07:12', '07:15')}` }; UI.dns = f;
    return `<div class="mcard"><div class="row">${av(k.name, 56)}<div><div class="b fs16">${k.name}</div><div class="fs12 mut">ID: ${k.id} | ${k.cls}</div></div></div>
    <div class="mlbl">Reason (Optional)</div><div class="msel"><select data-bind="u:dns.reason" data-norefresh>${['Not at stop', 'Parent informed absence', 'Sick', 'Arrived late', 'Other'].map(o => `<option ${f.reason === o ? 'selected' : ''}>${o}</option>`).join('')}</select>${ic('down', 16)}</div>
    <div class="mlbl">Notes</div><textarea class="minp" data-bind="u:dns.notes">${esc(f.notes)}</textarea>
    <button class="mbtn rsoft mt16" data-act="dnoshow" data-id="${k.id}">${ic('xc', 18)}Mark as No Show</button><div class="fs12 mut tc mt8">The parent will be notified immediately.</div></div>`; },
});
ACT.dnoshow = d => { D().board[d.id] = 'No Show'; D().noShowReason[d.id] = UI.dns; UI.dns = null; save(); closeSheet(); toast('Marked as No Show. Parent notified.', 'warn'); go('#/driver/board'); };
dp('stopstatus', {
  title: 'Stop Status', back: '#/driver/board',
  render: () => {
    const i = curStop(), s = DATA.drvStops[i], c = stopCounts(i), pct = c.total ? Math.round((c.b + c.n) / c.total * 100) : 100;
    const last = i === 4;
    return `<div class="mcard row"><span class="mic" style="background:#2563eb;color:#fff;border-radius:50%;width:34px;height:34px;font-weight:700">${i + 1}</span><div style="flex:1"><div class="b fs15">${s.name}</div><div class="fs12 mut">${s.time}</div></div><div class="tr"><div class="b fs15">${c.total}</div><div class="fs11 mut">Expected</div></div></div>
    <div class="mcard">${[['checkc', '#16a34a', 'Boarded', c.b, 'Boarded'], ['xc', '#ef4444', 'No Show', c.n, 'No Show'], ['clock', '#f59e0b', 'Pending', c.p, 'Pending']].map(x => `<button class="row" style="width:100%;padding:11px 0;border-bottom:1px solid #f0f2f6" data-act="set" data-k="u:dbt" data-v="${x[4]}" data-msg="Filtered: ${x[2]}">${ic(x[0], 22, `style="color:${x[1]};fill:${x[1]}22"`)}<span class="sb6 fs14" style="flex:1;text-align:left">${x[2]}</span><span class="b fs15" style="color:${x[1]}">${x[3]}</span></button>`).join('')}
    <div class="prog mt12" style="height:7px"><i style="width:${pct}%;background:#0e9f8e"></i></div><div class="row mt8 fs13"><span class="c-green sb6" style="flex:1">${c.b + c.n} / ${c.total} completed</span><span class="sb6">${pct}%</span></div>
    ${c.p ? `<div class="mbanner mt12" style="background:#fff7e8">${ic('info', 18, 'style="color:#b45309"')}<div class="s" style="margin:0">${c.p} student(s) pending. Scan them, verify manually or mark No Show before completing the stop.</div></div><button class="mbtn o mt12" data-go="#/driver/board">Back to Student List</button>` : ''}
    <button class="mbtn p mt12 ${c.p ? 'dis' : ''}" data-act="dcompletestop">${ic('flag', 18)}${last ? 'Complete Stop & Head to School' : 'Complete Stop'}</button></div>`;
  },
});
ACT.dcompletestop = () => {
  const i = curStop(); if (!D().stopsDone.includes(i)) D().stopsDone.push(i);
  if (i < 4) { D().stop = i + 1; save(); toast(`${DATA.drvStops[i].name} completed. Next: ${DATA.drvStops[i + 1].name}`); go('#/driver/allstops'); }
  else { D().stop = 5; save(); toast('All pickups completed. Heading to City Plus School.'); go('#/driver/complete'); }
};
dp('allstops', {
  title: 'All Stops Overview', back: '#/driver/stops',
  render: () => `<div class="mcard" style="padding:8px 10px">${DATA.drvStops.map((s, i) => { const c = stopCounts(i), done = D().stopsDone.includes(i) || (i === 0 && D().phase !== 'assigned'), cur = i === curStop() && D().phase === 'inprogress'; return `<button class="row" style="width:100%;text-align:left;padding:10px 6px;position:relative;${cur ? 'background:#eef4ff;border-radius:10px' : ''}" data-act="dstop" data-i="${i}"><span class="mic sm" style="border-radius:50%;width:26px;height:26px;font-size:12px;font-weight:700;background:${done ? '#16a34a' : cur ? '#2563eb' : '#94a3b8'};color:#fff">${done ? ic('check', 14, 'stroke-width="3"') : i + 1}</span><div style="flex:1"><div class="b fs14">${s.name}</div><div class="fs12 mut">${s.time}</div></div><span class="fs13 sb6">${c.b}/${c.total}</span>${cur ? `<span class="mic sm" style="width:22px;height:22px;border-radius:50%;background:#f59e0b;color:#fff;margin-left:6px">${ic('clock', 13)}</span>` : ic('right', 16, 'style="color:#94a3b8;margin-left:6px"')}</button>`; }).join('')}</div>
  <button class="mbtn p mt12" data-act="dstartboard">${D().phase === 'completed' ? 'Trip Completed' : 'Go to Current Stop'} ${ic('arrowr', 18)}</button>`,
});
page('driver/kid', {
  title: 'Student Details (Read Only)', back: '#/driver/board',
  render: r => { const k = kidById(r.sub[0]) || DATA.boardKids[0], st = kidStatus(k.id);
    return `<div class="mcard"><div class="row">${av(k.name, 60)}<div><div class="b fs16">${k.name}</div><div class="fs12 mut">ID: ${k.id} | ${k.cls}</div></div></div>
    <div class="mt12">${[['phone', 'Parent', '+91 98765 43210', true], ['pin', 'Pickup Stop', DATA.drvStops[k.stopI || 1].name + '<br>' + DATA.drvStops[k.stopI || 1].time], ['bus', 'Route', 'R-03 East Zone'], ['school', 'School', 'City Plus School']].map(x => `<div class="row" style="padding:12px 0;border-bottom:1px solid #f0f2f6;align-items:flex-start">${ic(x[0], 20, 'style="color:#1e3a8a"')}<span class="fs13 mut" style="width:96px">${x[1]}</span><span class="fs13 sb6" style="flex:1">${x[2]}</span>${x[3] ? `<button class="mic sm" style="background:#eaf1ff;color:#2563eb;width:30px;height:30px" data-act="call" data-who="Parent of ${k.name}">${ic('phone', 15)}</button>` : ''}</div>`).join('')}
    <div class="row" style="padding:12px 0;align-items:flex-start">${ic('checkc', 20, `style="color:${st === 'Boarded' ? '#16a34a' : st === 'No Show' ? '#ef4444' : '#f59e0b'}"`)}<span class="fs13 mut" style="width:96px">Boarding Status</span><div>${badge(st, false)}${D().boardTime[k.id] ? `<div class="fs12 mut mt4">${D().boardTime[k.id]}</div>` : ''}</div></div></div>
    ${st === 'Pending' ? `<div class="row" style="gap:10px"><button class="mbtn p sm" style="flex:1" data-act="dmanual1" data-id="${k.id}">Mark Boarded</button><button class="mbtn rsoft sm" style="flex:1" data-go="#/driver/noshow/${k.id}">No Show</button></div>` : ''}</div>`; },
});
ACT.dmanual1 = d => { boardKid(d.id, 'Manual (Driver)'); go('#/driver/verified/' + d.id); };

/* ---------- Complete trip ---------- */
dp('complete', {
  title: 'Complete Trip', back: '#/driver/stops',
  render: () => {
    const done = D().phase === 'completed', boarded = allKids().filter(k => kidStatus(k.id) === 'Boarded').length;
    return `<div class="mcard tc" style="padding:18px 14px">${tripDoneArt()}<div class="b fs20 mt12">${done ? 'Trip Completed!' : 'Arrived at City Plus School'}</div><div class="fs13 mut mt4">${done ? 'All stops completed and students dropped safely.' : 'Confirm all students have been dropped safely.'}</div>
    <div class="stat3 mt16">${[['cal', `${done ? 18 : Math.max(D().stopsDone.length, 0) + 13}/18`, 'Stops'], ['users', `${done ? 24 : Math.min(24, boarded + 12)}/24`, 'Students'], ['clock', '54 min', 'Duration']].map(x => `<div style="flex-direction:column;align-items:center;gap:2px"><div class="row" style="gap:6px">${ic(x[0], 18)}<span class="b fs15">${x[1]}</span></div><span class="fs11 mut">${x[2]}</span></div>`).join('')}</div>
    ${UI.dnotes ? `<textarea class="minp mt12" data-bind="s:driver.tripNotes" placeholder="Add notes for the transport desk">${esc(D().tripNotes || '')}</textarea>` : `<button class="mbtn soft mt16" data-act="set" data-k="u:dnotes" data-v="1">Add Notes (Optional)</button>`}
    ${done ? `<button class="mbtn o mt12" data-go="#/driver/history">View Trip History</button>` : `<button class="mbtn g mt12" data-act="dcomplete">${ic('checkc', 20)}Complete Trip</button>`}</div>`;
  },
});
ACT.dcomplete = () => { D().phase = 'completed'; D().stopsDone = [0, 1, 2, 3, 4, 5]; save(); toast('Trip completed. Parents and school notified.'); refresh(); };

/* ---------- Incidents ---------- */
dp('incidents', {
  title: 'Incidents', back: '#/driver/home', nav: 'driver:incidents',
  render: () => `<div class="mbanner" style="background:#fdeeee">${tile('warn', '#fff', '#ef4444', 50, 26, 12)}<div><div class="t">Stay Safe</div><div class="s">Report any incident immediately for quick assistance.</div></div></div>
  <div class="mcard mt12" style="padding:2px 12px">${Object.entries(INC_TYPES).map(([k, t]) => `<button class="mrow" style="padding:13px 0" data-go="${k === 'emergency' ? '#/driver/emergency' : '#/driver/report/' + k}">${tile(t.ic, '#fff', t.c, 42, 22, 11)}<div style="flex:1"><div class="t">${t.n === 'Accident' ? 'Report Accident' : t.n}</div><div class="s">${t.s}</div></div>${ic('right', 18, 'style="color:#64748b"')}</button>`).join('')}</div>
  <button class="mbtn soft mt12" data-go="#/driver/inclist">${ic('list', 18)}My Incidents (${D().incidents.length})</button>`,
});
page('driver/report', {
  title: r => (INC_TYPES[r.sub[0]] || INC_TYPES.accident).t, back: '#/driver/incidents', nav: 'driver:incidents',
  render: r => {
    const k = r.sub[0] || 'accident', t = INC_TYPES[k], f = UI['inc_' + k] || (UI['inc_' + k] = { desc: t.desc, issue: (t.issue || [''])[0], photos: [...t.photos], when: '2026-09-26T07:05', student: 'Aarav Sharma (Grade 5)' });
    return `<div class="mbanner" style="background:${t.bg}">${tile(t.ic, t.c, '#fff', 50, 26, 12)}<div><div class="t">${t.t.replace('Report ', '') === 'Accident' ? 'Report Accident' : t.n}</div><div class="s">${t.lead}</div></div></div>
    ${k === 'accident' ? `<div class="mlbl">Date & Time</div><input class="minp" type="datetime-local" data-bind="u:inc_${k}.when" value="${f.when}"/>` : ''}
    ${k === 'student' ? `<div class="mlbl">Student</div><div class="msel"><select data-bind="u:inc_${k}.student" data-norefresh>${DATA.boardKids.map(b => `<option ${f.student.startsWith(b.name) ? 'selected' : ''}>${b.name} (${b.cls})</option>`).join('')}</select>${ic('down', 16)}</div>` : ''}
    <div class="mlbl">${k === 'route' ? 'Current Location' : 'Location'}</div><div class="minp row">${ic('pin', 17)}<span class="fs13" style="flex:1">R-03 &nbsp;East Zone</span>${ic('crosshair', 17, 'style="color:#475569"')}</div>
    ${t.issue ? `<div class="mlbl">Issue Type</div><div class="msel"><select data-bind="u:inc_${k}.issue" data-norefresh>${t.issue.map(o => `<option ${f.issue === o ? 'selected' : ''}>${o}</option>`).join('')}</select>${ic('down', 16)}</div>` : ''}
    <div class="mlbl">Description</div><textarea class="minp" data-bind="u:inc_${k}.desc">${esc(f.desc)}</textarea>
    <div class="mlbl">Photos (Optional)</div><div class="mph">${f.photos.map((p, i) => `<div class="p">${scene(p).replace('<svg ', '<svg width="100%" height="100%" ')}<button class="x" data-act="incpdel" data-k="${k}" data-i="${i}">${ic('x', 11)}</button></div>`).join('')}<button class="add" data-act="incpadd" data-k="${k}">${ic('camera', 20)}</button></div>
    <button class="mbtn p mt16" data-act="incsubmit" data-k="${k}">${ic('send', 18)}Submit Report</button>`;
  },
});
ACT.incpadd = d => { const f = UI['inc_' + d.k]; f.photos.push(['engine', 'traffic', 'interior', 'brake'][f.photos.length % 4]); refresh(); };
ACT.incpdel = d => { UI['inc_' + d.k].photos.splice(Number(d.i), 1); refresh(); };
ACT.incsubmit = d => {
  const t = INC_TYPES[d.k], f = UI['inc_' + d.k];
  if (!f.desc.trim()) { toast('Add a short description', 'warn'); return; }
  const id = 'INC-2026-' + String(14 + D().incidents.filter(i => i.mine).length).padStart(3, '0');
  D().incidents.unshift({ id, type: d.k === 'route' ? 'Traffic / Route Issue' : t.n, loc: 'R-03 East Zone', time: '26 Sep 2026, ' + nowTime(), status: 'In Progress', desc: f.desc, photos: f.photos.length, mine: true });
  UI['inc_' + d.k] = null; save(); go('#/driver/incok/' + id);
};
dp('emergency', {
  title: 'Emergency', back: '#/driver/incidents', nav: 'driver:incidents',
  render: () => `<div class="mbanner" style="background:#fdeeee">${tile('siren', '#ef4444', '#fff', 50, 26, 12)}<div><div class="t">Emergency</div><div class="s">For serious issues, use the SOS button or report an emergency.</div></div></div>
  <button class="mbtn r mt16" style="height:58px;flex-direction:column;gap:0;font-size:20px" data-act="dsos"><span class="row">${ic('phone', 22)}<b>SOS</b></span><span class="fs12" style="font-weight:500">Tap to alert immediately</span></button>
  <div class="sb6 fs13 mt16 mb8">Or report specific emergency type:</div><div class="mcard" style="padding:2px 12px">${[['medical', 'Medical Emergency', '#ef4444'], ['flame', 'Fire', '#ef4444'], ['shield', 'Security Threat', '#2563eb'], ['storm', 'Natural Disaster', '#2563eb'], ['moreh', 'Other', '#64748b']].map(x => `<button class="mrow" style="padding:11px 0" data-act="demtype" data-t="${x[1]}"><span class="mic sm" style="background:${x[2]};color:#fff;border-radius:50%;width:30px;height:30px">${ic(x[0], 16)}</span><span class="fs14" style="flex:1">${x[1]}</span>${ic('right', 16, 'style="color:#64748b"')}</button>`).join('')}</div>`,
});
ACT.dsos = () => openSheet(`<div class="tc"><span class="mic" style="background:#fde8e8;color:#ef4444;margin:6px auto;width:64px;height:64px;border-radius:50%">${ic('siren', 32)}</span><div class="b fs18 mt8">Send SOS?</div><div class="fs13 mut mt4">Dispatcher, school and nearby team will receive your live location for GJ01EF9012.</div></div><button class="mbtn r mt16" data-act="dsos2" data-t="SOS">${ic('warn', 18)}Send SOS Now</button><button class="mbtn o mt8" data-act="sheetclose">Cancel</button>`);
ACT.demtype = d => openSheet(`<div class="b fs16">${esc(d.t)}</div><div class="fs13 mut mt4">This alerts the dispatcher immediately with your location.</div><div class="mlbl">Details (optional)</div><textarea class="minp" id="demd" placeholder="What happened?"></textarea><button class="mbtn r mt16" data-act="dsos2" data-t="${esc(d.t)}">${ic('siren', 18)}Send Alert</button>`);
ACT.dsos2 = d => { closeSheet(); const id = 'INC-2026-' + String(14 + D().incidents.filter(i => i.mine).length).padStart(3, '0'); D().incidents.unshift({ id, type: d.t === 'SOS' ? 'SOS Alert' : d.t, loc: 'R-03 East Zone', time: '26 Sep 2026, ' + nowTime(), status: 'In Progress', desc: (document.getElementById('demd') || {}).value || 'Emergency alert raised by driver.', mine: true }); save(); toast('Alert sent. Dispatcher is calling you.', 'warn'); go('#/driver/incok/' + id); };
dp('inclist', {
  title: 'Incidents', back: '#/driver/incidents', nav: 'driver:incidents',
  render: () => {
    const f = UI.dif || 'All Types', types = ['All Types', ...new Set(D().incidents.map(i => i.type))];
    const list = D().incidents.filter(i => f === 'All Types' || i.type === f);
    const icFor = t => /Break/.test(t) ? ['tools', '#f97316'] : /Accident/.test(t) ? ['crash', '#2563eb'] : /Student/.test(t) ? ['users', '#2563eb'] : /Route|Traffic/.test(t) ? ['routes', '#16a34a'] : ['siren', '#ef4444'];
    return `<div class="row" style="justify-content:flex-end">${sel('u:dif', types, f)}</div><div class="mcard mt8" style="padding:2px 12px">${list.map(i => { const [icn, c] = icFor(i.type); return `<button class="mrow" style="padding:12px 0" data-go="#/driver/incdetail/${i.id}">${tile(icn, '#fff', c, 38, 20, 10)}<div style="flex:1"><div class="t fs14">${esc(i.type)}</div><div class="s">${esc(i.loc)}</div><div class="s">${esc(i.time)}</div></div>${badge(i.status, false)}</button>`; }).join('')}</div>`;
  },
});
page('driver/incdetail', {
  title: 'Incident Details', back: '#/driver/inclist', nav: 'driver:incidents',
  render: r => { const i = D().incidents.find(x => x.id === r.sub[0]) || D().incidents[0];
    return `<div class="mcard"><div class="row" style="align-items:flex-start">${tile('crash', '#ef4444', '#fdeeee', 46, 24, 12)}<div style="flex:1"><div class="b fs16">${esc(i.type)}</div><div class="fs12 mut">ID: ${i.id}</div><div class="fs12 mut">${esc(i.time)}</div></div>${badge(i.status, false)}</div>
    <div class="mt12">${infoRow('pin', '#334155', 'Location', esc(i.loc))}${infoRow('bus', '#334155', 'Bus', 'GJ01EF9012')}${infoRow('file', '#334155', 'Description', esc(i.desc))}</div>
    <div class="row mt12 fs12 sb6">${ic('camera', 16)}Photos</div><div class="mph mt8"><div class="p" style="width:80px;height:64px">${scene('boarding').replace('<svg ', '<svg width="100%" height="100%" ')}</div><div class="p" style="width:80px;height:64px">${scene('brake').replace('<svg ', '<svg width="100%" height="100%" ')}</div><div class="p" style="width:80px;height:64px;background:#7d8ea8;color:#fff;display:flex;align-items:center;justify-content:center;font-size:18px">+2</div></div>
    <div class="mbanner mt12" style="background:#eef4fd">${ic('clip', 20, 'style="color:#2563eb"')}<div><div class="t fs14">Latest Update</div><div class="s">${i.status === 'Resolved' ? 'Resolved by transport desk.' : 'Maintenance team is on the way.<br>ETA: 20 minutes.'}</div></div></div>
    <button class="mbtn soft mt12" data-go="#/driver/incphotos/${i.id}">${ic('camera', 18)}Add Photos & Notes</button></div>`; },
});
page('driver/incphotos', {
  title: 'Add Photos & Notes', back: true,
  render: r => { const n = UI.dipc ?? 3;
    return `<div class="mcard"><div class="sb6 fs13">Photos</div><div class="mph mt8">${Array.from({ length: n }, (_, i) => `<div class="p" style="width:84px;height:70px">${scene(['boarding', 'brake', 'traffic', 'engine', 'interior'][i % 5]).replace('<svg ', '<svg width="100%" height="100%" ')}</div>`).join('')}<button class="add" style="width:84px;height:70px" data-act="set" data-k="u:dipc" data-v="${n + 1}" data-num>${ic('camera', 20)}Add Photo</button></div>
    <div class="mlbl mt16">Notes</div><textarea class="minp" id="dipn" placeholder="Add any additional information..."></textarea><div class="mlbl">Location</div><div class="minp row">${ic('pin', 17)}<span class="fs13" style="flex:1">R-03 East Zone</span>${ic('crosshair', 17)}</div>
    <button class="mbtn p mt16" data-act="dipsave" data-id="${r.sub[0]}">Save & Continue</button></div>`; },
});
ACT.dipsave = d => { UI.dipc = undefined; toast('Photos and notes added to incident'); go('#/driver/incdetail/' + d.id); };
page('driver/incok', {
  title: '', header: () => `<div class="mh" style="min-height:20px;padding:0 0 14px"></div>`,
  render: r => { const i = D().incidents.find(x => x.id === r.sub[0]) || D().incidents[0];
    return `<div class="mcard tc" style="padding:24px 16px;background:linear-gradient(#f2fbf5,#fff)"><span class="mic" style="background:#16a34a;color:#fff;border-radius:50%;width:60px;height:60px;margin:0 auto">${ic('check', 32, 'stroke-width="3"')}</span><div class="b fs20 mt12">Incident Reported</div><div class="fs13 mut mt8">Your incident has been reported successfully.<br>The relevant team has been notified.</div>
    <div class="mcard mt16" style="text-align:left">${[['cal', 'Incident ID', i.id], ['gear', 'Type', esc(i.type)], ['pin', 'Location', esc(i.loc)], ['user', 'Time', esc(i.time)]].map(x => `<div class="row" style="padding:10px 0;border-bottom:1px solid #f0f2f6">${ic(x[0], 17)}<span class="fs13 mut" style="width:100px">${x[1]}</span><span class="fs13 sb6">${x[2]}</span></div>`).join('')}</div>
    <button class="mbtn p mt16" data-go="#/driver/inclist">Go to Incidents</button></div>`; },
});

/* ---------- Messages, history, profile ---------- */
const DTHREADS = {
  admin: { n: 'Transportation Admin', last: 'Route updated for tomorrow', t: '10:24 AM', cat: 'School', unread: true, ic: 'user', c: '#2563eb' },
  disp1: { n: 'Dispatcher', last: 'Please reach Raysan stop by 7:00 AM', t: '09:12 AM', cat: 'Dispatcher', ic: 'user', c: '#64748b' },
  school: { n: 'School Admin', last: 'Student parent waiting at Infocity', t: 'Yesterday', cat: 'School', ic: 'user', c: '#2563eb' },
  alert: { n: 'System Alert', last: 'Heavy traffic on SG Highway', t: '25 Sep', cat: 'Alerts', ic: 'warn', c: '#ef4444' },
  disp2: { n: 'Dispatcher', last: "Good job on today's trip", t: '25 Sep', cat: 'Dispatcher', ic: 'user', c: '#64748b' },
};
dp('messages', {
  title: 'Messages', back: '#/driver/home', nav: 'driver:messages', pad: false,
  render: () => { const t = UI.dmt || 'All';
    return `<div class="mtabs">${['All', 'Dispatcher', 'School', 'Alerts'].map(x => `<button class="mtab ${t === x ? 'on' : ''}" data-act="set" data-k="u:dmt" data-v="${x}">${x}</button>`).join('')}</div><div style="background:#fff">${Object.entries(DTHREADS).filter(([, th]) => t === 'All' || th.cat === t).map(([k, th]) => { const mine = (D().msgs[k] || []); return `<button class="mrow" style="padding:13px 14px" data-go="#/driver/chat/${k}"><span class="mic" style="background:${th.c};color:#fff;border-radius:50%;width:40px;height:40px">${ic(th.ic, 20)}</span><div style="flex:1;min-width:0"><div class="row"><span class="t" style="flex:1">${th.n}</span><span class="fs12 mut">${mine.length ? mine[mine.length - 1][2] : th.t}</span></div><div class="row mt4"><span class="s ellip" style="flex:1;margin:0">${esc(mine.length ? (mine[mine.length - 1][0] === 'out' ? 'You: ' : '') + mine[mine.length - 1][1] : th.last)}</span>${th.unread && !UI['dread_' + k] ? '<span class="dotc" style="background:#ef4444"></span>' : ''}</div></div></button>`; }).join('')}</div>`; },
});
page('driver/chat', {
  pad: false,
  header: r => { const th = DTHREADS[r.sub[0]] || DTHREADS.admin; return `<div class="mh"><button class="mh-b" data-go="#/driver/messages">${ic('arrowl', 22)}</button><span class="mic" style="background:#fff;color:${th.c};border-radius:50%;width:38px;height:38px">${ic(th.ic, 20)}</span><div class="mh-t">${th.n}</div>${mhb('phone', `data-act="call" data-who="${th.n}"`)}</div>`; },
  render: r => { const k = r.sub[0] || 'admin', th = DTHREADS[k]; UI['dread_' + k] = true; const msgs = [['in', th.last + '.', th.t], ...(D().msgs[k] || [])];
    return `<div style="display:flex;flex-direction:column;min-height:100%;background:#fff"><div style="flex:1;padding:14px 12px;display:flex;flex-direction:column;gap:10px">${msgs.map(m => m[0] === 'in' ? `<div><div class="bubble in">${esc(m[1])}</div><div class="fs11 mut mt4">${m[2]}</div></div>` : `<div class="bubble out">${esc(m[1])}<div class="tm">${m[2]} ${ic('checks', 13, 'style="color:#2563eb"')}</div></div>`).join('')}</div>
    <div style="padding:8px 10px 0" class="row wrap">${['Reached stop', 'Running 5 min late', 'All students boarded', 'Need assistance'].map(q => `<button class="chipf" style="margin:0 6px 6px 0" data-act="dquick" data-k="${k}" data-v="${q}">${q}</button>`).join('')}</div>
    <div class="row" style="padding:10px;border-top:1px solid #eef1f6"><input class="minp" style="flex:1;border-radius:22px" placeholder="Type a message..." data-bind="u:dchat" data-enter="dsend" data-k="${k}" value="${esc(UI.dchat || '')}"/><button class="mh-b" style="color:#2563eb" data-act="dsend" data-k="${k}">${ic('send', 20)}</button></div></div>`; },
  mount: () => { const mc = document.querySelector('.mc'); if (mc) mc.scrollTop = mc.scrollHeight; },
});
ACT.dquick = d => { UI.dchat = d.v; ACT.dsend(d); };
ACT.dsend = d => { const v = (UI.dchat || '').trim(); if (!v) return; D().msgs[d.k] = [...(D().msgs[d.k] || []), ['out', v, nowTime()]]; UI.dchat = ''; save(); refresh(); const t = setTimeout(() => { if (CUR.page === 'chat') { D().msgs[d.k].push(['in', 'Noted, thank you.', nowTime()]); save(); refresh(); } }, 1600); TIMERS.push(t); };
const DHIST = [['Thu, 26 Sep 2026', 'R-03', 'East Zone', '07:05 AM - 08:00 AM', 'Completed', '24/24 Students'], ['Wed, 25 Sep 2026', 'R-03', 'East Zone', '07:05 AM - 08:00 AM', 'Completed', '24/24 Students'], ['Tue, 24 Sep 2026', 'R-05', 'Central Zone', '07:10 AM - 08:15 AM', 'Completed', '22/22 Students'], ['Mon, 23 Sep 2026', 'R-02', 'South Zone', '07:05 AM - 08:00 AM', 'Incident', '21/22 Students'], ['Sat, 21 Sep 2026', 'R-03', 'East Zone', '07:05 AM - 08:00 AM', 'Completed', '24/24 Students'], ['Fri, 20 Sep 2026', 'R-03', 'East Zone', '07:05 AM - 08:02 AM', 'Completed', '23/24 Students']];
dp('history', {
  title: 'Trip History', back: '#/driver/home', right: () => `<span style="margin-left:auto">${sel('u:dhm', ['Sep 2026', 'Aug 2026', 'Jul 2026'], UI.dhm || 'Sep 2026').replace('class="sel"', 'class="sel" style="height:32px"')}</span>`,
  render: () => { const live = D().phase === 'completed' ? [] : [[TODAY.replace('Thu', 'Thu'), 'R-03', 'East Zone', '07:05 AM - 08:00 AM', D().phase === 'inprogress' ? 'In Progress' : 'Scheduled', '0/24 Students']];
    const rows = (UI.dhm || 'Sep 2026') === 'Sep 2026' ? [...live, ...DHIST.slice(live.length ? 1 : 0)] : DHIST.slice(2).map(h => [h[0].replace('Sep', UI.dhm.slice(0, 3)), ...h.slice(1)]);
    return `<div class="mcard" style="padding:2px 12px">${rows.map(h => `<button class="mrow" style="padding:12px 0" data-act="dhist" data-d="${h[0]}"><div style="flex:1"><div class="t fs13">${h[0]}</div><div class="b fs14 mt4">${h[1]} &nbsp;${h[2]}</div><div class="s">${h[3]}</div></div><div class="tr">${h[4] === 'Incident' ? `<span class="bd bd-red dot">Incident</span>` : badge(h[4], false, h[4] === 'Completed' ? 'green' : 'orange')}<div class="fs12 mut mt8">${h[5]}</div></div>${ic('right', 16, 'style="color:#94a3b8"')}</button>`).join('')}</div>`; },
});
ACT.dhist = d => openSheet(`<div class="b fs16">${esc(d.d)}</div><div class="fs12 mut">R-03 East Zone | GJ01EF9012</div><div class="mt12">${[['Departure', '07:05 AM'], ['Arrival at school', '08:00 AM'], ['Distance', '28.5 km'], ['Students', '24/24'], ['Inspection', 'Passed'], ['Fuel used', '9 L']].map(x => kvRow(x[0], x[1])).join('')}</div><button class="mbtn p mt12" data-act="sheetclose">Close</button>`);
dp('profile', {
  title: 'My Profile', back: '#/driver/home', nav: 'driver:more',
  render: () => `<div class="mcard row">${av('Imran Shaikh', 70, 'mb')}<div><div class="b fs18">Imran Shaikh</div><div class="fs13 mut">Driver ID: DRV001</div><div class="fs13 mt4">+91 98765 43210</div></div></div>
  <div class="mcard" style="padding:2px 12px">${[[busSide(60), "Today's Vehicle", 'GJ01EF9012', 'Tata Starbus', '#/driver/inspect'], [ic('pin', 26, 'style="color:#ef4444;fill:#ef444422"'), 'Assigned Route', 'R-03 East Zone', '', '#/driver/trip'], [ic('phone', 24, 'style="color:#2563eb"'), 'Emergency Contact', '+91 98765 43210', '', '']].map(x => `<button class="mrow" style="padding:12px 0" ${x[4] ? `data-go="${x[4]}"` : 'data-act="call" data-who="Emergency Contact"'}><span style="width:60px;display:flex;justify-content:center">${x[0]}</span><div style="flex:1"><div class="t fs13">${x[1]}</div><div class="s" style="color:#334155">${x[2]}</div>${x[3] ? `<div class="s">${x[3]}</div>` : ''}</div>${ic('right', 16, 'style="color:#94a3b8"')}</button>`).join('')}</div>
  <div class="mcard" style="padding:2px 12px">${[['history', 'Trip History', '#/driver/history'], ['list', 'My Incidents', '#/driver/inclist'], ['monitor', 'Open Driver Web Portal', '#/portal/trip'], ['refresh', 'Reset Demo Data', 'reset']].map(x => `<button class="mrow" style="padding:12px 0" ${x[2] === 'reset' ? 'data-act="reset"' : `data-go="${x[2]}"`}>${ic(x[0], 20)}<span class="t fs13" style="flex:1">${x[1]}</span>${ic('right', 16, 'style="color:#94a3b8"')}</button>`).join('')}</div>
  <div class="fs11 mut" style="padding:4px 6px">Profile details are managed by the school admin. Contact the transport desk to update them.</div>
  <button class="mbtn rsoft mt12" data-go="#/">${ic('logout', 18)}Logout</button>`,
});

/* ======================= Driver web portal ======================= */
const PSTEPS = ["Today's Trip", 'Assigned Bus', 'Assigned Route', 'Stops', 'Students', 'Inspection', 'Driver Attendance', 'Start Trip', 'Complete Trip'];
const portalStepIdx = () => D().portalCompleted ? 9 : D().portalStarted ? 8 : 6;
function portalHead(title, sub) {
  return `<div class="ph" style="align-items:center"><h1>${title}${sub ? `<span class="ttl-sub">${sub}</span>` : ''}</h1><div class="ph-act"><span class="bd bd-green" style="height:34px;padding:0 14px;font-size:14px">${ic('checkc', 16)}${D().portalCompleted ? 'Trip Completed' : 'On Duty'}</span><span class="fs14 mut" style="margin-left:10px">Employee ID: &nbsp;<b style="color:#334155">DRV001</b></span><span class="tb-sep"></span><b class="fs16">Ramesh Patel</b></div></div>`;
}
function portalKpis() {
  return `<div class="kpis" style="grid-template-columns:repeat(5,1fr)">${[['blue', 'busf', 'BUS', 'GJ01AB1234', 'Tata Starbus'], ['green', 'routes', 'ROUTE', 'R-01 (North Zone)', `12 Stops ${fsep} 38 Students`], ['purple', 'clock', 'DEPARTURE TIME', '07:00 AM', 'On Schedule'], ['orange', 'users', 'STUDENTS', '38', `34 Present ${fsep} 4 Absent`], ['red', 'pin', 'TOTAL DISTANCE', '28.4 km', 'Est. Duration: 1h 35m']].map(k => `<div class="kpi k-${k[0]}" style="grid-template-columns:auto 1fr;padding:16px"><span class="kpi-ic soft" style="width:58px;height:58px;color:${{ blue: '#2563eb', green: '#16a34a', purple: '#7c3aed', orange: '#f59e0b', red: '#e11d48' }[k[0]]}">${ic(k[1], 28)}</span><div><div class="fs12 sb6" style="letter-spacing:.4px;color:#334155">${k[2]}</div><div class="b fs18 mt4 nowrap">${k[3]}</div><div class="fs13 mut mt4 row">${k[4]}</div></div></div>`).join('')}</div>`;
}
function stepper() {
  const cur = portalStepIdx();
  return `<div class="stepper">${PSTEPS.map((s, i) => { const st = i < cur ? 'done' : i === cur ? 'cur' : ''; const icn = i < cur ? 'check' : i === cur ? 'user' : i === 7 ? 'play' : 'flag'; return `${i ? `<span class="step-arrow">${ic('right', 16)}</span>` : ''}<div class="step ${st}" data-act="pstep" data-i="${i}"><span class="c">${ic(icn, 18, i < cur ? 'stroke-width="3"' : '')}</span><span><span class="n">${i + 1}</span>${s}</span></div>`; }).join('')}</div>`;
}
ACT.pstep = d => { const i = Number(d.i), go2 = ['#/portal/trip', '#/portal/route', '#/portal/route', '#/portal/route', '#/portal/students', '#/portal/inspection', '#/portal/attendance', '#/portal/trip', '#/portal/trip'][i]; if (i >= 7) { UI.ptab = 'Stops'; } go(go2); };
page('portal/trip', {
  title: "Today's Trip",
  render: () => {
    const t = UI.ptab || 'Stops', started = D().portalStarted, completed = D().portalCompleted;
    const stops = DATA.portalStops.map((s, i) => ({ s, i, st: completed ? 'Completed' : started ? (i < 4 ? 'Completed' : i === 4 ? 'In Progress' : 'Pending') : (i < 2 ? 'Completed' : 'Pending') }));
    const pg = paginate(stops, (UI.pst || {}).p || 1, 5);
    let tbl;
    if (t === 'Stops') tbl = `<table class="tbl"><thead><tr><th>#</th><th>Stop Name</th><th>Time</th><th>Location</th><th>Students</th><th>Status</th><th>Actions</th></tr></thead><tbody>${pg.rows.map(r => `<tr><td>${r.i + 1}</td><td class="m5">${r.s[0]}</td><td>${r.s[1]}</td><td><span class="row" style="gap:6px">${ic('pin', 15, 'style="color:#2563eb"')}${r.s[2]}</span></td><td>${r.s[3]}</td><td>${badge(r.st)}</td><td><span class="link" data-act="pstopview" data-i="${r.i}">${ic('eye', 15)}View</span></td></tr>`).join('')}</tbody></table>${pager('u:pst.p', pg, stops.length, 'stops', null)}`;
    else if (t === 'Students') { const kids = DATA.students.slice(0, 38).map((s, i) => ({ ...s, pres: UI['pp' + i] ?? (i % 9 !== 3) })); const kp = paginate(kids, (UI.psk || {}).p || 1, 8); tbl = `<table class="tbl"><thead><tr><th>Student</th><th>Admission No.</th><th>Class</th><th>Stop</th><th>Parent</th><th>Attendance</th></tr></thead><tbody>${kp.rows.map((s, j) => { const idx = (kp.p - 1) * 8 + j; return `<tr><td><span class="row">${av(s.name, 28)}<span class="m5">${s.name}</span></span></td><td>${s.adm}</td><td>${s.cls}</td><td>${DATA.portalStops[idx % 11][0]}</td><td>${s.parent}<div class="t2">${s.phone}</div></td><td><button class="bd ${s.pres ? 'bd-green' : 'bd-red'} dot" data-act="set" data-k="u:pp${idx}" data-v="${!s.pres}" style="cursor:pointer">${s.pres ? 'Present' : 'Absent'}</button></td></tr>`; }).join('')}</tbody></table>${pager('u:psk.p', kp, kids.length, 'students', null, 8)}`; }
    else tbl = `<div class="g" style="grid-template-columns:repeat(2,1fr);padding:14px">${DATA.inspectionItems.map(i => `<div class="mini">${tile(i.ic, i.c, i.c + '14', 38, 19, 10)}<div style="flex:1"><div class="mv fs13">${i.t}</div><div class="ml">${i.s}</div></div>${badge('Pass')}</div>`).join('')}</div><div class="fs12 mut" style="padding:0 14px 14px">Inspection completed at 06:52 AM by Ramesh Patel. All 7 checks passed.</div>`;
    return `${portalHead("Today's Trip", 'Thursday, 26 Sep 2026')}${portalKpis()}${stepper()}
    <div class="g rgrid" style="grid-template-columns:minmax(0,1fr) 480px;align-items:start">
      <div class="card"><div class="card-h" style="padding:16px 18px 10px">${ic('busf', 26, 'style="color:#2563eb"')}<span class="card-t fs18">Trip Details</span><button class="btn btn-outline" style="margin-left:auto;color:#2563eb" data-go="#/portal/route">${ic('pin', 16)}View Route on Map</button></div>
        <div class="row" style="padding:10px 18px 16px;border-top:1px solid var(--line);border-bottom:1px solid var(--line);gap:0">${[['Route', 'R-01 (North Zone)'], ['Bus', 'GJ01AB1234', 'Tata Starbus'], ['Start Time', '07:00 AM'], ['Estimated End Time', '08:35 AM'], ['Total Stops', '12'], ['Total Students', '38'], ['Distance', '28.4 km']].map((x, i) => `<div style="flex:${i < 2 ? 1.3 : 1};padding:0 12px;${i ? 'border-left:1px solid var(--line)' : 'padding-left:0'}"><div class="fs12 mut">${x[0]}</div><div class="fs15 m5 mt8">${x[1]}</div>${x[2] ? `<div class="fs12 mut mt4">${x[2]}</div>` : ''}</div>`).join('')}</div>
        <div class="row" style="padding:12px 18px 0;gap:10px">${[['Stops', 'pin', 'Stops (12)'], ['Students', 'users', 'Students (38)'], ['Inspection', 'clip', 'Inspection Checklist']].map(x => `<button class="btn ${t === x[0] ? '' : 'btn-outline'}" style="height:42px;${t === x[0] ? 'color:#2563eb;border:1px solid var(--line);border-bottom:2.5px solid #2563eb;background:#fff' : ''}" data-act="set" data-k="u:ptab" data-v="${x[0]}">${ic(x[1], 17)}${x[2]}</button>`).join('')}</div>
        <div style="padding:10px 18px 4px">${`<div style="border:1px solid var(--line);border-radius:10px;overflow:hidden">${tbl}</div>`}</div></div>
      <div class="col" style="gap:12px">
        <div class="card"><div class="card-h" style="padding:14px 16px">${ic('clock', 26, 'style="color:#2563eb"')}<span class="card-t fs18">Driver Attendance <span class="c-red m5">(Mandatory)</span></span><span style="margin-left:auto">${badge('Completed')}</span></div>
          <div class="card-b"><div class="sbox"><div class="row" style="align-items:flex-start">${av('Ramesh Patel', 74, 'mb').replace('border-radius:50%', 'border-radius:10px')}<div><div class="row"><b class="fs16">Ramesh Patel</b>${badge('On Duty')}</div><div class="fs13 mut mt4">Employee ID: &nbsp;DRV001</div><div class="fs13 mt8 row">${ic('cal', 15)}Checked In: 26 Sep 2026, &nbsp;06:45 AM</div></div></div>
          <div class="mbanner mt12" style="background:#ecf9f0;border:1px solid #bfe5cc">${tile('check', '#fff', '#16a34a', 30, 18, 15)}<div><div class="t">Attendance marked successfully!</div><div class="s">You are marked Present for today's duty.</div></div></div>
          <button class="btn btn-soft btn-block mt12" data-go="#/portal/attendance">${ic('cal', 16)}View Attendance History</button></div></div></div>
        <div class="card" style="${started ? '' : 'border:1.5px solid #9dbcff'}"><div class="card-h" style="padding:14px 16px">${tile('play', '#fff', '#64748b', 30, 14, 15)}<span class="card-t fs16">Start Trip</span><span style="margin-left:auto">${started ? badge('Completed') : badge('Enabled')}</span></div><div class="card-b">
          <div class="mbanner" style="background:#ecf9f0;border:1px solid #bfe5cc">${tile('check', '#fff', '#16a34a', 30, 18, 15)}<div><div class="t fs14">${started ? 'Trip started at ' + (D().portalStartAt || '07:00 AM') : 'Driver attendance completed'}</div><div class="s">${started ? 'Live tracking shared with parents and dispatch.' : 'You can now start the trip.'}</div></div></div>
          <button class="btn btn-primary btn-block btn-lg mt12 ${started ? 'disabled' : ''}" data-act="pstart">${ic('play', 16)}${started ? 'Trip In Progress' : 'Start Trip'}</button></div></div>
        <div class="card" style="${started && !completed ? 'border:1.5px solid #86d3a1' : ''}"><div class="card-h" style="padding:14px 16px">${tile('flag', started ? '#fff' : '#64748b', started ? '#16a34a' : '#e2e8f0', 30, 16, 8)}<span class="card-t fs16 ${started ? '' : 'mut'}">Complete Trip</span><span style="margin-left:auto">${completed ? badge('Completed') : started ? badge('Enabled') : `<span class="bd bd-gray">${ic('lock', 12)}Disabled</span>`}</span></div><div class="card-b">
          <div class="fs12 mut tc row" style="justify-content:center">${completed ? ic('checkc', 14) + 'Trip completed. Great job today.' : started ? ic('info', 14) + 'Complete the trip once all students are dropped at school.' : ic('lock', 14) + 'Complete Trip will be available after you start the trip.'}</div>
          <button class="btn btn-block btn-lg mt12 ${started && !completed ? 'btn-success' : 'disabled'}" style="${started && !completed ? '' : 'background:#d7dde7;color:#7b8798'}" data-act="pcomplete">${ic('flag', 16)}Complete Trip</button></div></div>
      </div></div>`;
  },
});
ACT.pstart = () => { D().portalStarted = true; D().portalStartAt = nowTime(); save(); toast('Trip started. Parents notified.'); refresh(); };
ACT.pcomplete = () => { openModal('Complete Trip?', `<p class="mt12">All 12 stops will be marked completed and parents will be notified that students reached City Plus School.</p><label class="fl">Trip notes (optional)</label><textarea class="inp" placeholder="Anything the transport desk should know?"></textarea>`, `<button class="btn btn-outline" data-act="mclose">Cancel</button><button class="btn btn-success" data-act="pcomplete2">${ic('flag', 16)}Complete Trip</button>`); };
ACT.pcomplete2 = () => { closeModal(); D().portalCompleted = true; save(); toast('Trip completed successfully'); refresh(); };
ACT.pstopview = d => { const s = DATA.portalStops[d.i]; openModal(`Stop ${Number(d.i) + 1}: ${s[0]}`, `<div class="map mt12" style="height:200px">${mapSVG({ w: 520, h: 200, seed: 70 + Number(d.i), routes: [{ pts: [[0, 150], [160, 120], [300, 100], [520, 60]], c: '#2563eb', w: 4 }], markers: [{ t: 'pin', x: 300, y: 100, c: '#ef4444' }, { t: 'call', x: 320, y: 40, lines: [[s[0], 700], [s[2], 500]] }] })}</div><div class="kv mt12"><dt>Scheduled Time</dt><dd>${s[1]}</dd><dt>Coordinates</dt><dd>${s[2]}</dd><dt>Students</dt><dd>${s[3]}</dd><dt>Landmark</dt><dd>Near main gate</dd></div>`, `<button class="btn btn-primary" data-act="mclose">Close</button>`); };
page('portal/dash', {
  title: 'Dashboard',
  render: () => `${portalHead('Dashboard', 'Good morning, Ramesh')}${portalKpis()}<div class="g rgrid" style="grid-template-columns:2fr 1fr">
  <div class="card">${cardH('Live Route', `<span class="row fs12" style="margin-left:8px"><span class="live-dot"></span>Live</span><span class="link" style="margin-left:auto" data-go="#/portal/route">Open Route</span>`)}<div class="card-b"><div class="map" style="height:300px">${mapSVG({ w: 760, h: 300, seed: 81, labels: [[120, 60, 'Sargasan', 14], [600, 70, 'Infocity', 14], [380, 250, 'Gandhinagar', 16, 700]], routes: [{ id: 'pdr', pts: [[60, 240], [180, 200], [300, 160], [420, 150], [560, 110], [700, 60]], c: '#2563eb', w: 5 }], markers: [{ t: 'pin', x: 60, y: 240, c: '#16a34a' }, { t: 'pin', x: 700, y: 60, c: '#ef4444', icon: 'school' }, { t: 'bus', x: 300, y: 160, anim: 'pdr', dur: 45, s: 1.3 }] })}</div></div></div>
  <div class="card">${cardH("Today's Checklist")}<div class="card-b">${[['Attendance marked', true, '#/portal/attendance'], ['Vehicle inspection', true, '#/portal/inspection'], ['Trip started', D().portalStarted, '#/portal/trip'], ['Trip completed', D().portalCompleted, '#/portal/trip']].map(x => `<button class="row" style="width:100%;padding:10px 0;border-bottom:1px solid var(--line-2)" data-go="${x[2]}">${x[1] ? ic('checkc', 20, 'style="color:#16a34a"') : ic('clock', 20, 'style="color:#f59e0b"')}<span class="fs14" style="flex:1;text-align:left">${x[0]}</span>${badge(x[1] ? 'Completed' : 'Pending')}</button>`).join('')}<button class="btn btn-primary btn-block mt12" data-go="#/portal/trip">Go to Today's Trip</button></div></div></div>`,
});
page('portal/route', {
  title: 'Route Details', render: () => `${portalHead('Route Details', 'R-01 North Zone')}<div class="g rgrid" style="grid-template-columns:2fr 1fr"><div class="card"><div class="card-b" style="padding:12px"><div class="map" style="height:520px">${mapSVG({ w: 900, h: 520, seed: 91, labels: [[120, 80, 'Sargasan', 14], [720, 90, 'Infocity', 14], [460, 300, 'Gandhinagar', 18, 700], [200, 460, 'Pethapur', 13]], routes: [{ id: 'prt', pts: DATA.portalStops.map((s, i) => [80 + i * 68, 440 - Math.sin(i / 2) * 120 - i * 18]), c: '#2563eb', w: 5 }], markers: [...DATA.portalStops.map((s, i) => ({ t: 'num', x: 80 + i * 68, y: Math.round(440 - Math.sin(i / 2) * 120 - i * 18), n: i + 1, c: i === 11 ? '#ef4444' : '#2563eb' })), { t: 'bus', x: 80, y: 440, anim: 'prt', dur: 60, s: 1.4 }] })}</div></div></div>
  <div class="card">${cardH('Stops (12)')}<div class="card-b">${DATA.portalStops.map((s, i) => `<div class="row" style="padding:8px 0;border-bottom:1px solid var(--line-2)"><span class="mic sm" style="width:26px;height:26px;border-radius:50%;background:${i === 11 ? '#ef4444' : '#2563eb'};color:#fff;font-size:11px;font-weight:700">${i + 1}</span><div style="flex:1"><div class="m5">${s[0]}</div><div class="fs11 mut">${s[2]}</div></div><span class="fs12 sb6">${s[1]}</span></div>`).join('')}</div></div></div>`,
});
page('portal/students', {
  title: 'My Students', render: () => { const q = (UI.psq || '').toLowerCase(); const list = DATA.students.slice(0, 38).filter(s => !q || s.name.toLowerCase().includes(q)); return `${portalHead('My Students', '38 students on R-01')}<div class="filters">${searchBox('u:psq', 'Search student by name...', UI.psq)}</div><div class="card"><table class="tbl"><thead><tr><th>Student</th><th>Admission No.</th><th>Class</th><th>Stop</th><th>Parent / Guardian</th><th>Today</th></tr></thead><tbody>${list.map((s, i) => `<tr><td><span class="row">${av(s.name, 30)}<span class="m5">${s.name}</span></span></td><td>${s.adm}</td><td>${s.cls}</td><td>${DATA.portalStops[i % 11][0]}</td><td>${s.parent}<div class="t2">${s.phone}</div></td><td>${badge(i % 9 === 3 ? 'Absent' : 'Present')}</td></tr>`).join('') || emptyRow(6)}</tbody></table></div>`; },
});
page('portal/attendance', {
  title: 'Attendance', render: () => `${portalHead('Attendance', 'September 2026')}<div class="kpis" style="grid-template-columns:repeat(4,1fr)">${kpi({ c: 'green', i: 'calcheck', v: '22', l: 'Days Present' })}${kpi({ c: 'orange', i: 'clock', v: '1', l: 'Late Check-in' })}${kpi({ c: 'red', i: 'xc', v: '0', l: 'Absent' })}${kpi({ c: 'blue', i: 'trendu', v: '100%', l: 'Attendance Rate' })}</div><div class="card"><table class="tbl"><thead><tr><th>Date</th><th>Check In</th><th>Check Out</th><th>Method</th><th>Status</th></tr></thead><tbody>${Array.from({ length: 14 }, (_, i) => `<tr><td>${26 - i} Sep 2026</td><td>06:${String(40 + (i * 3) % 18).padStart(2, '0')} AM</td><td>${i === 0 ? '-' : '04:1' + (i % 9) + ' PM'}</td><td>Face + GPS</td><td>${badge(i === 6 ? 'Late' : 'Present', true, i === 6 ? 'orange' : 'green')}</td></tr>`).join('')}</tbody></table></div>`,
});
page('portal/inspection', {
  title: 'Inspection', render: () => `${portalHead('Vehicle Inspection', 'GJ01AB1234 | Pre-Trip')}<div class="g rgrid" style="grid-template-columns:repeat(3,1fr)">${DATA.inspectionItems.map(i => `<div class="card"><div style="height:120px;overflow:hidden;border-radius:12px 12px 0 0">${scene(i.scene).replace('<svg ', '<svg width="100%" height="100%" ')}</div><div class="card-b" style="padding:12px 14px"><div class="row"><b class="fs15" style="flex:1">${i.t}</b>${badge('Pass')}</div><div class="fs12 mut mt4">${i.s}</div><div class="mt8">${(i.checks.length ? i.checks : DATA.otherChecks.map(c => c[0])).slice(0, 4).map(c => `<div class="row fs12" style="padding:3px 0">${ic('checkc', 15, 'style="color:#16a34a"')}${c.split('|')[0]}</div>`).join('')}</div></div></div>`).join('')}</div>`,
});
page('portal/history', {
  title: 'History', render: () => `${portalHead('Trip History')}<div class="card"><table class="tbl"><thead><tr><th>Date</th><th>Route</th><th>Bus</th><th>Time</th><th>Students</th><th>Distance</th><th>Status</th></tr></thead><tbody>${Array.from({ length: 12 }, (_, i) => `<tr><td>${25 - i} Sep 2026</td><td>${rc('R-01')} North Zone</td><td>GJ01AB1234</td><td>07:00 AM - 08:3${i % 6} AM</td><td>${36 + (i % 3)}/38</td><td>28.${i % 9} km</td><td>${badge(i === 4 ? 'Delayed' : 'Completed')}</td></tr>`).join('')}</tbody></table></div>`,
});
page('portal/messages', {
  title: 'Messages', render: () => `${portalHead('Messages')}<div class="card">${[['Transport Coordinator', 'Route R-01 updated for tomorrow. Please check stop 6.', '10:24 AM'], ['Dispatcher', 'Please reach Shanti Nagar by 7:00 AM.', '09:12 AM'], ['City Plus School', 'Two students have early pickup on Friday.', 'Yesterday']].map(m => `<button class="dd-i" style="padding:14px 16px;border-bottom:1px solid var(--line-2);border-radius:0" data-act="openmail" data-who="${m[0]}" data-msg="${m[1]}">${av(m[0], 40)}<span style="flex:1"><div class="row"><b>${m[0]}</b><span class="sp"></span><span class="fs12 mut">${m[2]}</span></div><div class="mut mt4">${m[1]}</div></span></button>`).join('')}</div>`,
});
page('portal/help', {
  title: 'Help & Support', render: () => `${portalHead('Help & Support')}<div class="g rgrid" style="grid-template-columns:repeat(3,1fr)">${[['phone', 'Call Transport Desk', '+91 79 1234 5678'], ['chat', 'Chat with Dispatcher', 'Average reply in 2 min'], ['file', 'Driver Handbook', 'Safety rules and SOPs']].map(x => `<button class="card" style="text-align:left;padding:18px" data-act="toast" data-msg="${x[1]}">${tile(x[0], '#2563eb', '#eaf1ff', 48, 24, 12)}<div class="b fs16 mt12">${x[1]}</div><div class="mut mt4">${x[2]}</div></button>`).join('')}</div>`,
});
page('portal/settings', { title: 'Settings', render: () => portalHead('Settings') });
