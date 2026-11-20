/* Parent mobile app */
const PC = () => DATA.children[S.parent.child] || DATA.children[0];
const PFIRST = () => PC().name.split(' ')[0];
const pRight = (bell = true) => bell ? mhb('bellring', 'data-go="#/parent/notifications"', PARENT_NOTIFS.filter((n, i) => !S.parent.notifRead.includes(i)).length || '') : '';
const menuBtn = () => mhb('menu', 'data-act="pmore"');
const pp = (k, def) => page('parent/' + k, { center: true, ...def });

PARENT_NOTIFS.push(
  { k: 'approaching', t: 'Bus Approaching', s: 'Bus is 5 minutes away from Maple Street (Stop 4)', time: '07:05 AM', ic: 'busf', c: '#f59e0b', bg: '#fff3df', cat: 'Transport', ago: 'now', push: "Aarav's bus is 5 minutes away from Maple Street (Stop 4)." },
  { k: 'started', t: 'Trip Started', s: "Aarav's bus has started from depot (East Zone).", time: '07:00 AM', ic: 'playc', c: '#16a34a', bg: '#e8f8ee', cat: 'Transport', ago: '35m ago', push: "Aarav's bus has started from the depot (East Zone)." },
  { k: 'delay', t: 'Delay', s: 'Bus is running 15 minutes late due to heavy traffic.', time: '08:15 AM', ic: 'clock', c: '#f59e0b', bg: '#fff3df', cat: 'Important', ago: '10m ago', push: 'Bus is running 15 minutes late due to heavy traffic. New ETA: 08:45 AM.' },
  { k: 'route', t: 'Route Change', s: 'Temporary route change. Stop moved to Lake View.', time: '08:30 AM', ic: 'share2', c: '#8b5cf6', bg: '#f1ecff', cat: 'Important', ago: '15m ago', push: 'Temporary route change. Your stop moved to Lake View (Stop 5).' },
  { k: 'completed', t: 'Trip Completed', s: 'Aarav has reached school safely.', time: '09:10 AM', ic: 'checkc', c: '#16a34a', bg: '#e8f8ee', cat: 'Transport', ago: '1h ago', push: 'Aarav has reached Greenwood International School safely at 08:05 AM.' },
  { k: 'emergency', t: 'Emergency Alert', s: 'Minor accident reported. All students are safe.', time: '12:20 PM', ic: 'warn', c: '#ef4444', bg: '#fde8e8', cat: 'Important', ago: 'now', push: 'Minor accident reported. All students are safe. School and driver have been notified.' },
);

/* ---------- shared bits ---------- */
function childBar(opts = {}) {
  const c = PC();
  return `<button class="mcard row" style="width:100%;text-align:left;padding:10px 12px;${opts.flat ? 'border:0;box-shadow:none;border-radius:0;background:#fff' : ''}" data-act="pchild">${av(c.name, opts.size || 50, c.name === 'Ananya Sharma' ? 'kid-f' : 'kid-m')}<div style="flex:1"><div class="b fs16">${c.name}</div><div class="fs12 mut mt4">${c.cls}</div></div>${opts.nochev ? '' : ic('down', 20, 'style="color:#475569"')}</button>`;
}
ACT.pchild = () => openSheet(`<div class="b fs16 mb12">Select Child</div>${DATA.children.map((c, i) => `<button class="mrow" data-act="pchildset" data-i="${i}">${av(c.name, 44, i ? 'kid-f' : 'kid-m')}<div style="flex:1"><div class="t">${c.name}</div><div class="s">${c.cls} | ${c.route}</div></div>${i === S.parent.child ? ic('checkc', 20, 'style="color:#16a34a"') : ''}</button>`).join('')}`);
ACT.pchildset = d => { S.parent.child = Number(d.i); save(); closeSheet(); refresh(); toast(`Showing ${DATA.children[d.i].name}`); };
ACT.pmore = () => openSheet(`<div class="row mb12">${av('Priya Sharma', 46, 'f')}<div><div class="b fs16">Priya Sharma</div><div class="fs12 mut">Parent | +91 98765 43210</div></div></div>
  ${[['users', 'My Children', '#/parent/children', '#2563eb'], ['history', 'Trip History', '#/parent/history', '#7c3aed'], ['clip', 'Pickup / Drop Requests', '#/parent/requests', '#f59e0b'], ['shieldcheck', 'Boarding Status', '#/parent/boarding', '#16a34a'], ['bellring', 'Push Notification Previews', '#/parent/push', '#0ea5e9'], ['siren', 'Emergency', '#/parent/emergency', '#ef4444'], ['mail', 'Messages', '#/parent/messages', '#2563eb'], ['swap', 'Switch Role', '#/', '#64748b']].map(x => `<button class="mrow" data-go="${x[2]}">${tile(x[0], x[3], x[3] + '18', 38, 19, 10)}<div class="t" style="flex:1">${x[1]}</div>${ic('right', 18, 'style="color:#94a3b8"')}</button>`).join('')}
  <button class="mbtn rsoft mt12" data-go="#/">${ic('logout', 18)}Logout</button>`);
function infoRow(icon, color, label, value, extra = '') { return `<div class="row" style="padding:10px 0;border-bottom:1px solid #eef1f6;align-items:flex-start">${ic(icon, 18, `style="color:${color};margin-top:1px"`)}<div style="flex:1"><div class="fs11 mut">${label}</div><div class="sb6 fs13 mt4">${value}</div></div>${extra}</div>`; }
function kvRow(k, v, extra = '') { return `<div class="row" style="padding:9px 0;border-bottom:1px solid #f0f2f6"><span class="mut fs13" style="width:130px">${k}</span><span class="fs13 m5" style="flex:1">${v}</span>${extra}</div>`; }
function photoCard(sceneName, cap1, cap2, h = 150) { return `<div style="position:relative;border-radius:12px;overflow:hidden;height:${h}px">${scene(sceneName).replace('<svg ', '<svg width="100%" height="100%" ')}${cap1 ? `<div style="position:absolute;left:0;right:0;bottom:0;padding:22px 10px 8px;background:linear-gradient(transparent,rgba(0,0,0,.7));color:#fff;font-size:12px;font-weight:600">${cap1}<br>${cap2}</div>` : ''}</div>`; }
function thumb(sceneName, w = 78, h = 58) { return `<span style="display:block;width:${w}px;height:${h}px;border-radius:8px;overflow:hidden;flex:none">${scene(sceneName).replace('<svg ', '<svg width="100%" height="100%" ')}</span>`; }
function pmap(kind, h) {
  const W = 360, H = h;
  if (kind === 'live') return mapSVG({ w: W, h: H, seed: 11, routes: [{ pts: [[-10, 80], [34, 120], [70, 205], [150, 265], [222, 325], [252, 345], [296, 352]], c: '#2563eb', w: 5 }], markers: [{ t: 'ring', x: 34, y: 120, c: '#2563eb', r: 9 }, { t: 'ring', x: 70, y: 205, c: '#2563eb', r: 9 }, { t: 'ring', x: 222, y: 325, c: '#2563eb', r: 8 }, { t: 'ring', x: 252, y: 345, c: '#2563eb', r: 8 }, { t: 'bus', x: 152, y: 266, a: 35, s: 1.4 }, { t: 'pin', x: 296, y: 352, c: '#ef4444', icon: 'medical', s: 1.2 }, { t: 'call', x: 238, y: 372, lines: [['Greenwood', 600, 11.5], ['International School', 600, 11.5]] }] });
  if (kind === 'transit') return mapSVG({ w: W, h: H, seed: 19, routes: [{ pts: [[70, 40], [86, 100], [150, 120], [250, 125], [290, 148]], c: '#2563eb', w: 5 }], markers: [{ t: 'pin', x: 70, y: 46, c: '#2563eb', icon: 'flag' }, { t: 'ring', x: 86, y: 100, c: '#2563eb', r: 8 }, { t: 'ring', x: 250, y: 125, c: '#2563eb', r: 8 }, { t: 'bus', x: 160, y: 120, a: 5, s: 1.3 }, { t: 'pin', x: 294, y: 120, c: '#ef4444', icon: 'flag' }, { t: 'tag', x: 132, y: 72, text: 'BUS-12  45 km/h', bg: '#1463ff', c: '#fff', w: 96 }] });
  if (kind === 'stop') return mapSVG({ w: W, h: H, seed: 23, routes: [{ pts: [[-10, 70], [90, 60], [180, 68], [250, 80], [320, 120]], c: '#2563eb', w: 5 }], markers: [{ t: 'bus', x: 150, y: 64, a: 3, s: 1.2 }, { t: 'pin', x: 200, y: 52, c: '#f97316' }, { t: 'call', x: 222, y: 12, lines: [['Maple Street', 600], ['(Stop 4)', 500]] }, { t: 'ring', x: 250, y: 80, c: '#2563eb', r: 7 }] });
  if (kind === 'approach') return mapSVG({ w: W, h: H, seed: 31, routes: [{ pts: [[0, 100], [80, 108], [130, 140], [190, 160], [240, 190], [300, 250]], c: '#2563eb', w: 5 }], markers: [{ t: 'bus', x: 120, y: 132, a: 30, s: 1.3 }, { t: 'pin', x: 190, y: 120, c: '#2563eb' }, { t: 'ring', x: 190, y: 160, c: '#2563eb', r: 9 }, { t: 'call', x: 212, y: 98, lines: [['Maple Street', 600], ['(Stop 4)', 500]] }] });
  if (kind === 'routechg') return mapSVG({ w: W, h: H, seed: 37, routes: [{ pts: [[40, 140], [110, 120], [160, 80], [230, 70], [300, 50]], c: '#1e40af', w: 4, dash: '7 6' }, { pts: [[40, 140], [110, 130], [200, 140], [280, 110], [310, 60]], c: '#2563eb', w: 4 }], markers: [{ t: 'pin', x: 40, y: 142, c: '#16a34a' }, { t: 'pin', x: 310, y: 58, c: '#ef4444' }, { t: 'ring', x: 200, y: 140, c: '#2563eb', r: 6 }] });
  if (kind === 'trip') return mapSVG({ w: W, h: H, seed: 41, routes: [{ pts: [[40, 60], [120, 70], [200, 80], [280, 95], [300, 70], [320, 40]], c: '#2563eb', w: 5 }], markers: [{ t: 'pin', x: 40, y: 62, c: '#16a34a' }, { t: 'bus', x: 180, y: 78, a: 6, s: 1.4 }, { t: 'pin', x: 320, y: 44, c: '#ef4444' }] });
  if (kind === 'eta') return mapSVG({ w: W, h: H, seed: 47, routes: [{ pts: [[0, 40], [90, 28], [170, 50], [230, 70], [280, 80], [330, 100], [360, 150]], c: '#2563eb', w: 5 }], markers: [{ t: 'ring', x: 90, y: 28, c: '#2563eb', r: 8 }, { t: 'bus', x: 150, y: 46, a: 14, s: 1.3 }, { t: 'tag', x: 60, y: 56, text: 'You are here', bg: '#1463ff', c: '#fff', w: 76 }, { t: 'pin', x: 222, y: 60, c: '#ef4444' }, { t: 'call', x: 236, y: 22, lines: [['Maple Street', 600, 10.5], ['(Next Stop)', 500, 10.5]] }, { t: 'ring', x: 330, y: 100, c: '#2563eb', r: 8 }] });
  if (kind === 'report') return mapSVG({ w: 100, h: 80, seed: 51, routes: [{ pts: [[0, 60], [60, 40], [100, 30]], c: '#93c5fd', w: 6 }], markers: [{ t: 'pin', x: 50, y: 40, c: '#ef4444' }] });
  return mapSVG({ w: W, h: H, seed: 5 });
}
const mapBox = (kind, h, extra = '') => `<div class="map" style="height:${h}px;${extra}">${pmap(kind, h)}</div>`;

/* ---------- Home ---------- */
pp('home', {
  title: 'Home', header: () => mhead('Home', false, pRight(), true).replace('<div class="mh-t">', menuBtn() + '<div class="mh-t">'), nav: 'parentB:home',
  render: () => {
    const c = PC(), conf = S.parent.homeConfirmed;
    return `${childBar()}
    <div class="mcard" style="padding:0;overflow:hidden;margin-top:10px"><div class="row" style="padding:12px"><span class="mic sm" style="background:#16a34a;color:#fff">${ic('bus', 18)}</span><div style="flex:1"><div class="b fs14">On the Way to School</div><div class="fs12 mut">${c.bus} | ${c.busModel}</div></div><span class="bd bd-green dot">Live</span></div>
      <div data-go="#/parent/tracking" style="cursor:pointer">${mapBox('transit', 150, 'border-radius:0')}</div>
      <div class="stat3" style="padding:10px"><div>${ic('clock', 18, 'style="color:#16a34a"')}<div><div class="fs11 mut">ETA School</div><div class="b fs13">08:20 AM</div></div></div><div>${ic('pin', 18, 'style="color:#ef4444"')}<div><div class="fs11 mut">Next Stop</div><div class="b fs13">${c.stop}</div></div></div><div>${ic('gauge', 18, 'style="color:#2563eb"')}<div><div class="fs11 mut">Speed</div><div class="b fs13" id="pspeed">45 km/h</div></div></div></div></div>
    ${conf === null ? `<div class="mbanner mt12" style="background:#fff7e8;border:1px solid #fde3b0"><span class="mic sm" style="background:#f59e0b;color:#fff">${ic('home', 18)}</span><div style="flex:1"><div class="t">Evening drop at 03:42 PM</div><div class="s">Please confirm once ${PFIRST()} reaches home safely.</div><button class="mbtn p sm mt8" data-go="#/parent/confirm">Confirm Arrival</button></div></div>` : conf === true ? `<div class="mbanner mt12" style="background:#ecf9f0;border:1px solid #c9ecd6"><span class="mic sm" style="background:#16a34a;color:#fff">${ic('check', 18)}</span><div><div class="t">${PFIRST()} reached home safely</div><div class="s">Confirmed today. Thank you.</div></div></div>` : ''}
    <div class="mlbl" style="margin-top:16px">Quick Access</div>
    <div class="g" style="grid-template-columns:repeat(3,1fr);gap:8px">${[['pin', 'Live Tracking', '#/parent/tracking', '#2563eb'], ['shieldcheck', 'Boarding Status', '#/parent/boarding', '#16a34a'], ['history', 'Trip History', '#/parent/history', '#7c3aed'], ['clip', 'Requests', '#/parent/requests', '#f59e0b'], ['siren', 'Emergency', '#/parent/emergency', '#ef4444'], ['users', 'My Children', '#/parent/children', '#0ea5e9']].map(q => `<button class="mcard" style="margin:0;display:flex;flex-direction:column;align-items:center;gap:8px;padding:12px 6px" data-go="${q[2]}">${tile(q[0], q[3], q[3] + '18', 40, 20, 11)}<span class="fs12 sb6 tc">${q[1]}</span></button>`).join('')}</div>
    <div class="mlbl" style="margin-top:16px">Today</div>
    <div class="mcard">${[['checkc', '#16a34a', 'Boarded at Maple Street', '07:18 AM'], ['school', '#16a34a', 'Arrived at School', '08:05 AM'], ['bus', '#2563eb', 'Dropped at Maple Street', '03:42 PM'], ['home', conf ? '#16a34a' : '#94a3b8', conf ? 'Reached Home (confirmed)' : 'Reached Home - Pending confirmation', conf ? nowTime() : '-']].map(x => `<div class="row" style="padding:7px 0">${ic(x[0], 18, `style="color:${x[1]}"`)}<span class="fs13" style="flex:1">${x[2]}</span><span class="fs12 mut">${x[3]}</span></div>`).join('')}</div>`;
  },
  mount: () => {
    let sp = 45; every(2000, () => { sp = Math.max(28, Math.min(52, sp + Math.round(Math.random() * 8 - 4))); const el = document.getElementById('pspeed'); if (el) el.textContent = sp + ' km/h'; });
    if (!UI.pushShown) { UI.pushShown = true; const t = setTimeout(() => { if (CUR.role === 'parent') pushNote({ title: 'Bus Approaching', body: "Aarav's bus is 5 minutes away from Maple Street (Stop 4).", icon: 'busf', bg: '#f59e0b', go: '#/parent/notif/approaching' }); }, 3500); TIMERS.push(t); }
  },
});

/* ---------- My Children ---------- */
pp('children', {
  title: 'My Children', header: () => mhead('My Children', false, pRight(), true).replace('<div class="mh-t">', menuBtn() + '<div class="mh-t">'), nav: 'parentA:children',
  render: () => `<div class="mbanner" style="background:#e9f1fd">${tile('users', '#fff', '#2563eb', 52, 26, 12)}<div class="s" style="font-size:13.5px;color:#334155;margin:0">Manage and view information about your children and their school transport details.</div></div>
  ${DATA.children.map((c, i) => `<div class="mcard" style="margin-top:12px;padding:12px">
    <button class="row" style="width:100%;text-align:left" data-act="pchildgo" data-i="${i}">${av(c.name, 76, i ? 'kid-f' : 'kid-m').replace('border-radius:50%', '')}<div style="flex:1"><div class="b fs18">${c.name}</div><div class="fs13 mut mt4">${c.cls}</div><div class="fs13 mut mt4">${c.school}</div></div>${ic('right', 20, 'style="color:#64748b"')}</button>
    <div class="row mt12" style="gap:6px">${[['route', 'Route', c.route, '#2563eb', '#eaf1ff'], ['bus', 'Bus', c.bus, '#f59e0b', '#fff3df'], ['pin', 'Stop', c.stop.replace(' Street', ' Street'), '#ef4444', '#fde8e8']].map(x => `<div class="row" style="flex:1;gap:6px">${tile(x[0], x[3], x[4], 34, 18, 9)}<div><div class="fs12 mut">${x[1]}</div><div class="fs12 sb6">${x[2]}</div></div></div>`).join('')}</div>
    <div class="row mt12" style="background:#ecf9f0;border-radius:8px;padding:8px 10px">${ic('bus', 18, 'style="color:#16a34a"')}<span class="c-green sb6 fs13">Active</span><span class="sp"></span><span class="fs12 sb6">Mon - Fri</span></div></div>`).join('')}
  <button class="mbtn p mt16" data-act="addchild">${ic('plus', 20)}Add Child</button>`,
});
ACT.pchildgo = d => { S.parent.child = Number(d.i); save(); go('#/parent/child'); };
ACT.addchild = () => openSheet(`<div class="b fs16">Add Child</div><div class="fs12 mut mt4">Link another child using the school admission number.</div><div class="mlbl">Admission Number</div><input class="minp" id="addadm" placeholder="e.g. AD202400311"/><div class="mlbl">Date of Birth</div><input class="minp" type="date"/><button class="mbtn p mt16" data-act="addchild2">Send Link Request</button>`);
ACT.addchild2 = () => { const v = (document.getElementById('addadm') || {}).value; if (!v) { toast('Enter the admission number', 'warn'); return; } closeSheet(); toast('Request sent to school for verification'); };

const childTabs = (base, tabs, cur) => `<div class="mtabs" style="margin:0 -12px">${tabs.map(t => `<button class="mtab ${cur === t ? 'on' : ''}" data-go="#/parent/${base}?t=${t}">${t}</button>`).join('')}</div>`;
function childHead(edit = true) { const c = PC(); return `<div class="row" style="padding:4px 0 14px">${av(c.name, 84, S.parent.child ? 'kid-f' : 'kid-m')}<div style="flex:1"><div class="b fs20">${c.name}</div><div class="fs13 mut mt4">${c.cls}</div><div class="fs13 mut mt4">Student ID: ${c.sid}</div></div>${edit ? `<button class="btn btn-soft btn-sm" data-act="toast" data-msg="Edit request sent to school">${ic('pencil', 14)}Edit</button>` : ''}</div>`; }
pp('child', {
  title: 'Child Details', back: '#/parent/children', right: () => mhb('more', 'data-act="pchild"'), nav: 'parentA:children',
  render: r => {
    const c = PC(), t = r.params.t || 'Overview';
    let body = '';
    if (t === 'Overview') body = `<div class="mcard" style="margin-top:12px;padding:4px 14px">${[['cap', 'Class', c.cls.replace(' (', ' (Class ').replace('Class Class', 'Class'), '#7c3aed', '#f1ecff'], ['school', 'School', c.school, '#16a34a', '#e8f8ee'], ['route', 'Route', c.routeName, '#2563eb', '#eaf1ff'], ['bus', 'Bus', `${c.bus} (Tata Starbus)`, '#f59e0b', '#fff3df'], ['pin', 'Stop', c.stopFull, '#ef4444', '#fde8e8']].map(x => `<div class="row" style="padding:12px 0;border-bottom:1px solid #f0f2f6">${tile(x[0], x[3], x[4], 44, 22, 12)}<div><div class="b fs14">${x[1]}</div><div class="fs13 mt4" style="color:#334155">${x[2]}</div></div></div>`).join('')}</div>
      <div class="mbanner mt12" style="background:#eef4fd">${ic('shieldcheck', 40, 'style="color:#16a34a"')}<div style="flex:1"><div class="row"><span class="t">Transport Status</span><span class="sp"></span>${badge('Active')}</div><div class="s">Your child is currently assigned to school transport.</div></div></div>`;
    if (t === 'Transport') return childHead() + childTabs('child', ['Overview', 'Transport', 'Profile', 'Documents'], t) + transportBody();
    if (t === 'Profile') return childHead() + childTabs('child', ['Overview', 'Transport', 'Profile', 'Documents'], t) + profileBody();
    if (t === 'Documents') body = docsBody();
    return childHead() + childTabs('child', ['Overview', 'Transport', 'Profile', 'Documents'], t) + body;
  },
});
function docsBody() { return `<div class="mcard mt12" style="padding:2px 12px">${[['Transport Pass 2026-27', 'PDF | 210 KB'], ['Fee Receipt - Term 1', 'PDF | 145 KB'], ['Medical Declaration', 'PDF | 98 KB'], ['Authorized Pickup Form', 'PDF | 120 KB']].map(d => `<div class="mrow" style="padding:11px 0">${tile('file', '#ef4444', '#fde8e8', 38, 18, 9)}<div style="flex:1"><div class="t fs13">${d[0]}</div><div class="s">${d[1]}</div></div><button class="ibtn blue" data-act="dl" data-name="${d[0]}">${ic('download', 16)}</button></div>`).join('')}</div>`; }
function transportBody() {
  const c = PC();
  return `<div class="mcard mt12"><button class="row" style="width:100%;text-align:left" data-go="#/parent/routestops">${tile('route', '#2563eb', '#eaf1ff', 50, 26, 12)}<div style="flex:1"><div class="fs12 sb6">Route</div><div class="b fs16 mt4">${c.routeName}</div><span class="link mt4">${ic('pin', 14)}View Route Map</span></div>${ic('right', 18)}</button></div>
  <div class="mcard"><button class="row" style="width:100%;text-align:left" data-go="#/parent/busdriver">${tile('bus', '#fff', '#f59e0b', 50, 26, 12)}<div style="flex:1"><div class="fs12 sb6">Bus</div><div class="b fs16 mt4">${c.bus}</div><div class="fs12 mut mt4">${c.busModel}</div></div><span style="width:110px">${busSide(110)}</span></button></div>
  <div class="mcard row">${av('Ramesh Kumar', 58, 'mb')}<div style="flex:1"><div class="fs12 sb6">Driver</div><div class="b fs16 mt4">Ramesh Kumar</div><div class="fs12 mut mt4">License No: GJ01DL1234</div></div><button class="mic" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="Ramesh Kumar" data-phone="+91 98765 43210">${ic('phone', 22)}</button></div>
  <div class="mcard row">${tile('pin', '#fff', '#ef4444', 50, 26, 12)}<div style="flex:1"><div class="fs12 sb6">Stop</div><div class="b fs15 mt4">${c.stopFull}</div><span class="link mt4" data-go="#/parent/eta">${ic('pin', 14)}View on Map</span></div>${thumb('busstop', 80, 64)}</div>
  <div class="mcard"><div class="row">${ic('clock', 20)}<span class="b fs15" style="flex:1">Schedule</span><span class="link" data-act="schedule">View Full Schedule</span></div>
    <div class="row mt12" style="padding:8px 0;border-bottom:1px solid #f0f2f6">${tile('sun', '#f59e0b', '#fff3df', 40, 22, 10)}<div><div class="sb6 fs14">Morning (Pickup)</div><div class="b fs15 mt4">7:15 AM</div><div class="fs12 mut mt4">${c.stopFull}</div></div></div>
    <div class="row" style="padding:8px 0">${tile('sunset', '#f97316', '#fff3df', 40, 22, 10)}<div><div class="sb6 fs14">Afternoon (Drop)</div><div class="b fs15 mt4">3:45 PM</div><div class="fs12 mut mt4">${c.stopFull}</div></div></div></div>`;
}
ACT.schedule = () => openSheet(`<div class="b fs16 mb12">Weekly Schedule</div>${['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday'].map(d => `<div class="row" style="padding:9px 0;border-bottom:1px solid #eef1f6"><span class="sb6" style="width:100px">${d}</span><span class="fs13 mut" style="flex:1">Pickup 7:15 AM</span><span class="fs13 mut">Drop 3:45 PM</span></div>`).join('')}<div class="fs12 mut mt12">Saturday and Sunday: No service</div>`);
pp('transport', { title: 'Transport Details', back: '#/parent/child', nav: 'parentA:children', render: () => transportBody() });
function profileBody() {
  const c = PC();
  const sec = (icon, color, title, rows, edit = true) => `<div class="mcard mt12"><div class="row mb8">${ic(icon, 22, `style="color:${color}"`)}<span class="b fs15" style="flex:1">${title}</span>${edit ? `<button class="btn btn-soft btn-sm" data-act="editsec" data-t="${title}">${ic('pencil', 13)}Edit</button>` : ''}</div>${rows}</div>`;
  return sec('user', '#2563eb', 'Personal Information', [['Full Name', c.name], ['Date of Birth', c.dob], ['Gender', c.gender], ['Class', c.cls], ['Roll Number', c.roll], ['Admission Number', c.adm]].map(x => kvRow(x[0], x[1])).join(''))
    + sec('building', '#2563eb', 'School Information', kvRow('School Name', c.school) + kvRow('School Address', 'Bodakdev, Ahmedabad, Gujarat') + kvRow('Class Teacher', 'Mrs. Priya Desai', `<button class="mic sm" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="Mrs. Priya Desai">${ic('phone', 17)}</button>`), false)
    + sec('heart', '#ef4444', 'Medical Information', kvRow('Blood Group', c.blood) + kvRow('Allergies', c.allergies) + kvRow('Medical Conditions', c.med));
}
ACT.editsec = d => openSheet(`<div class="b fs16">Edit ${esc(d.t)}</div><div class="fs12 mut mt4">Changes are reviewed by the school before they are applied.</div><div class="mlbl">What would you like to update?</div><textarea class="minp" id="editnote" placeholder="Describe the change"></textarea><button class="mbtn p mt16" data-act="editsec2">Submit for Review</button>`);
ACT.editsec2 = () => { closeSheet(); toast('Update request sent to school'); };
pp('profile', {
  title: 'Child Profile', back: '#/parent/child', right: () => mhb('more', 'data-act="pchild"'), nav: 'parentA:children',
  render: r => {
    const t = r.params.t || 'Profile';
    let body = profileBody();
    if (t === 'Transport') body = transportBody();
    if (t === 'Documents') body = docsBody();
    if (t === 'Emergency') body = emergencyInfoBody(true);
    return childHead() + childTabs('profile', ['Profile', 'Transport', 'Documents', 'Emergency'], t) + body;
  },
});

/* ---------- Live tracking ---------- */
pp('tracking', {
  title: 'Live Tracking', back: '#/parent/home', right: () => mhb('refresh', 'data-act="prefresh"'), nav: 'parentB:tracking', pad: false,
  render: r => {
    const t = r.params.t || 'Live Tracking', c = PC();
    const tabs = `<div class="mtabs">${['Live Tracking', 'Route', 'Stops', 'Timeline'].map(x => `<button class="mtab ${t === x ? 'on' : ''}" data-go="#/parent/tracking?t=${encodeURIComponent(x)}">${x}</button>`).join('')}</div>`;
    if (t === 'Route') return childBar({ flat: true }) + tabs + `<div style="padding:12px">${mapBox('trip', 220)}${routeInfoCard()}</div>`;
    if (t === 'Stops' || t === 'Timeline') return childBar({ flat: true }) + tabs + `<div style="padding:12px">${stopsList(t === 'Timeline')}</div>`;
    return `${childBar({ flat: true })}${tabs}<div style="position:relative">${mapBox('live', 470, 'border-radius:0')}
      <div class="map-pop" style="left:14px;top:18px;display:flex;gap:10px;align-items:flex-start">${ic('pin', 26, 'style="color:#16a34a;fill:#16a34a33"')}<div><div class="fs12">Next Stop</div><div class="b fs14">${c.stop}</div><div class="c-green fs13 sb6" id="nextmin">in 5 mins</div></div></div>
      <div class="map-ctl" style="top:20px"><button data-act="toast" data-msg="Map layers: Standard">${ic('layers', 18)}</button><button data-act="toast" data-msg="Centered on bus">${ic('locate', 18)}</button></div>
      <div class="map-pop" style="left:112px;top:186px;background:#1463ff;color:#fff;padding:6px 10px">${ic('busf', 14)} <b>${c.bus}</b><br><span id="pspd">45 km/h</span></div></div>
      <div class="mcard" style="margin:-20px 10px 0;position:relative;border-radius:16px">
        <div class="row">${busSide(78)}<div style="flex:1"><div class="b fs16">${c.bus}</div><div class="fs12 mut">Tata Starbus (52 Seater)</div></div>${badge('On Time', false)}</div>
        <div class="row mt12" style="align-items:flex-start"><div class="row" style="flex:1;align-items:flex-start">${ic('pin', 24, 'style="color:#16a34a"')}<div><div class="fs12 mut">ETA at School</div><div class="b fs15">08:20 AM</div><div class="c-green fs12">(25 mins)</div></div></div><div class="row" style="flex:1;align-items:flex-start;border-left:1px solid #eef1f6;padding-left:10px">${ic('road', 24, 'style="color:#2563eb"')}<div><div class="fs12 mut">Next Stop</div><div class="b fs15">${c.stop}</div><div class="c-green fs12">(5 mins)</div></div></div></div>
        <button class="mbtn p mt12" data-act="pnotify">${ic('bell', 18)}${UI.pnotify ? 'Notifications On' : 'Get Notifications'}</button></div>`;
  },
  mount: () => { let s = 45; every(2000, () => { s = Math.max(30, Math.min(52, s + Math.round(Math.random() * 6 - 3))); const el = document.getElementById('pspd'); if (el) el.textContent = s + ' km/h'; }); },
});
ACT.prefresh = () => { toast('Location updated just now'); refresh(); };
ACT.pnotify = () => { UI.pnotify = !UI.pnotify; toast(UI.pnotify ? 'You will be alerted when the bus is 5 minutes away' : 'Arrival alerts turned off', UI.pnotify ? 'ok' : 'warn'); refresh(); };
function routeInfoCard() { return `<div class="mcard mt12"><div class="row" style="align-items:flex-start">${ic('route', 30, 'style="color:#2563eb"')}<div><div class="sb6 fs14">Route Information</div><div class="b fs16 mt4">R-03 (East Zone)</div><div class="fs13 mut mt4">Greenwood School Route</div></div></div><div class="row mt12">${[['pin', '#16a34a', 'Start Time', '07:10 AM'], ['pin', '#ef4444', 'End Time', '08:20 AM'], ['route', '#64748b', 'Total Distance', '18.5 km']].map(x => `<div class="row" style="flex:1;gap:6px;align-items:flex-start">${ic(x[0], 17, `style="color:${x[1]}"`)}<div><div class="fs11 mut">${x[2]}</div><div class="b fs13">${x[3]}</div></div></div>`).join('')}</div></div>`; }
function stopsList(withChev = true) {
  return `<div class="mcard" style="padding:6px 10px">${DATA.pStops.map((s, i) => {
    const done = i < 2, cur = i === 2, last = i === DATA.pStops.length - 1;
    const [sub, sub2] = s.s.split('|');
    return `<div class="row" style="align-items:flex-start;padding:10px 4px;position:relative;${cur ? 'background:#eef4ff;border-radius:10px;margin:0 -6px;padding:12px 10px' : ''}"><span class="fs13 ${cur ? 'c-blue b' : 'mut'}" style="width:62px;padding-top:2px">${s.t}</span><span style="width:26px;display:flex;flex-direction:column;align-items:center;position:relative">${done ? `<span class="ck">${ic('check', 14, 'stroke-width="3"')}</span>` : last ? ic('pin', 26, 'style="color:#ef4444;fill:#ef4444"') : `<span class="tl-dot ${cur ? 'cur' : ''}"></span>`}${last ? '' : `<span style="position:absolute;top:26px;height:40px;width:2px;background:${done ? '#22c55e' : '#cbd5e1'};${done ? '' : 'background-image:linear-gradient(#cbd5e1 50%,transparent 50%);background-size:2px 6px'}"></span>`}</span>
    <div style="flex:1;padding-left:6px"><div class="b fs14">${s.n}</div><div class="fs12 mt4 ${cur ? 'c-green sb6' : 'mut'}">${cur ? 'Next Stop  •  in 5 mins' : sub}${sub2 ? '<br>' + sub2 : ''}</div></div>${cur && withChev ? ic('right', 18, 'style="color:#64748b;margin-top:8px"') : ''}</div>`;
  }).join('')}</div>`;
}
pp('busdriver', {
  title: 'Bus & Driver Details', back: '#/parent/tracking', right: () => mhb('more', 'data-act="toast" data-msg="Bus registered with Gandhinagar RTO"'), nav: 'parentB:tracking',
  render: () => `<div class="mcard" style="padding:0;overflow:hidden"><div style="position:relative;height:200px;background:linear-gradient(#cfe8f7,#e6f3ea)">${tree(40, 150, 2.6, '#43a047').replace('<g ', '<svg viewBox="0 0 360 200" width="100%" height="200" style="position:absolute;inset:0"><g ').concat(tree(320, 150, 2.4, '#388e3c'), '</svg>')}<div style="position:absolute;left:30px;right:30px;bottom:16px">${busSide(300, 'yellow', 'SCHOOL BUS')}</div><span class="bd bd-green" style="position:absolute;right:10px;top:10px;height:26px;padding:0 12px">Running</span></div>
    <div style="padding:12px 14px"><div class="b fs20">BUS-12</div><div class="fs13 mut mt4">Tata Starbus (52 Seater)</div>
    <div class="row mt12" style="background:#f7f9fc;border-radius:12px;padding:12px 4px">${[['busf', '#2563eb', 'Vehicle No.', 'GJ01AB1234'], ['seat', '#8b5cf6', 'Capacity', '52 Seater'], ['fuel', '#16a34a', 'Fuel', '68%']].map(x => `<div class="col" style="flex:1;align-items:center;gap:4px">${tile(x[0], x[1], x[1] + '1a', 40, 20, 10)}<span class="fs12 mut">${x[2]}</span><span class="b fs13">${x[3]}</span></div>`).join('')}</div></div></div>
    <div class="mcard"><div class="row"><span class="b fs16" style="flex:1">Driver Details</span>${badge('On Trip', false)}</div><div class="row mt12" style="align-items:flex-start">${av('Ramesh Kumar', 72, 'mb').replace('border-radius:50%', 'border-radius:12px')}<div style="flex:1"><div class="b fs15">Ramesh Kumar</div><div class="fs12 mut mt4">Driver ID: DRV001</div><div class="fs12 mut mt4">Mobile: +91 98765 43210</div><div class="fs12 mut mt4">Experience: 5+ Years</div></div><button class="mic" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="Ramesh Kumar" data-phone="+91 98765 43210">${ic('phone', 22)}</button></div></div>
    ${routeInfoCard()}`,
});
pp('routestops', {
  title: 'Route & Stops', back: '#/parent/tracking', nav: 'parentB:tracking',
  render: r => {
    const t = r.params.t || 'list';
    return `${childBar()}<div class="mtabs mt12" style="margin:12px -12px 0">${[['map', 'Map View'], ['list', 'Stops List']].map(x => `<button class="mtab ${t === x[0] ? 'on' : ''}" data-go="#/parent/routestops?t=${x[0]}">${x[1]}</button>`).join('')}</div><div class="mt12">${t === 'map' ? mapBox('live', 380) : stopsList()}</div><button class="mbtn soft mt12" data-go="#/parent/tracking?t=Route">${ic('map', 18)}View Full Route on Map</button>`;
  },
});
pp('eta', {
  title: 'ETA & Next Stop', back: '#/parent/tracking', nav: 'parentB:tracking',
  render: () => `${childBar()}<div class="mcard mt12"><div class="row" style="align-items:flex-start">${ic('timer', 40, 'style="color:#16a34a"')}<div style="flex:1"><div class="sb6 fs13">Estimated Arrival at School</div><div class="b" style="font-size:26px;margin-top:4px">08:20 AM</div></div>${badge('On Time', false)}</div><div class="c-green fs13 mt8" style="padding-left:52px"><span id="etamin">25</span> mins remaining  •  18.5 km away</div></div>
  <div class="mcard"><div class="row">${ic('pin', 30, 'style="color:#2563eb;fill:#2563eb33"')}<div style="flex:1"><div class="fs12 mut">Next Stop</div><div class="b fs15 mt4">Maple Street (Stop 4)</div></div><span class="bd bd-blue" style="height:28px;font-size:13px">in 5 mins</span></div><div class="row mt12"><div style="flex:1"><div class="fs12 mut">Estimated Time</div><div class="b fs15 mt4">07:28 AM</div></div><div style="flex:1"><div class="fs12 mut">Distance</div><div class="b fs15 mt4">2.3 km</div></div></div><div class="mt12">${photoCard('busstop', '', '', 120)}</div><div class="mt8">${mapBox('eta', 120)}</div></div>
  <div class="mcard" style="background:#fdf1f1;border-color:#f7dada"><div class="row">${ic('pin', 28, 'style="color:#94a3b8"')}<div style="flex:1"><div class="fs12">After Next Stop</div><div class="b fs14 mt4">Lake View Society (Stop 5)</div><div class="fs12 mut mt4">Estimated Time: 07:35 AM</div></div><span class="sb6 fs13">in 12 mins</span></div></div>`,
  mount: () => { let m = 25; every(6000, () => { m = Math.max(1, m - 1); const el = document.getElementById('etamin'); if (el) el.textContent = m; }); },
});

/* ---------- Boarding status ---------- */
pp('boarding', {
  title: 'Boarding Status', back: '#/parent/home', right: () => mhb('refresh', 'data-act="prefresh"'), nav: 'parentB:tracking',
  render: () => {
    const conf = S.parent.homeConfirmed, d = UI.bday || 0;
    const days = ['Today, 26 Sep 2026', 'Wed, 25 Sep 2026', 'Tue, 24 Sep 2026'];
    const rows = [
      ['done', 'bus', '#f59e0b', 'Boarded', '07:18 AM', 'Maple Street (Stop 4)', 'boarding', '#/parent/bdetail?k=morning'],
      ['done', 'school', '#16a34a', 'Arrived at School', '08:05 AM', 'Greenwood International School', 'school', '#/parent/bdetail?k=school'],
      [conf ? 'done' : 'cur', 'busf', '#2563eb', 'Dropped', '03:42 PM', 'Maple Street (Stop 4)', 'drop', '#/parent/bdetail?k=evening'],
      [conf ? 'done' : 'pend', 'home', '#475569', 'Reached Home', conf ? 'Confirmed by parent' : '', conf ? '' : '- Pending parent confirmation', '', conf ? '' : '#/parent/confirm'],
    ];
    return `${childBar({ nochev: true })}<div class="row mt12" style="gap:8px"><div class="mcard row" style="flex:1;margin:0;padding:8px"><button class="ibtn" data-act="set" data-k="u:bday" data-v="${Math.min(2, d + 1)}" data-num>${ic('left', 16)}</button><span class="sp tc sb6 fs13">${days[d]}</span><button class="ibtn" data-act="set" data-k="u:bday" data-v="${Math.max(0, d - 1)}" data-num>${ic('right', 16)}</button></div><button class="mcard" style="margin:0;padding:10px" data-go="#/parent/history">${ic('cal', 20)}</button></div>
    <div class="mcard mt12">${rows.map((x, i) => `<div class="row" style="align-items:flex-start;padding:8px 0;position:relative;cursor:${x[7] ? 'pointer' : 'default'}" ${x[7] ? `data-go="${x[7]}"` : ''}><span style="width:26px;display:flex;justify-content:center;position:relative">${x[0] === 'done' ? `<span class="ck">${ic('check', 14, 'stroke-width="3"')}</span>` : `<span class="tl-dot ${x[0] === 'cur' ? 'cur' : ''}"></span>`}${i < 3 ? `<span style="position:absolute;top:26px;height:60px;width:2px;background:${x[0] === 'done' ? '#22c55e' : '#cbd5e1'}"></span>` : ''}</span>${tile(x[1], x[2], x[2] + '1a', 40, 22, 10)}<div style="flex:1"><div class="b fs14">${x[3]}</div><div class="sb6 fs13 mt4">${x[4]}</div><div class="fs12 mut mt4">${x[5]}</div></div>${x[6] ? thumb(x[6], 78, 58) : `<span style="width:78px;height:50px;border-radius:8px;background:#e8eef8;display:flex;align-items:center;justify-content:center;color:#475569">${ic('home', 24)}</span>`}</div>`).join('')}</div>
    <div class="row mt12" style="gap:8px"><div class="mcard row" style="flex:1;margin:0;padding:8px" data-go="#/parent/busdriver">${busSide(64)}<div><div class="fs11 sb6">Bus Details</div><div class="b fs13">BUS-12</div><div class="fs11 mut">Tata Starbus</div></div></div><div class="mcard row" style="flex:1;margin:0;padding:8px">${av('Ramesh Kumar', 40, 'mb')}<div><div class="fs11 sb6">Driver</div><div class="b fs13">Ramesh Kumar</div><div class="fs11 mut">+91 98765 43210</div></div></div></div>
    <div class="mlbl mt16">Detailed views</div><div class="g" style="grid-template-columns:1fr 1fr;gap:8px">${[['Boarding (Morning)', '#/parent/bdetail?k=morning'], ['In-Transit', '#/parent/bdetail?k=transit'], ['Arrived at School', '#/parent/bdetail?k=school'], ['Drop (Evening)', '#/parent/bdetail?k=evening']].map(x => `<button class="mbtn soft sm" data-go="${x[1]}">${x[0]}</button>`).join('')}</div>`;
  },
});
pp('bdetail', {
  title: r => ({ morning: 'Student Boarding', transit: 'Trip in Progress', school: 'Arrival at School', evening: 'Evening Drop - Live' }[r.params.k] || 'Student Boarding'), back: '#/parent/boarding', nav: 'parentB:tracking',
  right: r => r.params.k === 'transit' ? mhb('bellring', 'data-act="pnotify"') : '',
  render: r => {
    const k = r.params.k || 'morning';
    const head = childBar({ nochev: true });
    if (k === 'transit') return `${head}<div class="mcard mt12" style="padding:0;overflow:hidden"><div class="row" style="padding:12px;background:#ecf9f0">${tile('bus', '#fff', '#16a34a', 38, 20, 19)}<span class="b fs15" style="flex:1">On the Way to School</span><span class="bd bd-green dot" style="background:#fff">Live</span></div>${mapBox('transit', 170, 'border-radius:0')}<button class="row" style="padding:12px;width:100%;text-align:left" data-go="#/parent/eta">${ic('pin', 26, 'style="color:#16a34a"')}<div style="flex:1"><div class="fs12 mut">Next Stop</div><div class="b fs14">Riverside Park (Stop 5)</div><div class="fs12 mut">ETA: 07:32 AM</div></div>${ic('right', 18)}</button></div>
      <div class="stat3 mt12">${[['road', '#16a34a', 'Distance', '2.8 km'], ['clock', '#16a34a', 'Estimated Time', '6 mins'], ['users', '#16a34a', 'Students On Board', '12 / 32']].map(x => `<div>${ic(x[0], 22, `style="color:${x[1]}"`)}<div><div class="fs11 mut">${x[2]}</div><div class="b fs13">${x[3]}</div></div></div>`).join('')}</div>`;
    if (k === 'school') return `${head}<div class="mbanner mt12" style="background:#ecf9f0;align-items:center">${tile('school', '#fff', '#16a34a', 44, 22, 22)}<div style="flex:1"><div class="sb6 fs14">Arrived at School</div><div class="b fs18 mt4">08:05 AM</div></div>${badge('On Time', false)}</div>
      <div class="mcard row" style="align-items:flex-start"><div style="flex:1">${infoRow('school', '#2563eb', 'School', 'Greenwood International School')}<div class="row mt12">${ic('users', 24, 'style="color:#16a34a"')}<div><div class="fs12 mut">Total Students Arrived</div><div class="b fs20 mt4">32 / 32</div></div></div></div><div style="width:140px">${photoCard('school', '08:05 AM', 'School Arrival', 160)}</div></div>
      <div class="mbanner" style="background:#ecf9f0;border:1px solid #c9ecd6;margin-top:10px">${tile('check', '#fff', '#16a34a', 36, 20, 18)}<div><div class="t fs14">${PFIRST()} has reached school safely.</div><div class="s">You will be notified again in the evening when the bus starts the return trip.</div></div></div>`;
    const evening = k === 'evening';
    return `${head}<div class="mbanner mt12" style="background:#ecf9f0;align-items:center">${tile('bus', '#fff', '#16a34a', 44, 22, 22)}<div style="flex:1"><div class="sb6 fs14">${evening ? 'Dropped at Stop' : 'Boarded'}</div><div class="b fs16 mt4">${evening ? '03:42 PM' : '07:18 AM'}</div></div>${badge('On Time', false)}</div>
      <div class="mcard row" style="align-items:flex-start;gap:10px"><div style="flex:1">${[['pin', '#ef4444', 'Stop', 'Maple Street (Stop 4)'], ['idcard', '#16a34a', 'Verification', evening ? 'Manual (Driver)' : 'RFID Card'], ['user', '#2563eb', evening ? 'Dropped By' : 'Boarded By', 'Driver (Ramesh Kumar)'], ['bus', '#f59e0b', 'Bus', 'BUS-12 (Tata Starbus)']].map(x => `<div class="row" style="align-items:flex-start;padding:6px 0">${ic(x[0], 18, `style="color:${x[1]}"`)}<div><div class="fs11 mut">${x[2]}</div><div class="fs12 sb6">${x[3]}</div></div></div>`).join('')}</div><div style="width:150px">${photoCard(evening ? 'drop' : 'boarding', evening ? '03:42 PM' : '07:18 AM', evening ? 'Drop Photo' : 'Boarding Photo', 176)}</div></div>
      <div class="mlbl">Stop Location</div>${mapBox('stop', 120)}${evening && S.parent.homeConfirmed === null ? `<button class="mbtn p mt12" data-go="#/parent/confirm">Confirm Arrival at Home</button>` : ''}`;
  },
});
pp('confirm', {
  title: 'Confirm Arrival at Home', back: '#/parent/boarding', right: () => mhb('more', 'data-go="#/parent/waiting"'), nav: 'parentB:tracking',
  render: () => `${childBar({ nochev: true })}<div class="mcard mt12 tc" style="padding:18px 14px"><div style="display:flex;justify-content:center">${houseArt()}</div><div class="b fs16 mt12">Has ${PFIRST()} reached home safely?</div><div class="fs13 mut mt4">Please confirm after the child has reached home.</div>
  <button class="mbtn p mt16" data-act="phome" data-v="yes">${ic('checkc', 20)}Yes, ${PFIRST()} has reached home</button><button class="mbtn o mt8" data-act="phome" data-v="no">Not yet, still in transit</button></div>
  <div class="mbanner" style="background:#eaf1fd;margin-top:10px">${ic('clock', 26, 'style="color:#2563eb"')}<div class="s" style="margin:0;color:#334155">You can confirm anytime within 15 minutes after drop time (till 03:57 PM).</div></div>`,
});
ACT.phome = d => {
  if (d.v === 'yes') { S.parent.homeConfirmed = true; save(); toast(`Thank you. ${PFIRST()}'s arrival is confirmed.`); go('#/parent/boarding'); }
  else { S.parent.confirmStart = Date.now(); save(); go('#/parent/waiting'); }
};
pp('waiting', {
  title: 'Waiting for Parent Confirmation', back: '#/parent/confirm', right: () => mhb('more', 'data-act="pwaitmenu"'), nav: 'parentB:tracking',
  render: () => `${childBar({ nochev: true })}<div class="mcard mt12" style="padding:18px 14px"><div class="row" style="align-items:flex-start">${ic('clock', 36, 'style="color:#475569"')}<div><div class="b fs16">Waiting for confirmation</div><div class="fs13 mut mt4">${PFIRST()} was dropped at 03:42 PM</div></div></div>
  <div style="display:flex;justify-content:center;margin:18px 0" id="ringwrap">${ringTimer(12.5 / 15, 140, '#16a34a', '<div class="b" style="font-size:26px" id="cdtxt">12:30</div><div class="fs12 mut">minutes left</div>')}</div>
  <div class="mbanner" style="background:#ecf9f0;border:1px solid #c9ecd6">${tile('check', '#fff', '#16a34a', 34, 18, 17)}<div class="s" style="margin:0;color:#1f2937">Please confirm once ${PFIRST()} has reached home.<br>You can confirm within 15 minutes after drop time (till 03:57 PM).</div></div>
  <button class="mbtn p mt12" data-act="phome" data-v="yes">${ic('checkc', 20)}Yes, ${PFIRST()} has reached home</button><button class="link mt12" style="width:100%;justify-content:center" data-go="#/parent/notconfirmed">Simulate: buffer time ends without confirmation</button></div>`,
  mount: () => {
    const start = S.parent.confirmStart || Date.now(); let left = Math.max(0, 750 - Math.floor((Date.now() - start) / 1000));
    const upd = () => { const el = document.getElementById('cdtxt'); if (!el) return; const m = Math.floor(left / 60), s = left % 60; el.textContent = `${m}:${String(s).padStart(2, '0')}`; const c = document.querySelector('#ringwrap circle:nth-child(2)'); if (c) { const C = 2 * Math.PI * 63; c.setAttribute('stroke-dasharray', `${(C * left / 900).toFixed(1)} ${C}`); } if (left <= 0) go('#/parent/notconfirmed'); };
    upd(); every(1000, () => { left--; upd(); });
  },
});
ACT.pwaitmenu = () => openSheet(`<button class="mrow" data-go="#/parent/notconfirmed">${ic('alertc', 20)}<span class="t">Preview: confirmation not received</span></button><button class="mrow" data-act="call" data-who="School Transport Desk">${ic('phone', 20)}<span class="t">Call transport desk</span></button>`);
pp('notconfirmed', {
  title: 'Confirmation Not Received', back: '#/parent/waiting', nav: 'parentB:tracking',
  render: () => `${childBar({ nochev: true })}<div class="mbanner mt12" style="background:#fdeeee;border:1px solid #f7cfcf">${ic('alertc', 36, 'style="color:#fff;fill:#ef4444"')}<div><div class="t" style="color:#dc2626">Parent confirmation not received</div><div class="s">It has been 15 minutes since ${PFIRST()} was dropped at 03:42 PM and the home arrival is not yet confirmed.</div></div></div>
  <div class="mlbl mt16">Automatic Alerts Sent</div><div class="mcard" style="padding:4px 12px">${[['school', 'Alert sent to School', 'Call triggered to transport coordinator', 'School Transport Coordinator'], ['user', 'Alert sent to Driver (Ramesh Kumar)', "Call triggered to driver's phone", 'Ramesh Kumar']].map((x, i) => `<div class="mrow" style="padding:11px 0">${i ? av('Ramesh Kumar', 38, 'mb') : tile('school', '#2563eb', '#eaf1ff', 38, 20, 10)}<div style="flex:1"><div class="row"><span class="t fs13" style="flex:1">${x[1]}</span><span class="fs11 mut">03:57 PM</span></div><div class="s">${x[2]}</div></div><button class="mic sm" style="background:#e8f8ee;color:#16a34a" data-act="call" data-who="${x[3]}">${ic('phone', 17)}</button></div>`).join('')}</div>
  <div class="mbanner" style="background:#eaf1fd;margin-top:10px">${ic('clock', 24, 'style="color:#2563eb"')}<div class="s" style="margin:0;color:#334155">The school and driver will contact you to confirm the child's safety. Please respond to their call.</div></div>
  <button class="mbtn p mt12" data-act="phome" data-v="yes">${ic('checkc', 20)}Confirm ${PFIRST()} is home now</button>`,
});

/* ---------- Notifications ---------- */
pp('notifications', {
  title: 'Notifications', header: () => mhead('Notifications', false, mhb('filter', 'data-act="pnfilter"'), true).replace('<div class="mh-t">', menuBtn() + '<div class="mh-t">'), nav: 'parentB:notifications', pad: false,
  render: () => {
    const t = UI.pnt || 'All';
    const list = PARENT_NOTIFS.map((n, i) => ({ ...n, i })).filter(n => t === 'All' || n.cat === t || (t === 'Messages' && false));
    return `<div class="mtabs">${['All', 'Transport', 'Important', 'Messages'].map(x => `<button class="mtab ${t === x ? 'on' : ''}" data-act="set" data-k="u:pnt" data-v="${x}">${x}</button>`).join('')}</div>
    <div style="background:#fff">${list.length ? list.map(n => `<button class="mrow" style="padding:14px" data-act="pnopen" data-i="${n.i}">${tile(n.ic, n.c, n.bg, 44, 24, 12)}<div style="flex:1"><div class="row"><span class="t" style="flex:1">${n.t}</span><span class="fs12 mut">${n.time}</span>${S.parent.notifRead.includes(n.i) ? '<span style="width:8px"></span>' : '<span class="dotc" style="background:#ef4444;width:8px;height:8px;margin-left:8px"></span>'}</div><div class="s" style="font-size:13px">${n.s}</div></div></button>`).join('') : `<div class="tc mut" style="padding:40px 20px">${ic('mail', 30)}<div class="mt8">No messages yet. Chats are under Messages.</div><button class="mbtn soft sm mt12" data-go="#/parent/messages">Open Messages</button></div>`}</div>
    <div style="padding:12px"><button class="mbtn soft sm" data-act="preadall">${ic('checks', 18)}Mark all as read</button><button class="mbtn o sm mt8" data-go="#/parent/push">${ic('phonem', 18)}Preview push notifications</button></div>`;
  },
});
ACT.pnfilter = () => openSheet(`<div class="b fs16 mb12">Notification Settings</div>${[['Bus approaching (5 min)', true], ['Trip started / completed', true], ['Delays and route changes', true], ['Emergency alerts', true], ['School announcements', false]].map((x, i) => `<label class="row" style="padding:10px 0;border-bottom:1px solid #eef1f6"><span class="fs14" style="flex:1">${x[0]}</span><input type="checkbox" ${x[1] ? 'checked' : ''} ${i === 3 ? 'disabled' : ''} style="width:20px;height:20px"/></label>`).join('')}<div class="fs12 mut mt8">Emergency alerts cannot be turned off.</div><button class="mbtn p mt12" data-act="sheetclose">Save</button>`);
ACT.pnopen = d => { const i = Number(d.i); if (!S.parent.notifRead.includes(i)) S.parent.notifRead.push(i); save(); go('#/parent/notif/' + PARENT_NOTIFS[i].k); };
ACT.preadall = () => { S.parent.notifRead = PARENT_NOTIFS.map((n, i) => i); save(); refresh(); toast('All notifications marked as read'); };
page('parent/notif', {
  center: true, back: '#/parent/notifications', nav: 'parentB:notifications',
  title: r => ({ approaching: 'Bus Approaching', delay: 'Delay Notification', route: 'Route Change', started: 'Trip Started', completed: 'Trip Completed', emergency: 'Emergency Alert' }[r.sub[0]] || 'Notification'),
  render: r => {
    const k = r.sub[0], head = childBar();
    const ban = (icon, c, bg, t1, t2) => `<div class="mbanner mt12" style="background:${bg}">${tile(icon, '#fff', c, 44, 24, 22)}<div><div class="t" style="color:${c === '#f59e0b' ? '#b45309' : '#0f172a'}">${t1}</div><div class="s" style="color:${c === '#f59e0b' ? '#b45309' : '#475569'}">${t2}</div></div></div>`;
    const liveBtn = `<button class="mbtn soft mt12" style="justify-content:flex-start;padding:0 14px" data-go="#/parent/tracking">${ic('map', 22)}<span style="flex:1;text-align:left">View Live Location</span>${ic('right', 18)}</button>`;
    if (k === 'approaching') return `${head}${ban('busf', '#f59e0b', '#fff3df', 'Bus is Approaching', 'Arriving in 5 minutes')}<div class="mt12">${mapBox('approach', 180)}</div><div class="mcard mt12 row">${[['clock', '#16a34a', 'ETA', '07:12 AM', '(5 mins)'], ['pin', '#ef4444', 'Distance', '1.2 km', ''], ['gauge', '#334155', 'Current Speed', '28 km/h', '']].map(x => `<div style="flex:1"><div class="fs11 mut">${x[2]}</div><div class="row mt4" style="gap:4px">${ic(x[0], 16, `style="color:${x[1]}"`)}<span class="b fs13">${x[3]}</span></div>${x[4] ? `<div class="c-green fs11">${x[4]}</div>` : ''}</div>`).join('')}</div><div class="mcard row">${ic('pin', 30, 'style="color:#ef4444;fill:#ef444433"')}<div><div class="fs12 mut">Next Stop</div><div class="b fs15">Maple Street (Stop 4)</div><div class="fs12 mut">ETA: 07:12 AM</div></div></div>${liveBtn}`;
    if (k === 'delay') return `${head}${ban('clock', '#f59e0b', '#fff3df', 'Bus is Delayed', '15 minutes late')}<div class="fs13 mt12" style="color:#334155;line-height:1.5;padding:0 4px">Due to heavy traffic on SG Highway, the bus is running approximately 15 minutes late.</div><div class="mbanner mt12" style="background:#fdeeee;align-items:center">${tile('clock', '#fff', '#ef4444', 44, 24, 22)}<div><div class="sb6 fs13">New Estimated Arrival</div><div class="b" style="font-size:26px;color:#dc2626">08:45 AM</div><div class="fs12 mut">(Original: 08:30 AM)</div></div></div><div class="mcard row">${ic('pin', 24, 'style="color:#ef4444"')}<div style="flex:1"><div class="sb6 fs14">Reason</div><div class="fs13 mut">Heavy traffic</div></div>${thumb('traffic', 104, 70)}</div>${liveBtn}<div class="mbanner mt12" style="background:#eef4fd">${ic('info', 20, 'style="color:#2563eb"')}<div class="s" style="margin:0">You will receive another notification when the bus is 5 minutes away from your stop.</div></div>`;
    if (k === 'route') return `${head}${ban('share2', '#8b5cf6', '#f1ecff', 'Route Change', 'Temporary change')}<div class="fs13 mt12" style="color:#334155;padding:0 4px">Due to road closure, the route has been temporarily changed for today.</div><div class="mcard mt12"><div class="row">${ic('pin', 22, 'style="color:#94a3b8"')}<div><div class="fs12 mut">Previous Stop</div><div class="fs13 sb6">Maple Street (Stop 4)</div></div></div><div style="padding:4px 0 4px 6px">${ic('arrowd', 18, 'style="color:#94a3b8"')}</div><div class="row" style="background:#ecf9f0;border-radius:10px;padding:8px">${ic('pin', 22, 'style="color:#16a34a"')}<div><div class="fs12 c-green">New Stop</div><div class="fs14 b c-green">Lake View (Stop 5)</div><div class="fs12 mut">1.8 km away</div></div></div></div>${mapBox('routechg', 150)}<div class="row fs11 mut mt8" style="gap:14px"><span>- - Original Route</span><span style="color:#2563eb">— New Route</span><span>${ic('pin', 12, 'style="color:#ef4444"')} New Stop</span></div><div class="mbanner mt12" style="background:#eef4fd">${ic('info', 20, 'style="color:#2563eb"')}<div class="s" style="margin:0">Please ensure your child is at the new stop 5 minutes before the estimated time.</div></div><button class="mbtn p mt12" data-act="toast" data-msg="Thanks, the school has been informed you saw the change">Acknowledge Change</button>`;
    if (k === 'started') return `${head}${ban('play', '#16a34a', '#ecf9f0', 'Trip Started', 'Bus has left the depot')}<div class="mcard mt12 row" data-go="#/parent/busdriver">${busSide(96)}<div><div class="b fs14">Bus</div><div class="b fs16">BUS-12</div><div class="fs12 mut">Tata Starbus (52 Seater)</div></div></div><div class="mcard row">${av('Ramesh Kumar', 50, 'mb')}<div style="flex:1"><div class="b fs14">Driver</div><div class="b fs15">Ramesh Kumar</div><div class="fs12 mut">+91 98765 43210</div></div><button class="mic sm" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="Ramesh Kumar">${ic('phone', 18)}</button></div><div class="mcard">${[['done', '07:00 AM', 'Trip Started', 'East Zone Depot'], ['', 'Next Stop', 'Sunrise Apartments', 'ETA: 07:18 AM'], ['', '', 'Maple Street (Stop 4)', 'ETA: 07:28 AM'], ['', '', 'Greenwood International School', 'ETA: 08:05 AM']].map(x => `<div class="tl-i ${x[0]}"><span class="tl-dot ${x[0] ? 'done' : ''}" style="${x[0] ? 'background:#fff;border-color:#22c55e' : ''}"></span><div>${x[1] ? `<div class="fs12 ${x[0] ? 'c-green b' : 'sb6'}">${x[1]}</div>` : ''}<div class="sb6 fs14">${x[2]}</div><div class="fs12 mut">${x[3]}</div></div></div>`).join('')}</div><button class="mbtn soft" data-go="#/parent/tracking?t=Route">${ic('map', 20)}View Full Route${ic('right', 18)}</button>`;
    if (k === 'completed') return `${head}${ban('check', '#16a34a', '#ecf9f0', 'Trip Completed', `${PFIRST()} has reached school safely`)}<div class="mt12">${photoCard('school', '', '', 150)}</div><div class="mcard mt12" style="padding:2px 14px">${[['school', 'Arrival Time', '08:05 AM', '26 Sep 2026'], ['timer', 'Total Travel Time', '1 hour 5 minutes', ''], ['route', 'Distance Covered', '18.5 km', '']].map(x => `<div class="row" style="padding:12px 0;border-bottom:1px solid #f0f2f6">${ic(x[0], 26, 'style="color:#2563eb"')}<div><div class="fs12 mut">${x[1]}</div><div class="b fs15">${x[2]}</div>${x[3] ? `<div class="fs12 mut">${x[3]}</div>` : ''}</div></div>`).join('')}</div><div class="mbanner" style="background:#ecf9f0;margin-top:10px">${ic('checkc', 24, 'style="color:#16a34a"')}<div><div class="t fs13">${PFIRST()} has reached school safely.</div><div class="s">Thank you for using our transport service.</div></div></div>`;
    if (k === 'emergency') return `${head}${ban('warn', '#ef4444', '#fdeeee', 'Emergency Alert', 'Minor accident reported')}<div class="mcard mt12"><div class="fs13" style="line-height:1.55">A minor accident was reported near Maple Street at 12:20 PM. <b>All students are safe.</b> The school and driver have been notified and a replacement bus is on the way.</div></div><div class="mcard">${infoRow('pin', '#ef4444', 'Location', 'Maple Street, East Zone')}${infoRow('bus', '#f59e0b', 'Bus', 'BUS-12 (GJ01AB1234)')}${infoRow('clock', '#2563eb', 'Reported at', '12:20 PM, 26 Sep 2026')}</div><button class="mbtn r" data-act="call" data-who="School Emergency Desk">${ic('phone', 18)}Call School</button><button class="mbtn soft mt8" data-go="#/parent/messages">${ic('msg', 18)}Message Transport Team</button>`;
    return head;
  },
});
pp('push', {
  title: 'Push Notifications', back: '#/parent/notifications',
  render: () => `<div class="fs12 mut" style="padding:0 4px 10px">How alerts appear on your lock screen. Tap a card to open it.</div>${PARENT_NOTIFS.map((n, i) => `<div style="position:relative;border-radius:16px;overflow:hidden;height:210px;margin-bottom:12px"><div class="lock" style="padding-top:16px"><div class="tm" style="font-size:48px">9:41</div><div class="dt" style="font-size:14px">Thu, 26 Sep</div></div><button class="push" style="position:absolute;top:96px;animation:none;text-align:left" data-act="pnopen" data-i="${i}"><span class="mic sm" style="background:${n.c};color:#fff">${ic(n.ic, 18)}</span><div style="flex:1;min-width:0"><div class="row" style="font-size:11.5px;color:#475569"><span>School Transport</span><span class="sp"></span><span>${n.ago}</span></div><div class="b fs14 mt4" style="color:#0f172a">${n.t === 'Delay' ? 'Bus Delayed' : n.t === 'Route Change' ? 'Route Changed' : n.t}</div><div class="fs12" style="color:#334155;margin-top:2px">${n.push}</div></div></button></div>`).join('')}<button class="mbtn p" data-act="pushdemo">${ic('bellring', 18)}Send a test push now</button>`,
});
ACT.pushdemo = () => { const n = PARENT_NOTIFS[Math.floor(Math.random() * PARENT_NOTIFS.length)]; pushNote({ title: n.t, body: n.push, icon: n.ic, bg: n.c, go: '#/parent/notif/' + n.k }); };

/* ---------- Trip history & records ---------- */
const TRIPS = [['Tue, 26 Sep 2026', 'School', 'Home', 'sun'], ['Mon, 25 Sep 2026', 'School', 'Home', 'sun'], ['Fri, 22 Sep 2026', 'School', 'Home', 'sun'], ['Thu, 21 Sep 2026', 'Home', 'School', 'moon'], ['Wed, 20 Sep 2026', 'School', 'Home', 'moon']];
const BREC = [['26 Sep', 1, '07:18 AM'], ['25 Sep', 1, '07:16 AM'], ['24 Sep', 1, '07:20 AM'], ['23 Sep', 0], ['22 Sep', 1, '07:15 AM'], ['21 Sep', 1, '07:18 AM'], ['20 Sep', 1, '07:17 AM'], ['19 Sep', 0]];
const AREC = [['26 Sep', 1, '08:05 AM'], ['25 Sep', 1, '08:08 AM'], ['24 Sep', 1, '08:03 AM'], ['23 Sep', 0], ['22 Sep', 1, '08:06 AM'], ['21 Sep', 1, '08:05 AM'], ['20 Sep', 1, '08:07 AM'], ['19 Sep', 0]];
function monthBar(k) { const m = UI[k] || 0, ms = ['September 2026', 'August 2026', 'July 2026']; return `<div class="row" style="gap:8px"><div class="mcard row" style="flex:1;margin:0;padding:8px"><button class="ibtn" data-act="set" data-k="u:${k}" data-v="${Math.min(2, m + 1)}" data-num>${ic('left', 16)}</button><span class="sp tc b fs14">${ms[m]}</span><button class="ibtn" data-act="set" data-k="u:${k}" data-v="${Math.max(0, m - 1)}" data-num>${ic('right', 16)}</button></div><button class="mcard" style="margin:0;padding:10px" data-act="toast" data-msg="Showing ${ms[m]}">${ic('cal', 20)}</button></div>`; }
function recTable(rows, kind) {
  return `<div class="mcard" style="padding:0;overflow:hidden;margin-top:10px"><table class="tbl compact" style="font-size:12px"><thead><tr><th>Date</th><th>Status</th><th>Time</th><th>${kind === 'b' ? 'Stop' : 'School'}</th><th>Photo</th></tr></thead><tbody>${rows.map(r => `<tr><td>${r[0]}</td><td>${r[1] ? `<span class="row c-green sb6" style="gap:4px">${ic('checkc', 15, 'style="fill:#16a34a;color:#fff"')}${kind === 'b' ? 'Boarded' : 'Arrived'}</span>` : kind === 'b' ? `<span class="row c-red sb6" style="gap:4px">${ic('alertc', 15, 'style="fill:#ef4444;color:#fff"')}Not Boarded</span>` : `<span class="mut">${ic('dotfill', 14)} -</span>`}</td><td>${r[2] || '-'}</td><td class="fs11">${r[1] ? (kind === 'b' ? 'Maple Street<br>(Stop 4)' : 'Greenwood<br>International School') : '-'}</td><td>${r[1] ? thumb(kind === 'b' ? 'boarding' : 'school', 46, 36) : `<span style="display:flex;width:46px;height:36px;border-radius:6px;background:#f1f5f9;align-items:center;justify-content:center;color:#94a3b8">${ic('ban', 18)}</span>`}</td></tr>`).join('')}</tbody></table></div>`;
}
pp('history', {
  title: 'Trip History', back: '#/parent/home', right: () => mhb('cal', 'data-act="toast" data-msg="Pick a date range"'), nav: 'parentB:home',
  render: r => {
    const t = r.params.t || 'all';
    const tabs = `<div class="mtabs" style="margin:12px -12px 0">${[['all', 'All Trips'], ['boarding', 'Boarding Records'], ['arrival', 'Arrival Records']].map(x => `<button class="mtab ${t === x[0] ? 'on' : ''}" data-go="#/parent/history?t=${x[0]}">${x[1]}</button>`).join('')}</div>`;
    if (t === 'boarding') return childBar({ nochev: true }) + tabs + `<div class="mt12">${monthBar('bm')}</div>` + recTable(BREC, 'b');
    if (t === 'arrival') return childBar({ nochev: true }) + tabs + `<div class="mt12">${monthBar('am')}</div>` + recTable(AREC, 'a');
    return `${childBar({ nochev: true })}${tabs}<div class="b fs15" style="margin:14px 2px 6px">September 2026</div><div class="mcard" style="padding:2px 12px">${TRIPS.map((x, i) => `<button class="mrow" style="padding:12px 0" data-go="#/parent/tripdetail?i=${i}">${ic(x[3], 24, `style="color:${x[3] === 'sun' ? '#f59e0b' : '#2563eb'};${x[3] === 'moon' ? 'fill:#2563eb' : ''}"`)}<div style="flex:1"><div class="t">${x[0]}</div><div class="s" style="font-size:13px">${x[1]} → ${x[2]}</div></div>${badge('Completed', false)}${ic('right', 18, 'style="color:#64748b"')}</button>`).join('')}</div>`;
  },
});
pp('tripdetail', {
  title: 'Trip Details', back: '#/parent/history', nav: 'parentB:home',
  render: r => {
    const i = Number(r.params.i || 0), dir = UI.tdir || 'sh';
    return `<div class="mcard row" style="padding:8px"><button class="ibtn" data-go="#/parent/tripdetail?i=${Math.min(4, i + 1)}">${ic('left', 16)}</button><span class="sp tc b fs15">${TRIPS[i][0]}</span><button class="ibtn" data-go="#/parent/tripdetail?i=${Math.max(0, i - 1)}">${ic('right', 16)}</button><button class="ibtn" data-act="toast" data-msg="Pick a date">${ic('cal', 16)}</button></div>
    <div class="mbanner mt12" style="background:#ecf9f0">${tile('check', '#fff', '#16a34a', 38, 20, 19)}<div><div class="t">Completed</div><div class="s">Trip completed successfully.</div></div></div>
    <div class="mtabs mt12" style="margin:12px -12px 0">${[['sh', 'School to Home'], ['hs', 'Home to School']].map(x => `<button class="mtab ${dir === x[0] ? 'on' : ''}" data-act="set" data-k="u:tdir" data-v="${x[0]}">${x[1]}</button>`).join('')}</div>
    <div class="mt12">${mapBox('trip', 110)}</div>
    <div class="mcard mt12">${(dir === 'hs' ? [['07:05 AM', 'Trip Started', 'East Zone Depot', ''], ['07:18 AM', 'Boarded', 'Maple Street (Stop 4)', 'boarding'], ['08:05 AM', 'Arrived at School', 'Greenwood International School', 'school']] : [['03:10 PM', 'Trip Started', 'Greenwood International School', ''], ['03:12 PM', 'Boarded', 'Greenwood International School', 'school'], ['03:42 PM', 'Dropped', 'Maple Street (Stop 4)', 'drop']]).map(x => `<div class="row" style="align-items:flex-start;padding:8px 0">${ic('checkc', 26, 'style="color:#16a34a"')}<div style="flex:1"><div class="sb6 fs14">${x[1]}</div><div class="b fs15">${x[0]}</div><div class="fs13 mut">${x[2]}</div></div>${x[3] ? thumb(x[3], 86, 66) : ''}</div>`).join('')}</div>`;
  },
});

/* ---------- Requests ---------- */
const REQ = {
  absent: { t: 'Child Absent Today', menu: 'Child Absent Today', d: 'Inform school that your child will not use the bus today.', ic: 'cal', c: '#ef4444', bg: '#fdeeee', ban: 'Mark Child Absent' },
  temp: { t: 'Temporary Pickup Change', menu: 'Temporary Pickup Change', d: 'Request a temporary change in pickup location for a specific date or date range.', ic: 'pin', c: '#f59e0b', bg: '#fff5e6', ban: 'Temporary Change' },
  auth: { t: 'Authorized Pickup Person', menu: 'Authorized Pickup Person', d: 'Add or manage authorized pickup persons.', ic: 'users', c: '#2563eb', bg: '#eaf1ff' },
  pickup: { t: 'Change Pickup Location', menu: 'Change Pickup Location', d: 'Request a permanent change in pickup location.', ic: 'share2', c: '#8b5cf6', bg: '#f3ecff', ban: 'Change Pickup Location' },
  drop: { t: 'Change Drop Location', menu: 'Change Drop Location', d: 'Request a permanent change in drop location.', ic: 'pin', c: '#16a34a', bg: '#ecf9f0', ban: 'Change Drop Location' },
};
pp('requests', {
  title: 'Pickup / Drop Requests', back: '#/parent/home', nav: 'parentB:home',
  render: () => `${childBar({ nochev: true })}<div class="mcard mt12" style="padding:2px 12px">${Object.entries(REQ).map(([k, q]) => `<button class="mrow" style="padding:12px 0" data-go="#/parent/req/${k}">${tile(q.ic, q.c, q.bg, 44, 24, 12)}<div style="flex:1"><div class="t">${q.menu}</div><div class="s">${q.d}</div></div>${ic('right', 18, 'style="color:#64748b"')}</button>`).join('')}</div>
  ${S.parent.requests.length ? `<div class="mlbl mt16">My Requests</div><div class="mcard" style="padding:2px 12px">${S.parent.requests.map(q => `<div class="mrow" style="padding:11px 0">${tile(REQ[q.k].ic, REQ[q.k].c, REQ[q.k].bg, 36, 18, 9)}<div style="flex:1"><div class="t fs13">${esc(q.t)}</div><div class="s">${esc(q.summary)} | ${q.when}</div></div>${badge(q.status)}</div>`).join('')}</div>` : ''}`,
});
page('parent/req', {
  center: true, back: '#/parent/requests', nav: 'parentB:home', title: r => REQ[r.sub[0]] ? REQ[r.sub[0]].t : 'Request',
  render: r => {
    const k = r.sub[0], q = REQ[k]; if (!q) return '';
    const head = childBar({ nochev: true });
    if (k === 'auth') return `${head}<div class="mcard mt12" style="padding:2px 12px">${S.parent.authorized.map((p, i) => `<div class="mrow" style="padding:12px 0">${av(p.name, 56, p.rel === 'Grandmother' ? 'fo' : p.rel === 'Mother' ? 'f' : 'mb')}<div style="flex:1"><div class="t">${esc(p.name)}</div><div class="s">${esc(p.rel)}</div><div class="s">${esc(p.phone)}</div></div>${p.primary ? `<span class="bd bd-green dot" style="align-self:flex-start">Primary</span>` : ''}<button class="mh-b" style="color:#475569" data-act="authmenu" data-i="${i}">${ic('more', 20)}</button></div>`).join('')}</div><button class="mbtn o mt12" data-act="authadd">${ic('plus', 18)}Add Authorized Person</button>`;
    const loc = (label, def, opts) => `<div class="mlbl">${label}</div><div class="msel">${ic('pin', 17, 'style="position:absolute;left:12px;right:auto;color:' + (k === 'drop' ? '#16a34a' : '#ef4444') + '"')}<select name="loc" style="padding-left:36px">${opts.map(o => `<option ${o === def ? 'selected' : ''}>${o}</option>`).join('')}</select>${ic('down', 16)}</div>`;
    const stops = ['Riverside Park (Stop 5)', 'Lake View (Stop 5)', 'Sunrise Apartments (Stop 8)', 'City Mall (Stop 6)', 'Maple Street (Stop 4)'];
    const reasonSel = (opts) => `<div class="mlbl">Reason${k === 'absent' ? ' (Optional)' : ''}</div><div class="msel"><select name="reason">${opts.map(o => `<option>${o}</option>`).join('')}</select>${ic('down', 16)}</div>`;
    const dateIn = (label, v) => `<div class="mlbl">${label}</div><input class="minp" type="date" name="date" value="${v}"/>`;
    let fields = '';
    if (k === 'absent') fields = dateIn('Date', '2026-09-26') + reasonSel(['Sick', 'Family function', 'Travelling', 'Other']) + `<div class="mlbl">Additional Notes (Optional)</div><textarea class="minp" name="notes">${PFIRST()} is unwell today.</textarea>`;
    if (k === 'temp') fields = dateIn('Date(s)', '2026-09-26') + loc('New Pickup Location', 'Riverside Park (Stop 5)', stops) + reasonSel(['Special class', 'Staying with relatives', 'Other']) + `<div class="mlbl">Additional Notes (Optional)</div><textarea class="minp" name="notes">Please pick up from Riverside Park today.</textarea>`;
    if (k === 'pickup') fields = loc('New Pickup Location', 'Lake View (Stop 5)', stops) + dateIn('Effective From', '2026-10-02') + reasonSel(['Change of residence', 'Closer stop', 'Other']) + `<div class="mlbl">Additional Notes (Optional)</div><textarea class="minp" name="notes">We have moved to a new address.</textarea>`;
    if (k === 'drop') fields = loc('New Drop Location', 'Sunrise Apartments (Stop 8)', stops) + dateIn('Effective From', '2026-10-02') + reasonSel(['After school activity', 'Change of residence', 'Other']) + `<div class="mlbl">Additional Notes (Optional)</div><textarea class="minp" name="notes">Please drop at Sunrise Apartments.</textarea>`;
    return `${head}<div class="mbanner mt12" style="background:${q.bg}">${tile(q.ic, '#fff', q.c, 42, 22, 21)}<div><div class="t" style="color:${q.c}">${q.ban}</div><div class="s">${q.d}</div></div></div><div id="reqform">${fields}</div><button class="mbtn p mt16" data-act="reqsubmit" data-k="${k}">Submit Request</button>`;
  },
});
ACT.reqsubmit = d => {
  const f = formVals(document.getElementById('reqform')), q = REQ[d.k];
  const summary = d.k === 'absent' ? `${f.date || 'Today'} | ${f.reason}` : `${f.loc || ''}${f.date ? ' from ' + f.date : ''}`;
  S.parent.requests.unshift({ k: d.k, t: q.t, summary, when: 'Just now', status: d.k === 'absent' ? 'Approved' : 'Pending' });
  save(); toast(d.k === 'absent' ? 'Driver and school informed. Marked absent.' : 'Request submitted for approval'); go('#/parent/requests');
};
ACT.authmenu = d => { const p = S.parent.authorized[d.i]; openSheet(`<div class="b fs16 mb8">${esc(p.name)}</div><button class="mrow" data-act="call" data-who="${esc(p.name)}" data-phone="${esc(p.phone)}">${ic('phone', 20)}<span class="t">Call</span></button>${p.primary ? '' : `<button class="mrow" data-act="authprimary" data-i="${d.i}">${ic('star', 20)}<span class="t">Make Primary</span></button><button class="mrow" data-act="authdel" data-i="${d.i}" style="color:#dc2626">${ic('trash', 20)}<span class="t">Remove</span></button>`}`); };
ACT.authprimary = d => { S.parent.authorized.forEach((p, i) => p.primary = i === Number(d.i)); save(); closeSheet(); refresh(); toast('Primary contact updated'); };
ACT.authdel = d => { S.parent.authorized.splice(Number(d.i), 1); save(); closeSheet(); refresh(); toast('Authorized person removed'); };
ACT.authadd = () => openSheet(`<div class="b fs16">Add Authorized Person</div><div id="authf"><div class="mlbl">Full Name</div><input class="minp" name="name" placeholder="Full name"/><div class="mlbl">Relationship</div><div class="msel"><select name="rel"><option>Uncle</option><option>Aunt</option><option>Grandfather</option><option>Grandmother</option><option>Neighbour</option></select>${ic('down', 16)}</div><div class="mlbl">Mobile Number</div><input class="minp" name="phone" placeholder="+91"/></div><button class="mbtn p mt16" data-act="authadd2">Add Person</button>`);
ACT.authadd2 = () => { const f = formVals(document.getElementById('authf')); if (!f.name || !f.phone) { toast('Name and mobile number are required', 'warn'); return; } S.parent.authorized.push({ name: f.name, rel: f.rel, phone: f.phone }); save(); closeSheet(); refresh(); toast('Authorized person added. The driver will verify ID at pickup.'); };

/* ---------- Emergency ---------- */
pp('emergency', {
  title: 'Emergency', header: () => mhead('Emergency', false, '', true).replace('<div class="mh-t">', menuBtn() + '<div class="mh-t">'), nav: 'parentB:home',
  render: () => `<div class="mbanner" style="background:#fdeeee;align-items:center"><button class="btn btn-danger" style="height:44px;font-size:20px;font-weight:800;padding:0 16px" data-act="sos">SOS</button><div class="s" style="margin:0;color:#dc2626">In case of immediate danger, tap SOS to alert school and transport team.</div></div>
  ${[['phone', '#16a34a', 'Contact School', 'Call or message school for immediate help.', '#/parent/contactschool', '#fff'], ['info', '#2563eb', 'Emergency Information', "View important contacts and your child's details.", '#/parent/emerinfo', '#eef4fd'], ['warn', '#ef4444', 'Report Issue', 'Report accident, breakdown, medical issue or any other problem.', '#/parent/report', '#fdeeee']].map(x => `<button class="mcard row" style="width:100%;text-align:left;margin-top:10px;background:${x[5]}" data-go="${x[4]}"><span class="mic" style="background:${x[1]};color:#fff;border-radius:50%;width:46px;height:46px">${ic(x[0], 22)}</span><div style="flex:1"><div class="b fs15">${x[2]}</div><div class="fs12 mut mt4">${x[3]}</div></div>${ic('right', 18)}</button>`).join('')}
  <div class="mcard mt12" style="background:#fdeeee;border-color:#f8d4d4"><div class="row" style="align-items:flex-start">${ic('siren', 34, 'style="color:#ef4444"')}<div><div class="b fs15 c-red">Emergency Alert</div><div class="fs12 mt4" style="color:#475569">Your child's safety is our priority. This will notify the School and Transport Team immediately.</div></div></div><button class="mbtn r mt12" data-act="sos">${ic('warn', 18)}SOS - Emergency Alert</button></div>
  <div class="b fs15" style="margin:14px 2px 8px">Quick Actions</div><div class="g" style="grid-template-columns:repeat(3,1fr);gap:8px">${[['phone', '#16a34a', 'Call School', 'School Reception'], ['users', '#8b5cf6', 'Call Driver', 'Ramesh Kumar'], ['help', '#f97316', 'Call Transport Team', 'Transport Coordinator']].map(x => `<button class="mcard" style="margin:0;display:flex;flex-direction:column;align-items:center;gap:6px;padding:10px 4px" data-act="call" data-who="${x[3]}"><span class="mic sm" style="background:${x[1]};color:#fff;border-radius:50%">${ic(x[0], 18)}</span><span class="fs12 sb6 tc">${x[2]}</span></button>`).join('')}</div>`,
});
ACT.sos = () => openSheet(`<div class="tc"><span class="mic" style="background:#fde8e8;color:#ef4444;margin:6px auto;width:64px;height:64px;border-radius:50%">${ic('siren', 32)}</span><div class="b fs18 mt8">Send SOS Alert?</div><div class="fs13 mut mt4">School, transport team and the driver of BUS-12 will be alerted with your location.</div></div><button class="mbtn r mt16" data-act="sos2">${ic('warn', 18)}Send SOS Now</button><button class="mbtn o mt8" data-act="sheetclose">Cancel</button>`);
ACT.sos2 = () => { closeSheet(); toast('SOS sent. The school is calling you now.', 'warn'); };
pp('contactschool', {
  title: 'Contact School', back: '#/parent/emergency', nav: 'parentB:home',
  render: () => `<div class="mcard" style="padding:0;overflow:hidden">${photoCard('school', '', '', 150).replace('border-radius:12px', 'border-radius:0')}<div style="padding:12px"><div class="b fs16">Greenwood International School</div><div class="row mt8">${badge('Open Now')}<span class="fs12 mut">08:00 AM - 05:00 PM</span></div></div></div>
  ${[['phone', '#16a34a', 'Call School', '+91 79 1234 5678', 'call'], ['chat', '#2563eb', 'Message School', '', 'msg'], ['mail', '#ef4444', 'Email School', 'admin@greenwoodschool.edu.in', 'mail']].map(x => `<button class="mcard row" style="width:100%;text-align:left;margin-top:10px${x[4] === 'msg' ? ';background:#eef4fd' : ''}" ${x[4] === 'msg' ? 'data-go="#/parent/chat/school"' : x[4] === 'call' ? 'data-act="call" data-who="Greenwood International School"' : 'data-act="toast" data-msg="Opening mail app"'}><span class="mic" style="background:${x[1]};color:#fff;border-radius:50%">${ic(x[0], 22)}</span><div><div class="b fs15 ${x[4] === 'msg' ? 'c-blue' : ''}">${x[2]}</div>${x[3] ? `<div class="fs12 mut mt4">${x[3]}</div>` : ''}</div></button>`).join('')}
  <div class="b fs15" style="margin:14px 2px 8px">Key Contacts</div><div class="mcard" style="padding:2px 12px">${[['School Reception', '+91 79 1234 5678', 'headset', '#f59e0b'], ['Transport Coordinator', '+91 79 98765 4321', 'user', '#2563eb'], ['Principal', '+91 79 87654 3210', 'user', '#8b5cf6']].map(x => `<div class="mrow" style="padding:11px 0"><span class="mic sm" style="background:${x[3]};color:#fff;border-radius:50%">${ic(x[2] === 'headset' ? 'help' : 'user', 17)}</span><div style="flex:1"><div class="t fs13">${x[0]}</div><div class="s">${x[1]}</div></div><button class="mic sm" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="${x[0]}" data-phone="${x[1]}">${ic('phone', 17)}</button></div>`).join('')}</div>`,
});
function emergencyInfoBody(inProfile) {
  const c = PC();
  const contact = (n, rel, ph, k) => `<div class="mrow" style="padding:10px 0">${av(n, 40, k)}<div style="flex:1"><div class="t fs13">${n} <span class="mut m5">${rel}</span></div><div class="s">${ph}</div></div><button class="mic sm" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="${n}" data-phone="${ph}">${ic('phone', 17)}</button></div>`;
  return `${inProfile ? '' : `<div class="row" style="padding:4px 0 12px">${av(c.name, 64)}<div><div class="b fs16">${c.name}</div><div class="fs12 mut">${c.cls}</div><div class="fs12 mut">Student ID: ${c.sid}</div></div></div>`}
  <div class="mcard ${inProfile ? 'mt12' : ''}" style="background:#fdeeee;border-color:#f8d4d4">${[['heart', 'Blood Group', c.blood], ['warn', 'Allergies', c.allergies], ['', 'Medical Conditions', c.med]].map(x => `<div class="row" style="padding:5px 0">${x[0] ? ic(x[0], 18, 'style="color:#ef4444;fill:#ef4444"') : '<span style="width:18px"></span>'}<span class="fs13" style="flex:1">${x[1]}</span><span class="fs13 sb6 c-red">${x[2]}</span></div>`).join('')}</div>
  <div class="b fs15" style="margin:12px 2px 4px">Emergency Contacts</div><div class="mcard" style="padding:2px 12px">${contact('Priya Sharma', '(Mother)', '+91 98765 43210', 'f')}${contact('Rohit Sharma', '(Father)', '+91 98765 43211', 'mb')}${contact('Neha Patel', '(Grandmother)', '+91 98765 43212', 'fo')}</div>
  <div class="b fs15" style="margin:12px 2px 4px">School & Transport Contacts</div><div class="mcard" style="padding:2px 12px">${[['School Reception', '+91 79 1234 5678', '#f59e0b'], ['Transport Coordinator', '+91 79 98765 4321', '#2563eb']].map(x => `<div class="mrow" style="padding:10px 0"><span class="mic sm" style="background:${x[2]};color:#fff;border-radius:50%">${ic('user', 17)}</span><div style="flex:1"><div class="t fs13">${x[0]}</div><div class="s">${x[1]}</div></div><button class="mic sm" style="background:#eaf1ff;color:#2563eb" data-act="call" data-who="${x[0]}">${ic('phone', 17)}</button></div>`).join('')}${contact('Driver (Ramesh Kumar)', '', '+91 98765 43210', 'mb')}</div>`;
}
pp('emerinfo', { title: 'Emergency Information', back: '#/parent/emergency', nav: 'parentB:home', render: () => emergencyInfoBody(false) });
pp('report', {
  title: 'Report Issue', back: '#/parent/emergency', nav: 'parentB:home',
  render: () => {
    const t = UI.prt || 'Accident', len = (UI.prd ?? 'Minor accident near Maple Street. All students are safe. Bus is stopped.').length;
    const types = [['crash', 'Accident', '#ef4444'], ['tools', 'Breakdown', '#f59e0b'], ['user', 'Student Issue', '#2563eb'], ['route', 'Route Issue', '#2563eb'], ['medical', 'Medical Issue', '#ef4444'], ['moreh', 'Other', '#475569']];
    const photos = UI.prp ?? ['boarding', 'traffic', 'lights'];
    return `<div class="g" style="grid-template-columns:repeat(3,1fr);gap:8px">${types.map(x => `<button class="mcard" style="margin:0;display:flex;flex-direction:column;align-items:center;gap:6px;padding:12px 4px;${t === x[1] ? 'border:1.5px solid #ef4444;background:#fff5f5' : ''}" data-act="set" data-k="u:prt" data-v="${x[1]}">${x[1] === 'Medical Issue' ? `<span class="mic sm" style="background:#ef4444;color:#fff;border-radius:50%">${ic('plus', 20, 'stroke-width="3"')}</span>` : ic(x[0], 30, `style="color:${x[2]}"`)}<span class="fs12 sb6 ${t === x[1] ? 'c-red' : ''}">${x[1]}</span></button>`).join('')}</div>
    <div class="b fs15" style="margin:14px 2px 0">Issue Details</div><div class="mlbl">Issue Type</div><div class="msel"><select data-bind="u:prt">${types.map(x => `<option ${t === x[1] ? 'selected' : ''}>${x[1]}</option>`).join('')}</select>${ic('down', 16)}</div>
    <div class="mlbl">Description</div><textarea class="minp" maxlength="500" data-bind="u:prd" data-live>${esc(UI.prd ?? 'Minor accident near Maple Street. All students are safe. Bus is stopped.')}</textarea><div class="tr fs11 mut mt4">${len}/500</div>
    <div class="mlbl">Location (Auto-filled)</div><div class="row" style="gap:8px"><div class="minp row" style="flex:1">${ic('pin', 17, 'style="color:#ef4444"')}<span class="fs13">Maple Street, East Zone</span></div><div class="map" style="width:96px;height:60px">${pmap('report', 80)}</div></div>
    <div class="row"><div class="mlbl" style="flex:1">Add Photos (Optional)</div><span class="fs12 mut">${photos.length}/5</span></div><div class="mph">${photos.map((p, i) => `<div class="p">${scene(p).replace('<svg ', '<svg width="100%" height="100%" ')}<button class="x" data-act="prpdel" data-i="${i}">${ic('x', 11)}</button></div>`).join('')}${photos.length < 5 ? `<button class="add" data-act="prpadd">${ic('plus', 22)}</button>` : ''}</div>
    <button class="mbtn p mt16" data-act="prsubmit">Submit Report</button>`;
  },
});
ACT.prpadd = () => { const p = UI.prp ?? ['boarding', 'traffic', 'lights']; UI.prp = [...p, ['school', 'busstop', 'interior'][p.length % 3]]; toast('Photo added'); refresh(); };
ACT.prpdel = d => { const p = [...(UI.prp ?? ['boarding', 'traffic', 'lights'])]; p.splice(Number(d.i), 1); UI.prp = p; refresh(); };
ACT.prsubmit = () => { S.parent.reported.unshift({ t: UI.prt || 'Accident', d: UI.prd, at: nowTime() }); save(); UI.prd = undefined; UI.prp = undefined; toast('Report sent to school and transport team'); go('#/parent/emergency'); };

/* ---------- Messages ---------- */
const PTHREADS = {
  school: { n: 'School Admin', sub: 'Greenwood International School', ic: 'building', c: '#2563eb', last: 'Bus will be 10 minutes late today...', t: '09:15 AM', unread: 2, msgs: [['in', 'Bus will be 10 minutes late today due to traffic on SG Highway.', '09:15 AM'], ['out', 'Thank you for the update.', '09:16 AM']] },
  transport: { n: 'Transport Team', sub: 'Transport Department', ic: 'bus', c: '#f59e0b', last: 'Route change for East Zone...', t: '08:30 AM', unread: 1, msgs: [['in', 'Route change for East Zone today. Stop moved to Lake View.', '08:30 AM'], ['out', 'Noted. Thank you.', '08:31 AM']] },
  driver: { n: 'Driver (Ramesh Kumar)', title: 'Ramesh Kumar', sub: 'Driver • BUS-12', person: 'Ramesh Kumar', last: 'Bus has left the depot.', t: '07:05 AM', msgs: [['in', 'Bus has left the depot. ETA to Maple Street: 5 mins.', '07:05 AM'], ['out', 'Okay, thank you.', '07:06 AM']] },
  teacher: { n: 'Class Teacher', person: 'Mrs. Priya Desai', sub: 'Grade 5 (Class V)', last: 'Aarav was present today.', t: 'Yesterday', msgs: [['in', 'Aarav was present today.', 'Yesterday']] },
  coord: { n: 'Transport Coordinator', ic: 'user', c: '#2563eb', sub: 'Transport Department', last: 'Pickup location updated.', t: 'Yesterday', msgs: [['in', 'Pickup location updated.', 'Yesterday']] },
  principal: { n: 'School Principal', person: 'Rohit Sharma', sub: 'Greenwood International School', last: 'Diwali holiday schedule...', t: '24 Sep', msgs: [['in', 'Diwali holiday schedule: School remains closed from 20 Oct to 25 Oct.', '24 Sep']] },
  general: { n: 'General Announcements', ic: 'megaphone', c: '#f59e0b', sub: 'Greenwood International School', last: 'School will remain closed on...', t: '22 Sep', msgs: [['in', 'School will remain closed on 2 Oct for Gandhi Jayanti.', '22 Sep']] },
};
const threadMsgs = k => [...PTHREADS[k].msgs, ...((S.parent.msgs || {})[k] || [])];
pp('messages', {
  title: 'Messages', header: () => mhead('Messages', false, '', true).replace('<div class="mh-t">', menuBtn() + '<div class="mh-t">'), nav: 'parentB:messages', pad: false,
  render: () => {
    const t = UI.pmt || 'All';
    const keys = Object.keys(PTHREADS).filter(k => t === 'All' || (t === 'School' ? ['school', 'teacher', 'principal', 'general'].includes(k) : ['transport', 'driver', 'coord'].includes(k)));
    return `<div class="mtabs">${['All', 'School', 'Transport Team'].map(x => `<button class="mtab ${t === x ? 'on' : ''}" data-act="set" data-k="u:pmt" data-v="${x}">${x}</button>`).join('')}</div><div style="background:#fff">${keys.map(k => { const th = PTHREADS[k], mine = (S.parent.msgs || {})[k] || [], last = mine.length ? mine[mine.length - 1] : null; return `<button class="mrow" style="padding:13px 14px" data-go="#/parent/chat/${k}">${th.person ? av(th.person, 44) : tile(th.ic, th.c, th.c + '1f', 44, 24, 12)}<div style="flex:1;min-width:0"><div class="row"><span class="t" style="flex:1">${th.n}</span><span class="fs12 mut">${last ? last[2] : th.t}</span></div><div class="row mt4"><span class="s ellip" style="flex:1;margin:0">${esc(last ? (last[0] === 'out' ? 'You: ' : '') + last[1] : th.last)}</span>${th.unread && !UI['read_' + k] ? `<span class="sb-badge" style="min-width:18px;height:18px;font-size:10.5px">${th.unread}</span>` : ''}</div></div></button>`; }).join('')}</div>
    <button class="fab" data-go="#/parent/newmsg" aria-label="New message">${ic('pencil', 22)}</button>`;
  },
});
page('parent/chat', {
  pad: false,
  header: r => { const th = PTHREADS[r.sub[0]] || PTHREADS.school; return `<div class="mh"><button class="mh-b" data-go="#/parent/messages">${ic('arrowl', 22)}</button>${th.person ? av(th.person, 40) : `<span class="mic" style="background:#fff;color:${th.c};border-radius:50%;width:40px;height:40px">${ic(th.ic, 22)}</span>`}<div style="flex:1;min-width:0"><div class="b fs15">${th.title || th.n}</div><div class="fs12" style="opacity:.8">${th.sub}</div></div>${mhb('phone', `data-act="call" data-who="${esc(th.title || th.n)}"`)}${mhb('more', 'data-act="toast" data-msg="Chat muted for 8 hours" data-tt="warn"')}</div>`; },
  render: r => {
    const k = r.sub[0] || 'school', th = PTHREADS[k]; UI['read_' + k] = true;
    return `<div style="display:flex;flex-direction:column;min-height:100%;background:#fff"><div style="flex:1;padding:14px 12px;display:flex;flex-direction:column;gap:10px" id="chatlog">${threadMsgs(k).map(m => m[0] === 'in' ? `<div class="row" style="align-items:flex-end">${th.person ? av(th.person, 34) : `<span class="mic sm" style="background:${th.c}1f;color:${th.c};border-radius:50%">${ic(th.ic, 17)}</span>`}<div><div class="bubble in">${esc(m[1])}</div><div class="fs11 mut mt4">${m[2]}</div></div></div>` : `<div class="bubble out">${esc(m[1])}<div class="tm">${m[2]} ${ic('checks', 13, 'style="color:#2563eb"')}</div></div>`).join('')}</div>
    <div class="row" style="padding:10px;border-top:1px solid #eef1f6;position:sticky;bottom:0;background:#fff"><button class="mh-b" style="color:#475569" data-act="toast" data-msg="Attach photo or document">${ic('clipattach', 20)}</button><input class="minp" style="flex:1;border-radius:22px" placeholder="Type a message..." data-bind="u:pchat" data-enter="psend" data-k="${k}" value="${esc(UI.pchat || '')}"/><button class="mh-b" style="color:#2563eb" data-act="psend" data-k="${k}">${ic(UI.pchat ? 'send' : 'mic', 20)}</button></div></div>`;
  },
  mount: () => { const mc = document.querySelector('.mc'); if (mc) mc.scrollTop = mc.scrollHeight; },
});
ACT.psend = d => {
  const v = (UI.pchat || '').trim(); if (!v) { toast('Hold to record a voice note', 'warn'); return; }
  S.parent.msgs[d.k] = [...(S.parent.msgs[d.k] || []), ['out', v, nowTime()]]; UI.pchat = ''; save(); refresh();
  const t = setTimeout(() => { if (CUR.page === 'chat') { S.parent.msgs[d.k].push(['in', 'Thank you, noted. We will update you shortly.', nowTime()]); save(); refresh(); } }, 1800); TIMERS.push(t);
};
pp('newmsg', {
  title: 'New Message', back: '#/parent/messages', right: () => mhb('more', 'data-act="toast" data-msg="Message templates coming soon" data-tt="warn"'), pad: false,
  render: () => {
    const to = UI.pto || 'coord', msg = UI.pnew ?? 'Hi,\nCan you please confirm if Aarav will be dropped at the usual stop today?';
    return `<div style="background:#fff;min-height:100%;padding:12px"><div class="row" style="border-bottom:1px solid #eef1f6;padding-bottom:10px"><span class="sb6">To</span><div class="msel" style="flex:1">${ic('user', 16, 'style="position:absolute;left:10px;right:auto;color:#2563eb"')}<select data-bind="u:pto" data-norefresh style="padding-left:32px">${Object.entries(PTHREADS).filter(([k]) => k !== 'general').map(([k, th]) => `<option value="${k}" ${to === k ? 'selected' : ''}>${th.n}</option>`).join('')}</select>${ic('down', 16)}</div></div>
    <textarea class="minp mt12" style="height:160px;font-size:15px" maxlength="500" data-bind="u:pnew">${esc(msg)}</textarea><div class="tr fs11 mut mt4">${msg.length}/500</div>
    <div class="row mt12"><button class="mh-b" style="color:#475569" data-act="toast" data-msg="Attach photo or document">${ic('clipattach', 20)}</button><span class="sp"></span><button class="btn btn-primary btn-lg" data-act="pnewsend">${ic('send', 18)}Send</button></div></div>`;
  },
});
ACT.pnewsend = () => { const to = UI.pto || 'coord', v = (UI.pnew ?? 'Hi,\nCan you please confirm if Aarav will be dropped at the usual stop today?').trim(); if (!v) { toast('Type a message', 'warn'); return; } S.parent.msgs[to] = [...(S.parent.msgs[to] || []), ['out', v, nowTime()]]; UI.pnew = undefined; save(); toast('Message sent'); go('#/parent/chat/' + to); };
