/* Vector artwork: avatars, buses, maps, charts and illustrated scenes. */
let __uid = 0;
const uid = p => (p || 'u') + (++__uid);
function hashStr(s) { let h = 2166136261; for (const c of String(s)) { h ^= c.charCodeAt(0); h = Math.imul(h, 16777619); } return h >>> 0; }
function rng(seed) { let a = seed >>> 0; return () => { a |= 0; a = a + 0x6D2B79F5 | 0; let t = Math.imul(a ^ a >>> 15, 1 | a); t = t + Math.imul(t ^ t >>> 7, 61 | t) ^ t; return ((t ^ t >>> 14) >>> 0) / 4294967296; }; }

/* ---------------- Avatars ---------------- */
const SKIN = ['#f2c9a5', '#e3b08a', '#d39b72', '#bf855c'];
const HAIR = ['#1d1714', '#2a1f1a', '#3a2a21', '#151515'];
const AVBG = ['#dfe9f7', '#e6eef9', '#e9f1fb', '#e3ecf6'];
/* kind: 'kid-m' | 'kid-f' | 'm' | 'mb' (beard) | 'f' | 'fo' (older woman) | 'mo' (older man) */
function avatarSVG(name, kind) {
  const r = rng(hashStr(name));
  const skin = SKIN[Math.floor(r() * SKIN.length)], hair = HAIR[Math.floor(r() * HAIR.length)], bg = AVBG[Math.floor(r() * AVBG.length)];
  const kid = kind.startsWith('kid'), f = kind === 'f' || kind === 'kid-f' || kind === 'fo';
  const old = kind === 'fo' || kind === 'mo';
  const hc = old ? '#8a8580' : hair;
  const cid = uid('av');
  const shirts = ['#cfe0f5', '#f3f4f6', '#dbe7f3', '#e8e2d5', '#c7d7ea'];
  const top = f && !kid ? ['#d6336c', '#7c3aed', '#0f766e', '#be185d', '#9a3412'][Math.floor(r() * 5)] : shirts[Math.floor(r() * shirts.length)];
  let s = `<svg viewBox="0 0 64 64" xmlns="http://www.w3.org/2000/svg"><defs><clipPath id="${cid}"><circle cx="32" cy="32" r="32"/></clipPath></defs><g clip-path="url(#${cid})"><rect width="64" height="64" fill="${bg}"/>`;
  if (f) s += `<path d="M17.5 31 C15 13 24 8.5 32 8.5 C40 8.5 49 13 46.5 31 L49 ${kid ? 50 : 56} L15 ${kid ? 50 : 56}Z" fill="${hc}"/>`;
  if (kid) {
    s += `<path d="M5 64 C6 50 18 45.5 32 45.5 C46 45.5 58 50 59 64Z" fill="#1f3b6f"/><path d="M24.5 45.5 L32 57 L39.5 45.5Z" fill="#fff"/>`;
    s += f ? `<path d="M29 46 L35 46 L32 50Z" fill="#b91c1c"/>` : `<path d="M30.8 47 L33.2 47 L34.6 56 L32 59.5 L29.4 56Z" fill="#b91c1c"/>`;
  } else if (f) {
    s += `<path d="M6 64 C7 50 18 45.5 32 45.5 C46 45.5 57 50 58 64Z" fill="${top}"/><path d="M26 45.6 Q32 52 38 45.6" fill="${skin}"/>`;
  } else {
    s += `<path d="M5 64 C6 50 18 45.5 32 45.5 C46 45.5 58 50 59 64Z" fill="${top}"/><path d="M25.5 45.4 L32 52 L38.5 45.4 L36 44 L32 48 L28 44Z" fill="#fff"/><path d="M32 52 V64" stroke="rgba(0,0,0,.12)" stroke-width="1"/>`;
  }
  s += `<rect x="27" y="35" width="10" height="12" rx="4" fill="${skin}" /><rect x="27" y="35" width="10" height="5" fill="rgba(0,0,0,.08)"/>`;
  s += `<ellipse cx="20.6" cy="28.5" rx="2.4" ry="3.4" fill="${skin}"/><ellipse cx="43.4" cy="28.5" rx="2.4" ry="3.4" fill="${skin}"/>`;
  s += `<ellipse cx="32" cy="27" rx="11.6" ry="${kid ? 12.8 : 13.6}" fill="${skin}"/>`;
  if (f) s += `<path d="M20.4 27 C19.4 13.5 26 10 32 10 C39 10 45.2 13.5 43.6 27 C41 18.5 36 15.8 28.5 17.8 C24.5 19 22 22 20.4 27Z" fill="${hc}"/>`;
  else s += `<path d="M20.3 25.5 C18.8 12.8 26 10 32.5 10 C40 10 45.6 13.5 43.7 25.5 C42.3 19.6 39 17 32 17 C26 17 22.3 19.6 20.3 25.5Z" fill="${hc}"/>`;
  if (old && !f) s += `<path d="M20.3 25.5 C20.5 22 21.5 20 23 19" stroke="${hc}" stroke-width="2" fill="none"/>`;
  s += `<path d="M25.6 23.3 h4" stroke="${hc}" stroke-width="1.5" stroke-linecap="round"/><path d="M34.4 23.3 h4" stroke="${hc}" stroke-width="1.5" stroke-linecap="round"/>`;
  s += `<circle cx="27.6" cy="27.4" r="1.45" fill="#1f2937"/><circle cx="36.4" cy="27.4" r="1.45" fill="#1f2937"/>`;
  s += `<path d="M32 28.6 q-1.3 3.3 .6 3.7" stroke="rgba(0,0,0,.25)" stroke-width="1.1" fill="none" stroke-linecap="round"/>`;
  if (kind === 'mb') s += `<path d="M21.6 29.5 C22 39 27 42 32 42 C37 42 42 39 42.4 29.5 C41 35 37.5 37.4 32 37.4 C26.5 37.4 23 35 21.6 29.5Z" fill="${hc}" opacity=".92"/><path d="M28 33 Q32 31.3 36 33 Q32 34.4 28 33Z" fill="${hc}"/>`;
  else if (kind === 'm' || kind === 'mo') s += `<path d="M28.3 32.9 Q32 31.6 35.7 32.9 Q32 34 28.3 32.9Z" fill="${hc}" opacity=".85"/>`;
  s += `<path d="M28.6 34.6 Q32 37.2 35.4 34.6" stroke="#8b3a3a" stroke-width="1.3" fill="none" stroke-linecap="round"/>`;
  if (old) s += `<circle cx="27.6" cy="27.4" r="3.3" fill="none" stroke="#555" stroke-width=".9"/><circle cx="36.4" cy="27.4" r="3.3" fill="none" stroke="#555" stroke-width=".9"/><path d="M30.9 27.4h2.2" stroke="#555" stroke-width=".9"/>`;
  return s + '</g></svg>';
}
function av(name, size = 40, kind) {
  kind = kind || (window.DATA && DATA.kindOf ? DATA.kindOf(name) : 'm');
  return `<span class="mav" style="width:${size}px;height:${size}px">${avatarSVG(name, kind)}</span>`;
}
function initials(n) { return n.split(/\s+/).map(w => w[0]).slice(0, 2).join('').toUpperCase(); }

/* ---------------- Buses ---------------- */
function busSide(w = 120, color = 'yellow', text = '') {
  const y = color === 'yellow';
  const body = y ? '#f6b70f' : '#f8fafc', hi = y ? '#ffd34d' : '#ffffff', edge = y ? '#d99a06' : '#cbd5e1';
  const id = uid('bs');
  let s = `<svg viewBox="0 0 240 118" width="${w}" style="display:block" xmlns="http://www.w3.org/2000/svg"><defs><linearGradient id="${id}" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="${hi}"/><stop offset=".35" stop-color="${body}"/><stop offset="1" stop-color="${y ? '#eaa806' : '#e2e8f0'}"/></linearGradient></defs>`;
  s += `<ellipse cx="120" cy="108" rx="112" ry="6" fill="rgba(15,23,42,.14)"/>`;
  s += `<path d="M10 22 Q10 12 20 12 H194 Q205 12 211 21 L225 46 Q230 53 230 61 V88 Q230 96 222 96 H18 Q10 96 10 88Z" fill="url(#${id})" stroke="${edge}" stroke-width="1.2"/>`;
  if (y) {
    for (let i = 0; i < 5; i++) s += `<rect x="${20 + i * 30}" y="24" width="25" height="27" rx="3" fill="#24364d"/><path d="M${22 + i * 30} 49 L${33 + i * 30} 26 H${38 + i * 30} L${27 + i * 30} 49Z" fill="rgba(255,255,255,.14)"/>`;
    s += `<rect x="172" y="23" width="23" height="62" rx="2.5" fill="#24364d"/><path d="M183.5 23 V85" stroke="${body}" stroke-width="1.6"/>`;
    s += `<path d="M199 22 H205 Q209 22 211 26 L222 48 Q223.5 53 219 53 H199Z" fill="#24364d"/>`;
    s += `<rect x="10" y="62" width="160" height="3.5" fill="#1f2937" opacity=".85"/><rect x="10" y="73" width="160" height="2.4" fill="#1f2937" opacity=".7"/><rect x="198" y="62" width="32" height="3.5" fill="#1f2937" opacity=".85"/>`;
    if (text) s += `<text x="90" y="${text.length > 8 ? 59.5 : 59.5}" font-size="8.5" font-weight="800" font-family="Arial" fill="#1f2937" text-anchor="middle" letter-spacing=".6">${text}</text>`;
  } else {
    s += `<path d="M18 24 H196 L210 50 H18Z" fill="#1e2b3c"/>`;
    for (let i = 1; i < 6; i++) s += `<rect x="${18 + i * 30}" y="24" width="2.4" height="26" fill="#f8fafc"/>`;
    s += `<rect x="172" y="24" width="20" height="61" rx="2" fill="#1e2b3c"/>`;
    s += `<path d="M10 66 H230" stroke="#1463ff" stroke-width="4"/><path d="M10 72 H230" stroke="#f6b70f" stroke-width="2.6"/><path d="M60 82 Q120 70 170 82" stroke="#1463ff" stroke-width="2" fill="none"/>`;
  }
  s += `<rect x="221" y="68" width="8" height="9" rx="2" fill="#fff8d6"/><rect x="10" y="68" width="5" height="10" rx="1" fill="#ef4444"/>`;
  s += `<rect x="8" y="88" width="224" height="8" rx="3.5" fill="#334155"/>`;
  s += `<path d="M36 96 a22 22 0 0 1 44 0Z" fill="#1e293b"/><path d="M164 96 a22 22 0 0 1 44 0Z" fill="#1e293b"/>`;
  for (const cx of [58, 186]) s += `<circle cx="${cx}" cy="96" r="15.5" fill="#111827"/><circle cx="${cx}" cy="96" r="8" fill="#9ca3af"/><circle cx="${cx}" cy="96" r="3" fill="#4b5563"/>`;
  s += `<rect x="226" y="34" width="3" height="12" rx="1.5" fill="#111827"/>`;
  return s + '</svg>';
}
/* Top-down bus marker for maps. */
function busTop(x, y, a = 0, attrs = '', s = 1) {
  return `<g ${attrs} transform="translate(${x},${y}) rotate(${a}) scale(${s})"><rect x="-17" y="-8" width="34" height="16" rx="4" fill="#f6b70f" stroke="#a87405" stroke-width="1"/><rect x="-14" y="-5.5" width="23" height="11" rx="2" fill="#ffd34d"/><path d="M-12 -5.5v11M-7 -5.5v11M-2 -5.5v11M3 -5.5v11" stroke="#e0a40a" stroke-width="1"/><rect x="11" y="-6" width="4.5" height="12" rx="1.5" fill="#24364d"/><rect x="-17" y="-8" width="34" height="2.2" rx="1" fill="#24364d" opacity=".8"/><rect x="-17" y="5.8" width="34" height="2.2" rx="1" fill="#24364d" opacity=".8"/></g>`;
}

/* ---------------- Maps ---------------- */
function smoothPath(pts) {
  if (pts.length < 3) return 'M' + pts.map(p => p.join(',')).join(' L');
  let d = `M${pts[0][0]},${pts[0][1]}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2;
    const c1 = [p1[0] + (p2[0] - p0[0]) / 6, p1[1] + (p2[1] - p0[1]) / 6], c2 = [p2[0] - (p3[0] - p1[0]) / 6, p2[1] - (p3[1] - p1[1]) / 6];
    d += ` C${c1[0].toFixed(1)},${c1[1].toFixed(1)} ${c2[0].toFixed(1)},${c2[1].toFixed(1)} ${p2[0]},${p2[1]}`;
  }
  return d;
}
function mapBase(W, H, seed) {
  const r = rng(seed);
  let s = `<rect width="${W}" height="${H}" fill="#eceee9"/>`;
  for (let i = 0; i < 9; i++) {
    const x = r() * W, y = r() * H, rx = 30 + r() * 70, ry = 20 + r() * 50;
    s += `<ellipse cx="${x.toFixed(0)}" cy="${y.toFixed(0)}" rx="${rx.toFixed(0)}" ry="${ry.toFixed(0)}" fill="#d6e8cd" transform="rotate(${(r() * 60 - 30).toFixed(0)} ${x.toFixed(0)} ${y.toFixed(0)})"/>`;
  }
  for (let i = 0; i < 5; i++) {
    const x = r() * W, y = r() * H;
    s += `<rect x="${x.toFixed(0)}" y="${y.toFixed(0)}" width="${(40 + r() * 70).toFixed(0)}" height="${(26 + r() * 40).toFixed(0)}" rx="6" fill="#e2e4df"/>`;
  }
  const g = 46 + r() * 16, ang = r() * 16 - 8;
  s += `<g stroke="#ffffff" stroke-width="3" opacity=".95" transform="rotate(${ang.toFixed(1)} ${W / 2} ${H / 2})">`;
  for (let x = -W * .3; x < W * 1.3; x += g + r() * 30) s += `<path d="M${x.toFixed(0)},${-H * .3} L${(x + r() * 60 - 30).toFixed(0)},${H * 1.3}"/>`;
  for (let y = -H * .3; y < H * 1.3; y += g + r() * 30) s += `<path d="M${-W * .3},${y.toFixed(0)} L${W * 1.3},${(y + r() * 60 - 30).toFixed(0)}"/>`;
  s += `</g>`;
  for (let i = 0; i < 4; i++) {
    const pts = []; const vertical = i % 2 === 0; const off = (i + 1) * (vertical ? W : H) / 5 + (r() * 80 - 40);
    for (let k = 0; k <= 5; k++) { const t = k / 5; pts.push(vertical ? [off + (r() * 90 - 45), t * H * 1.2 - H * .1] : [t * W * 1.2 - W * .1, off + (r() * 90 - 45)]); }
    const d = smoothPath(pts.map(p => [Math.round(p[0]), Math.round(p[1])]));
    s += `<path d="${d}" stroke="#d9d4c3" stroke-width="10" fill="none"/><path d="${d}" stroke="#fbf2cf" stroke-width="7" fill="none"/>`;
  }
  return s;
}
/* opts: {w,h,seed,labels:[[x,y,text,size]],routes:[{id,pts,c,w,dash,glow}],markers:[...]} */
function mapSVG(o) {
  const W = o.w || 600, H = o.h || 340;
  let s = `<svg class="mapsvg" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMid slice" xmlns="http://www.w3.org/2000/svg"><defs><filter id="msh" x="-30%" y="-30%" width="160%" height="160%"><feDropShadow dx="0" dy="2" stdDeviation="2.5" flood-opacity=".18"/></filter></defs>`;
  s += mapBase(W, H, o.seed || 7);
  for (const l of o.labels || []) s += `<text x="${l[0]}" y="${l[1]}" font-size="${l[3] || 12}" font-weight="${l[4] || 500}" fill="${l[5] || '#5d6672'}" text-anchor="middle" font-family="Inter,Arial" stroke="#f3f4f0" stroke-width="3" paint-order="stroke">${l[2]}</text>`;
  for (const rt of o.routes || []) {
    const d = smoothPath(rt.pts);
    if (rt.glow !== false) s += `<path d="${d}" stroke="#fff" stroke-width="${(rt.w || 4) + 3}" fill="none" stroke-linecap="round" opacity=".9"/>`;
    s += `<path ${rt.id ? `id="${rt.id}"` : ''} d="${d}" stroke="${rt.c}" stroke-width="${rt.w || 4}" fill="none" stroke-linecap="round" stroke-linejoin="round" ${rt.dash ? `stroke-dasharray="${rt.dash}"` : ''}/>`;
  }
  for (const m of o.markers || []) s += marker(m);
  return s + '</svg>';
}
function marker(m) {
  const { x, y } = m, c = m.c || '#1463ff';
  const act = m.act ? `data-act="${m.act}" data-id="${m.id || ''}" class="mk"` : '';
  switch (m.t) {
    case 'badge': return `<g ${act} filter="url(#msh)"><path d="M${x - 14},${y - 34} h28 a4 4 0 0 1 4 4 v20 a4 4 0 0 1 -4 4 h-9 l-5 6 l-5 -6 h-9 a4 4 0 0 1 -4 -4 v-20 a4 4 0 0 1 4 -4z" fill="${c}" stroke="#fff" stroke-width="1.6"/><g transform="translate(${x - 9},${y - 31}) scale(.75)" stroke="#fff" fill="none" stroke-width="2.3" stroke-linecap="round" stroke-linejoin="round">${ICONS[m.icon || 'busf']}</g></g>`;
    case 'pin': return `<g ${act} filter="url(#msh)" transform="translate(${x},${y}) scale(${m.s || 1})"><path d="M0,0 C-8,-10 -12,-15 -12,-22 A12,12 0 1 1 12,-22 C12,-15 8,-10 0,0Z" fill="${c}" stroke="#fff" stroke-width="1.6"/>${m.icon ? `<g transform="translate(-7.5,-29.5) scale(.62)" stroke="#fff" fill="none" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">${ICONS[m.icon]}</g>` : `<circle cx="0" cy="-22" r="4.6" fill="#fff"/>`}</g>`;
    case 'num': return `<g ${act} filter="url(#msh)"><circle cx="${x}" cy="${y}" r="${m.r || 10}" fill="${c}" stroke="#fff" stroke-width="2.5"/><text x="${x}" y="${y + 4}" font-size="11" font-weight="700" fill="#fff" text-anchor="middle" font-family="Inter,Arial">${m.n}</text></g>`;
    case 'ring': return `<circle cx="${x}" cy="${y}" r="${m.r || 7}" fill="#fff" stroke="${c}" stroke-width="3"/>`;
    case 'dot': return `<g>${m.pulse ? `<circle cx="${x}" cy="${y}" r="${(m.r || 6) * 2}" fill="${c}" opacity=".25" class="pulse" style="transform-origin:${x}px ${y}px"/>` : ''}<circle cx="${x}" cy="${y}" r="${m.r || 6}" fill="${c}" stroke="#fff" stroke-width="2"/></g>`;
    case 'bus': return busTop(x, y, m.a || 0, `${act} ${m.anim ? `data-anim="${m.anim}" data-dur="${m.dur || 30}" data-off="${m.off || 0}"` : ''} filter="url(#msh)"`, m.s || 1);
    case 'tag': {
      const w = m.w || (m.text.length * 6.6 + 18);
      return `<g ${act} filter="url(#msh)"><rect x="${x}" y="${y - 11}" width="${w}" height="22" rx="${m.rx || 6}" fill="${m.bg || '#fff'}"/><text x="${x + w / 2}" y="${y + 4}" font-size="${m.fs || 11}" font-weight="700" fill="${m.c || '#0f172a'}" text-anchor="middle" font-family="Inter,Arial">${m.text}</text></g>`;
    }
    case 'call': {
      const lines = m.lines, w = m.w || Math.max(...lines.map(l => String(l[0]).length)) * 6.4 + 22, h = lines.length * 15 + 12;
      let t = `<g ${act} filter="url(#msh)"><rect x="${x}" y="${y}" width="${w}" height="${h}" rx="8" fill="#fff"/>`;
      lines.forEach((l, i) => t += `<text x="${x + 11}" y="${y + 19 + i * 15}" font-size="${l[2] || 11.5}" font-weight="${l[1] || 500}" fill="${l[3] || '#1f2937'}" font-family="Inter,Arial">${l[0]}</text>`);
      return t + '</g>';
    }
    case 'home': return `<g filter="url(#msh)"><circle cx="${x}" cy="${y}" r="13" fill="#16a34a" stroke="#fff" stroke-width="2"/><g transform="translate(${x - 7},${y - 7}) scale(.58)" stroke="#fff" fill="none" stroke-width="2.6">${ICONS.home}</g></g>`;
    default: return '';
  }
}

/* ---------------- Charts ---------------- */
function donut(segs, size = 120, w = 16, center = '', gap = 1.2) {
  const r = (size - w) / 2, C = 2 * Math.PI * r, tot = segs.reduce((a, b) => a + b.v, 0) || 1;
  let acc = 0, arcs = '';
  for (const sg of segs) {
    const len = Math.max(0, sg.v / tot * C - (segs.length > 1 ? gap : 0));
    if (sg.v > 0) arcs += `<circle cx="${size / 2}" cy="${size / 2}" r="${r}" fill="none" stroke="${sg.c}" stroke-width="${w}" stroke-dasharray="${len.toFixed(2)} ${(C - len).toFixed(2)}" stroke-dashoffset="${(-acc).toFixed(2)}" transform="rotate(-90 ${size / 2} ${size / 2})"/>`;
    acc += sg.v / tot * C;
  }
  return `<div class="donut" style="width:${size}px;height:${size}px"><svg width="${size}" height="${size}"><circle cx="${size / 2}" cy="${size / 2}" r="${r}" fill="none" stroke="#eef1f5" stroke-width="${w}"/>${arcs}</svg><div class="donut-c">${center}</div></div>`;
}
/* o: {w,h,labels,series:[{v,c}],ticks,fmt,line:{v,c,ticks},yl,yr,showVals,hl} */
function bars(o) {
  const W = o.w || 520, H = o.h || 160, pl = o.pl ?? 36, pr = o.line ? 34 : (o.pr ?? 8), pt = o.pt ?? 10, pb = o.pb ?? 22;
  const ticks = o.ticks, mx = ticks[ticks.length - 1], iw = W - pl - pr, ih = H - pt - pb, n = o.labels.length, bw = iw / n;
  const fmt = o.fmt || (v => v);
  let s = `<svg viewBox="0 0 ${W} ${H}" width="100%" style="display:block" font-family="Inter,Arial">`;
  ticks.forEach(t => { const y = pt + ih - t / mx * ih; s += `<line x1="${pl}" x2="${W - pr}" y1="${y}" y2="${y}" stroke="#eef1f5"/><text x="${pl - 6}" y="${y + 3.5}" font-size="10" fill="#64748b" text-anchor="end">${fmt(t)}</text>`; });
  if (o.line) o.line.ticks.forEach((t, i) => { const y = pt + ih - i / (o.line.ticks.length - 1) * ih; s += `<text x="${W - pr + 6}" y="${y + 3.5}" font-size="10" fill="#64748b">${t}</text>`; });
  const ns = o.series.length, gw = bw * (o.bw || .56), sw = gw / ns;
  o.labels.forEach((lb, i) => {
    const cx = pl + bw * i + bw / 2;
    o.series.forEach((se, k) => {
      const v = se.v[i], h = Math.max(0, v / mx * ih), x = cx - gw / 2 + k * sw + (ns > 1 ? 1 : 0), y = pt + ih - h;
      const c = Array.isArray(se.c) ? se.c[i] : se.c;
      s += `<rect x="${x.toFixed(1)}" y="${y.toFixed(1)}" width="${(sw - (ns > 1 ? 2 : 0)).toFixed(1)}" height="${h.toFixed(1)}" rx="2" fill="${c}"><title>${lb}: ${fmt(v)}</title></rect>`;
      if (o.showVals) s += `<text x="${(x + sw / 2).toFixed(1)}" y="${(y - 5).toFixed(1)}" font-size="11" font-weight="600" fill="#334155" text-anchor="middle">${v}</text>`;
    });
    s += `<text x="${cx}" y="${H - 6}" font-size="10" fill="${o.hl === i ? '#1463ff' : '#64748b'}" font-weight="${o.hl === i ? 700 : 400}" text-anchor="middle">${lb}</text>`;
  });
  if (o.line) {
    const lm = parseFloat(o.line.ticks[o.line.ticks.length - 1]);
    const pts = o.line.v.map((v, i) => [pl + bw * i + bw / 2, pt + ih - v / lm * ih]);
    s += `<polyline points="${pts.map(p => p.map(n => n.toFixed(1)).join(',')).join(' ')}" fill="none" stroke="${o.line.c}" stroke-width="2.2"/>`;
    pts.forEach(p => s += `<circle cx="${p[0].toFixed(1)}" cy="${p[1].toFixed(1)}" r="4.2" fill="#fff" stroke="${o.line.c}" stroke-width="2.2"/>`);
  }
  if (o.yl) s += `<text transform="translate(10 ${pt + ih / 2}) rotate(-90)" font-size="10.5" fill="#475569" text-anchor="middle">${o.yl}</text>`;
  if (o.yr) s += `<text transform="translate(${W - 4} ${pt + ih / 2}) rotate(-90)" font-size="10.5" fill="#475569" text-anchor="middle">${o.yr}</text>`;
  return s + '</svg>';
}
function ringTimer(frac, size, color, center) {
  const r = (size - 14) / 2, C = 2 * Math.PI * r;
  return `<div class="ring" style="width:${size}px;height:${size}px"><svg width="${size}" height="${size}"><circle cx="${size / 2}" cy="${size / 2}" r="${r}" stroke="#e5e9f0" stroke-width="10" fill="none"/><circle cx="${size / 2}" cy="${size / 2}" r="${r}" stroke="${color}" stroke-width="10" fill="none" stroke-linecap="round" stroke-dasharray="${(C * frac).toFixed(1)} ${C}" transform="rotate(-90 ${size / 2} ${size / 2})" style="transition:stroke-dasharray 1s linear"/></svg><div class="c">${center}</div></div>`;
}

/* ---------------- Status bar ---------------- */
function statusIcons(dark) {
  const c = dark ? '#0f172a' : '#fff';
  return `<span class="si"><svg width="18" height="12" viewBox="0 0 18 12"><rect x="0" y="8" width="3" height="4" rx="1" fill="${c}"/><rect x="5" y="5.5" width="3" height="6.5" rx="1" fill="${c}"/><rect x="10" y="3" width="3" height="9" rx="1" fill="${c}"/><rect x="15" y="0" width="3" height="12" rx="1" fill="${c}"/></svg><svg width="16" height="12" viewBox="0 0 16 12"><path d="M8 11.5 L5.6 8.8 A3.4 3.4 0 0 1 10.4 8.8Z M2.6 5.8 A7.6 7.6 0 0 1 13.4 5.8 L12 7.3 A5.6 5.6 0 0 0 4 7.3Z M0 3 A11.4 11.4 0 0 1 16 3 L14.6 4.5 A9.4 9.4 0 0 0 1.4 4.5Z" fill="${c}"/></svg><svg width="26" height="12" viewBox="0 0 26 12"><rect x=".5" y=".5" width="22" height="11" rx="3.2" fill="none" stroke="${c}" opacity=".5"/><rect x="2" y="2" width="19" height="8" rx="2" fill="${c}"/><rect x="23.6" y="4" width="1.8" height="4" rx=".9" fill="${c}" opacity=".5"/></svg></span>`;
}

/* ---------------- Scenes ---------------- */
function svgWrap(vb, body, extra = '') { return `<svg viewBox="${vb}" preserveAspectRatio="xMidYMid slice" xmlns="http://www.w3.org/2000/svg" ${extra}>${body}</svg>`; }
function tree(x, y, s = 1, c = '#4caf50') { return `<g transform="translate(${x},${y}) scale(${s})"><rect x="-2" y="-4" width="4" height="16" fill="#7a5532"/><circle cx="0" cy="-12" r="11" fill="${c}"/><circle cx="-7" cy="-6" r="7" fill="${c}"/><circle cx="7" cy="-6" r="7" fill="${c}"/><circle cx="-3" cy="-15" r="5" fill="rgba(255,255,255,.12)"/></g>`; }
function kid(x, y, s = 1, flip = false) {
  return `<g transform="translate(${x},${y}) scale(${flip ? -s : s},${s})"><rect x="-9" y="-30" width="13" height="20" rx="4" fill="#1d4ed8"/><rect x="-5" y="-34" width="12" height="22" rx="4" fill="#1f3b6f"/><rect x="-4" y="-12" width="4" height="13" rx="2" fill="#334155"/><rect x="2" y="-12" width="4" height="13" rx="2" fill="#334155"/><circle cx="1" cy="-40" r="6.5" fill="#d39b72"/><path d="M-5.5 -41 C-5 -48 7 -48 7.5 -41 C5 -44 -2 -44 -5.5 -41Z" fill="#1d1714"/></g>`;
}
const SCENES = {
  tyre: () => svgWrap('0 0 300 140', `<defs><radialGradient id="tg" cx=".5" cy=".5" r=".6"><stop offset="0" stop-color="#3a3f46"/><stop offset="1" stop-color="#14171b"/></radialGradient></defs><rect width="300" height="140" fill="#2a2e33"/><rect width="300" height="54" fill="#f2b20d"/><rect y="50" width="300" height="6" fill="#1f2937" opacity=".5"/><path d="M60 140 A90 90 0 0 1 240 140" fill="#1b1e22"/><circle cx="150" cy="128" r="74" fill="url(#tg)"/>${Array.from({ length: 28 }, (_, i) => { const a = i / 28 * Math.PI * 2; return `<rect x="146" y="56" width="8" height="12" rx="2" fill="#0d0f12" transform="rotate(${a * 57.3} 150 128)"/>`; }).join('')}<circle cx="150" cy="128" r="44" fill="#b9bec6"/><circle cx="150" cy="128" r="36" fill="#d7dbe0"/><circle cx="150" cy="128" r="14" fill="#8e949c"/>${Array.from({ length: 8 }, (_, i) => `<circle cx="${150 + 25 * Math.cos(i * Math.PI / 4)}" cy="${128 + 25 * Math.sin(i * Math.PI / 4)}" r="3.2" fill="#6b7280"/>`).join('')}`),
  brake: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#30353b"/><circle cx="150" cy="120" r="88" fill="#1a1d21"/><circle cx="150" cy="120" r="62" fill="#9aa0a8"/><circle cx="150" cy="120" r="56" fill="#b4bac2"/>${Array.from({ length: 18 }, (_, i) => `<circle cx="${150 + 46 * Math.cos(i / 18 * Math.PI * 2)}" cy="${120 + 46 * Math.sin(i / 18 * Math.PI * 2)}" r="2.6" fill="#6b7280"/>`).join('')}<circle cx="150" cy="120" r="24" fill="#5b6169"/>${Array.from({ length: 6 }, (_, i) => `<circle cx="${150 + 15 * Math.cos(i * Math.PI / 3)}" cy="${120 + 15 * Math.sin(i * Math.PI / 3)}" r="3" fill="#2d3238"/>`).join('')}<path d="M196 70 Q222 90 222 122 L204 122 Q204 96 186 82Z" fill="#c62828"/><rect x="200" y="96" width="14" height="10" rx="2" fill="#8e1b1b"/>`),
  lights: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#f2b20d"/><rect y="0" width="300" height="30" fill="#e3a508"/><rect x="0" y="96" width="300" height="44" fill="#1f2937"/><rect x="110" y="56" width="80" height="34" rx="6" fill="#1f2937"/>${[0, 1, 2, 3, 4].map(i => `<rect x="118" y="${60 + i * 6}" width="64" height="2.5" fill="#4b5563"/>`).join('')}${[[52, 70], [248, 70]].map(([x, y]) => `<rect x="${x - 40}" y="${y - 22}" width="80" height="42" rx="12" fill="#2b2f36"/><circle cx="${x - 16}" cy="${y}" r="15" fill="#fff9e0"/><circle cx="${x - 16}" cy="${y}" r="9" fill="#fff"/><circle cx="${x + 18}" cy="${y}" r="11" fill="#ffe8a3"/><rect x="${x - 36}" y="${y + 14}" width="18" height="5" rx="2" fill="#f97316"/>`).join('')}<rect x="120" y="104" width="60" height="16" rx="2" fill="#fff"/><text x="150" y="116" font-size="9" font-weight="700" text-anchor="middle" font-family="Arial">GJ01EF9012</text>`),
  emkit: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#23272d"/><rect y="104" width="300" height="36" fill="#3a3f46"/><path d="M60 118 L104 34 L148 118Z" fill="#e53935"/><path d="M78 108 L104 58 L130 108Z" fill="#23272d"/><rect x="150" y="70" width="80" height="44" rx="5" fill="#c62828"/><rect x="180" y="62" width="20" height="10" rx="3" fill="#8e1b1b"/><rect x="156" y="84" width="68" height="4" fill="#8e1b1b"/><path d="M246 118 L258 70 L270 118Z" fill="#fb8c00"/><rect x="251" y="92" width="14" height="5" fill="#fff"/><rect x="232" y="40" width="40" height="22" rx="4" fill="#9ca3af" transform="rotate(-18 252 51)"/>`),
  extinguisher: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#2b2f35"/><rect x="40" y="10" width="220" height="130" rx="6" fill="#3a3f46"/><rect x="128" y="34" width="44" height="110" rx="18" fill="#d32f2f"/><rect x="128" y="34" width="12" height="110" rx="6" fill="#ef5350" opacity=".6"/><rect x="138" y="18" width="24" height="18" rx="3" fill="#1f2937"/><path d="M162 24 Q190 24 186 60 L182 92" stroke="#111" stroke-width="5" fill="none"/><circle cx="150" cy="26" r="6" fill="#e5e7eb" stroke="#111"/><path d="M150 26 l3 -3" stroke="#16a34a" stroke-width="1.6"/><rect x="134" y="70" width="32" height="34" rx="3" fill="#fff"/><text x="150" y="83" font-size="6.5" font-weight="800" text-anchor="middle" font-family="Arial" fill="#d32f2f">FIRE</text><text x="150" y="92" font-size="5" text-anchor="middle" font-family="Arial">ABC TYPE</text><rect x="120" y="46" width="60" height="6" rx="3" fill="#111" opacity=".6"/>`),
  firstaid: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#3b4048"/><rect y="110" width="300" height="30" fill="#4b5058"/><rect x="80" y="36" width="140" height="86" rx="10" fill="#e5e7eb"/><rect x="80" y="36" width="140" height="18" rx="8" fill="#cbd0d6"/><rect x="132" y="24" width="36" height="14" rx="4" fill="#9ca3af"/><rect x="138" y="58" width="24" height="56" rx="2" fill="#e53935"/><rect x="122" y="74" width="56" height="24" rx="2" fill="#e53935"/>`),
  school: () => svgWrap('0 0 300 160', `<rect width="300" height="160" fill="#bfe0f5"/><rect y="118" width="300" height="42" fill="#9ccc65"/><rect y="128" width="300" height="32" fill="#cfd8dc"/><rect x="50" y="56" width="200" height="72" fill="#b5523b"/><path d="M40 58 L150 22 L260 58Z" fill="#8d3b2a"/><rect x="128" y="40" width="44" height="88" fill="#c76446"/><path d="M122 42 L150 26 L178 42Z" fill="#6d2b1f"/><rect x="138" y="92" width="24" height="36" fill="#5b2a1e"/>${[0, 1, 2, 3].map(i => `<rect x="${60 + i * 16}" y="68" width="10" height="14" fill="#fff"/><rect x="${60 + i * 16}" y="94" width="10" height="14" fill="#fff"/><rect x="${186 + i * 16}" y="68" width="10" height="14" fill="#fff"/><rect x="${186 + i * 16}" y="94" width="10" height="14" fill="#fff"/>`).join('')}<rect x="132" y="48" width="36" height="10" fill="#f5f5f5"/><text x="150" y="56" font-size="5.5" text-anchor="middle" font-family="Arial" font-weight="700">GREENWOOD</text>${tree(26, 112, 1.3)}${tree(276, 112, 1.3)}${tree(8, 120, 1)}`),
  busstop: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#cfe7f5"/><rect y="104" width="300" height="36" fill="#9e9e9e"/><rect y="98" width="300" height="8" fill="#bdbdbd"/>${tree(40, 92, 1.6, '#43a047')}${tree(262, 92, 1.7, '#388e3c')}${tree(220, 88, 1.2)}<rect x="96" y="40" width="112" height="8" rx="2" fill="#8d6e63"/><rect x="100" y="48" width="4" height="52" fill="#5d4037"/><rect x="200" y="48" width="4" height="52" fill="#5d4037"/><rect x="104" y="48" width="96" height="40" fill="#e0f2f1" opacity=".55"/><rect x="118" y="82" width="70" height="6" rx="2" fill="#a1887f"/><rect x="214" y="44" width="4" height="56" fill="#424242"/><rect x="208" y="38" width="16" height="12" rx="2" fill="#1565c0"/>`),
  boarding: () => svgWrap('0 0 200 180', `<rect width="200" height="180" fill="#f2b20d"/><rect x="0" y="0" width="200" height="22" fill="#e3a508"/><rect x="88" y="20" width="76" height="150" rx="4" fill="#24364d"/><rect x="92" y="26" width="32" height="70" fill="#3d5470"/><rect x="128" y="26" width="32" height="70" fill="#3d5470"/><rect x="0" y="150" width="200" height="30" fill="#6b7280"/><rect x="10" y="40" width="64" height="44" rx="4" fill="#24364d"/>${kid(110, 158, 2.2)}`),
  drop: () => svgWrap('0 0 200 180', `<rect width="200" height="180" fill="#f2b20d"/><rect x="0" y="0" width="200" height="22" fill="#e3a508"/><rect x="20" y="20" width="76" height="150" rx="4" fill="#24364d"/><rect x="24" y="26" width="32" height="70" fill="#3d5470"/><rect x="60" y="26" width="32" height="70" fill="#3d5470"/><rect x="0" y="150" width="200" height="30" fill="#6b7280"/>${tree(160, 120, 2, '#43a047')}${kid(110, 166, 2.2, true)}`),
  traffic: () => svgWrap('0 0 300 140', `<rect width="300" height="140" fill="#b0bec5"/><rect width="300" height="40" fill="#cfd8dc"/>${tree(20, 40, 1.2)}${tree(280, 40, 1.2)}<path d="M120 40 L180 40 L300 140 L0 140Z" fill="#616161"/><path d="M150 44 V140" stroke="#fff" stroke-dasharray="8 8" stroke-width="2"/>${[[130, 52, '#e53935'], [160, 54, '#1e88e5'], [118, 70, '#fdd835'], [170, 74, '#43a047'], [100, 94, '#fff'], [180, 98, '#8e24aa'], [76, 120, '#e53935'], [200, 122, '#1e88e5'], [140, 112, '#fb8c00']].map(([x, y, c]) => { const s = 0.5 + y / 140; return `<rect x="${x - 10 * s}" y="${y}" width="${20 * s}" height="${12 * s}" rx="${3 * s}" fill="${c}"/><rect x="${x - 7 * s}" y="${y + 2 * s}" width="${14 * s}" height="${4 * s}" fill="#263238" opacity=".6"/>`; }).join('')}`),
  engine: () => svgWrap('0 0 200 140', `<rect width="200" height="140" fill="#37474f"/><rect x="30" y="30" width="140" height="80" rx="8" fill="#546e7a"/>${[0, 1, 2, 3].map(i => `<rect x="${44 + i * 30}" y="20" width="22" height="30" rx="4" fill="#78909c"/>`).join('')}<circle cx="60" cy="86" r="16" fill="#263238"/><circle cx="60" cy="86" r="8" fill="#90a4ae"/><rect x="96" y="70" width="60" height="30" rx="4" fill="#455a64"/><path d="M30 60 Q10 70 18 110" stroke="#212121" stroke-width="5" fill="none"/>`),
  brakepad: () => svgWrap('0 0 200 140', `<rect width="200" height="140" fill="#455a64"/><path d="M40 90 Q100 30 160 90 L150 104 Q100 56 50 104Z" fill="#9e9e9e"/><path d="M50 90 Q100 46 150 90 L144 98 Q100 62 56 98Z" fill="#424242"/>`),
  under: () => svgWrap('0 0 200 140', `<rect width="200" height="140" fill="#263238"/><rect x="0" y="40" width="200" height="16" fill="#455a64"/><rect x="0" y="84" width="200" height="16" fill="#455a64"/><rect x="60" y="20" width="18" height="100" fill="#37474f"/><rect x="130" y="20" width="18" height="100" fill="#37474f"/><circle cx="100" cy="70" r="16" fill="#546e7a"/>`),
  interior: () => svgWrap('0 0 200 140', `<rect width="200" height="140" fill="#8d6e63"/><rect width="200" height="40" fill="#d7ccc8"/>${[0, 1, 2].map(r => [0, 1, 2, 3].map(c => `<rect x="${14 + c * 46}" y="${50 + r * 30}" width="34" height="22" rx="4" fill="#5d4037"/>`).join('')).join('')}${[0, 1, 2, 3, 4].map(i => `<rect x="${10 + i * 38}" y="8" width="30" height="20" rx="3" fill="#90caf9"/>`).join('')}`),
  windshield: () => svgWrap('0 0 200 140', `<rect width="200" height="140" fill="#37474f"/><path d="M20 20 H180 L170 110 H30Z" fill="#81d4fa"/><path d="M60 110 L110 40" stroke="#212121" stroke-width="3"/><path d="M120 110 L170 40" stroke="#212121" stroke-width="3"/>`),
};
function scene(name) { return (SCENES[name] || SCENES.interior)(); }
/* Thumbnail tile with an icon on a dark gradient (used for check items) */
function thumbIcon(icon, from = '#475569', to = '#1e293b') {
  const id = uid('tg');
  return svgWrap('0 0 60 52', `<defs><linearGradient id="${id}" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${from}"/><stop offset="1" stop-color="${to}"/></linearGradient></defs><rect width="60" height="52" fill="url(#${id})"/><g transform="translate(18,14) scale(1)" stroke="#fff" fill="none" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${ICONS[icon] || ''}</g>`);
}
function partArt(t) {
  const body = {
    oil: `<rect x="14" y="16" width="30" height="34" rx="4" fill="#c62828"/><rect x="20" y="10" width="12" height="8" rx="2" fill="#424242"/><rect x="18" y="26" width="22" height="14" rx="2" fill="#fff"/><path d="M44 22 h6 v10 h-6" stroke="#424242" stroke-width="3" fill="none"/>`,
    pad: `<path d="M8 40 Q30 8 52 40 L46 46 Q30 22 14 46Z" fill="#424242"/><path d="M14 38 Q30 16 46 38" stroke="#9e9e9e" stroke-width="3" fill="none"/>`,
    filter: `<rect x="16" y="10" width="28" height="40" rx="3" fill="#a1887f"/>${[0, 1, 2, 3, 4, 5].map(i => `<path d="M${19 + i * 4.4} 12 v36" stroke="#6d4c41" stroke-width="1.6"/>`).join('')}`,
    fuelf: `<rect x="20" y="8" width="20" height="42" rx="8" fill="#e0e0e0" stroke="#9e9e9e"/><rect x="26" y="4" width="8" height="6" fill="#757575"/><rect x="20" y="22" width="20" height="10" fill="#1565c0"/>`,
    tyre: `<circle cx="30" cy="30" r="22" fill="#212121"/><circle cx="30" cy="30" r="11" fill="#9e9e9e"/><circle cx="30" cy="30" r="4" fill="#616161"/>`,
    bulb: `<circle cx="24" cy="30" r="12" fill="#eceff1" stroke="#90a4ae" stroke-width="2"/><rect x="34" y="25" width="16" height="10" rx="2" fill="#616161"/><circle cx="24" cy="30" r="5" fill="#fff59d"/>`,
    battery: `<rect x="10" y="18" width="40" height="30" rx="3" fill="#263238"/><rect x="16" y="12" width="8" height="7" fill="#e53935"/><rect x="36" y="12" width="8" height="7" fill="#424242"/><rect x="14" y="28" width="32" height="8" fill="#ffb300"/>`,
    wiper: `<path d="M8 50 L52 10" stroke="#212121" stroke-width="4" stroke-linecap="round"/><path d="M14 50 L52 16" stroke="#757575" stroke-width="2"/>`,
  }[t] || '';
  return `<svg viewBox="0 0 60 60" width="100%" height="100%">${body}</svg>`;
}
function qrArt(seed = 3, size = 21) {
  const r = rng(seed); let s = '';
  const fin = (x, y) => `<rect x="${x}" y="${y}" width="7" height="7" fill="#111"/><rect x="${x + 1}" y="${y + 1}" width="5" height="5" fill="#fff"/><rect x="${x + 2}" y="${y + 2}" width="3" height="3" fill="#111"/>`;
  for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) {
    if ((x < 8 && y < 8) || (x > size - 9 && y < 8) || (x < 8 && y > size - 9)) continue;
    if (r() > .52) s += `<rect x="${x}" y="${y}" width="1" height="1" fill="#111"/>`;
  }
  return `<svg viewBox="-1 -1 ${size + 2} ${size + 2}" width="100%" height="100%" shape-rendering="crispEdges"><rect x="-1" y="-1" width="${size + 2}" height="${size + 2}" fill="#fff"/>${s}${fin(0, 0)}${fin(size - 7, 0)}${fin(0, size - 7)}</svg>`;
}
function houseArt() {
  return `<svg viewBox="0 0 220 120" width="190">${tree(30, 100, 1.4, '#43a047')}${tree(192, 100, 1.4, '#43a047')}<rect x="20" y="104" width="180" height="6" rx="3" fill="#cfe8d0"/><rect x="62" y="56" width="96" height="50" fill="#fde7c8"/><path d="M50 60 L110 18 L170 60Z" fill="#f57c4f"/><rect x="98" y="74" width="24" height="32" fill="#8d5a3b"/><rect x="70" y="68" width="20" height="18" fill="#90caf9" stroke="#fff" stroke-width="2"/><rect x="130" y="68" width="20" height="18" fill="#90caf9" stroke="#fff" stroke-width="2"/><circle cx="168" cy="22" r="17" fill="#16a34a" stroke="#fff" stroke-width="3"/><path d="M160 22 l6 6 l10 -11" stroke="#fff" stroke-width="3.5" fill="none" stroke-linecap="round"/></svg>`;
}
function tripDoneArt() {
  return `<div style="position:relative;width:150px;margin:0 auto">${busSide(150, 'yellow', 'SCHOOL BUS')}<svg viewBox="0 0 40 40" width="40" style="position:absolute;right:-6px;top:-12px"><circle cx="20" cy="20" r="18" fill="#16a34a" stroke="#fff" stroke-width="3"/><path d="M12 20 l6 6 l10 -12" stroke="#fff" stroke-width="3.5" fill="none" stroke-linecap="round"/></svg></div>`;
}
function logoMark(size = 38, c = '#fff') {
  return `<svg width="${size}" height="${size}" viewBox="0 0 40 40" fill="none"><path d="M12 12h14a6 6 0 0 1 0 12H14a6 6 0 0 0 0 12h14" stroke="${c}" stroke-width="3.6" stroke-linecap="round"/><circle cx="12" cy="12" r="4" fill="${c}"/><circle cx="28" cy="36" r="4" fill="${c}"/></svg>`;
}
