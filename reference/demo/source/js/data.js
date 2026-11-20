/* Demo data. Static reference data lives in DATA; anything the user can change lives in seedState(). */
const DATA = (() => {
  const R = rng(20260926);
  const pick = a => a[Math.floor(R() * a.length)];
  const FEMALE = new Set(['Ananya', 'Diya', 'Mira', 'Siya', 'Zara', 'Kavya', 'Priya', 'Neha', 'Sneha', 'Pooja', 'Alka', 'Ritu', 'Isha', 'Riya', 'Tanvi', 'Saanvi', 'Myra', 'Aditi', 'Nisha', 'Kiran', 'Anika', 'Pari', 'Navya', 'Kiara', 'Avni', 'Meera', 'Jiya', 'Prisha', 'Sara', 'Hetal', 'Bhavna', 'Komal', 'Dhara', 'Jinal', 'Nidhi', 'Rupal', 'Payal', 'Mrs.']);
  const BEARD = new Set(['Ramesh Patel', 'Imran Shaikh', 'Ramesh Kumar', 'Imran Khan', 'Vikram Singh', 'Rohit Sharma', 'Ketan Parmar']);
  const OLD = { 'Neha Patel': 'fo' };

  const routeColors = { 'R-01': '#2563eb', 'R-02': '#16a34a', 'R-03': '#8b5cf6', 'R-04': '#f97316', 'R-05': '#ef4444', 'R-06': '#0ea5e9', 'R-07': '#ec4899', 'R-08': '#4f46e5', 'R-09': '#10b981', 'R-10': '#cddc39' };
  const rcol = c => routeColors[c] || ['#2563eb', '#16a34a', '#8b5cf6', '#f97316', '#ef4444', '#0ea5e9', '#ec4899', '#4f46e5', '#10b981', '#eab308'][parseInt(c.slice(2)) % 10];

  /* ---------- Vehicles (52) ---------- */
  const V0 = [
    ['GJ01AB1234', 'Tata Starbus', 42, 'On Road', 124560, 78, 'Ramesh Patel', '98765 43210', 'R-01', 32],
    ['GJ01CD5678', 'Ashok Leyland', 52, 'Delayed', 98430, 35, 'Vikram Singh', '98765 43211', 'R-05', 28],
    ['GJ01EF9012', 'Tata Starbus', 52, 'On Road', 145230, 65, 'Imran Shaikh', '98765 43212', 'R-03', 36],
    ['GJ01GH3456', 'Eicher', 40, 'In Maintenance', 78920, 20, null, null, null, 0],
    ['GJ01IJ7890', 'Tata Ultra', 35, 'On Road', 112480, 82, 'Suresh Kumar', '98765 43213', 'R-02', 30],
    ['GJ01KL1122', 'Mahindra', 30, 'Idle', 86540, 40, 'Manoj Das', '98765 43214', 'R-06', 24],
    ['GJ01MN3344', 'Tata Starbus', 52, 'On Road', 162310, 72, 'Ajay Mehta', '98765 43215', 'R-04', 40],
    ['GJ01OP5566', 'Eicher', 40, 'In Maintenance', 69880, 15, null, null, null, 0],
    ['GJ01QR7788', 'Tata Starbus', 52, 'On Road', 138760, 68, 'Prakash Yadav', '98765 43216', 'R-08', 26],
    ['GJ01ST9900', 'Ashok Leyland', 52, 'On Road', 105640, 55, 'Deepak Joshi', '98765 43217', 'R-07', 34],
  ];
  const FN_M = ['Aarav', 'Vihaan', 'Arjun', 'Kabir', 'Ishaan', 'Reyansh', 'Dhruv', 'Krish', 'Rudra', 'Vivaan', 'Ayaan', 'Atharv', 'Shaurya', 'Om', 'Parth', 'Yash', 'Harsh', 'Jay', 'Dev', 'Rohan', 'Vedant', 'Kian', 'Aditya', 'Neel'];
  const FN_F = ['Ananya', 'Diya', 'Mira', 'Siya', 'Kavya', 'Riya', 'Saanvi', 'Myra', 'Aditi', 'Anika', 'Pari', 'Navya', 'Kiara', 'Avni', 'Meera', 'Jiya', 'Prisha', 'Sara', 'Isha', 'Tanvi', 'Zara'];
  const LN = ['Sharma', 'Patel', 'Mehta', 'Shah', 'Joshi', 'Desai', 'Trivedi', 'Singh', 'Khan', 'Verma', 'Parmar', 'Chauhan', 'Rathod', 'Pandya', 'Bhatt', 'Modi', 'Solanki', 'Vyas', 'Thakkar', 'Rana'];
  const DRV_FN = ['Ramesh', 'Vikram', 'Imran', 'Suresh', 'Manoj', 'Ajay', 'Prakash', 'Deepak', 'Ketan', 'Nitin', 'Mahesh', 'Rakesh', 'Dinesh', 'Jignesh', 'Hitesh', 'Bharat', 'Kamlesh', 'Naresh', 'Paresh', 'Mukesh', 'Rajesh', 'Sanjay', 'Anil', 'Sunil'];
  const models = [['Tata Starbus', 52], ['Tata Starbus', 42], ['Ashok Leyland', 52], ['Eicher', 40], ['Tata Ultra', 35], ['Mahindra', 30], ['Ashok Leyland', 40]];
  const letters = 'ABCDEFGHJKLMNPQRSTUVWXYZ';
  const vehicles = V0.map(v => ({ no: v[0], model: v[1], seats: v[2], status: v[3], km: v[4], fuel: v[5], driver: v[6], phone: v[7], route: v[8], students: v[9], fuelType: 'Diesel', bs: 'BS6' }));
  for (let i = vehicles.length; i < 52; i++) {
    const m = pick(models), no = `GJ01${letters[Math.floor(R() * 24)]}${letters[Math.floor(R() * 24)]}${1000 + Math.floor(R() * 8999)}`;
    const st = i === 51 ? 'Inactive' : i === 40 ? 'In Maintenance' : 'On Road';
    const dn = st === 'On Road' ? `${DRV_FN[i % DRV_FN.length]} ${LN[i % LN.length]}` : null;
    const rt = st === 'On Road' ? `R-${String(11 + (i % 18)).padStart(2, '0')}` : null;
    vehicles.push({ no, model: m[0], seats: m[1], status: st, km: 60000 + Math.floor(R() * 110000), fuel: 25 + Math.floor(R() * 70), driver: dn, phone: dn ? `98765 4${3220 + i}` : null, route: rt, students: rt ? 20 + Math.floor(R() * 22) : 0, fuelType: R() > .9 ? 'CNG' : 'Diesel', bs: 'BS6' });
  }

  /* ---------- Routes (28) ---------- */
  const RT0 = [
    ['R-01', 'North Zone', 'Gandhinagar North', 12, 42, 'GJ01AB1234', 'Tata Starbus', '07:00 AM', '08:10 AM'],
    ['R-02', 'South Zone', 'Koba, Raysan', 14, 38, 'GJ01CD5678', 'Ashok Leyland', '07:05 AM', '08:20 AM'],
    ['R-03', 'East Zone', 'Infocity, Indroda', 11, 36, 'GJ01EF9012', 'Tata Starbus', '06:50 AM', '08:00 AM'],
    ['R-04', 'West Zone', 'Sargasan', 13, 40, 'GJ01GH3456', 'Eicher', '07:10 AM', '08:25 AM'],
    ['R-05', 'Central Zone', 'Gandhinagar City', 16, 32, 'GJ01IJ7890', 'Tata Ultra', '06:45 AM', '07:55 AM'],
    ['R-06', 'Adalaj Route', 'Adalaj', 10, 24, 'GJ01KL1122', 'Mahindra', '07:15 AM', '08:30 AM'],
    ['R-07', 'Koba Route', 'Koba, Gift City', 11, 34, 'GJ01MN3344', 'Tata Starbus', '07:00 AM', '08:15 AM'],
    ['R-08', 'Raysan Route', 'Raysan', 12, 26, 'GJ01OP5566', 'Eicher', '07:05 AM', '08:20 AM'],
    ['R-09', 'Infocity Route', 'Infocity', 15, 38, 'GJ01QR7788', 'Tata Starbus', '06:55 AM', '08:10 AM'],
    ['R-10', 'Indroda Route', 'Indroda, Kudasan', 13, 30, 'GJ01ST9900', 'Ashok Leyland', '07:10 AM', '08:25 AM'],
  ];
  const areas = ['Kudasan', 'Sector 21', 'Sector 7', 'Pethapur', 'Randesan', 'Vavol', 'Urjanagar', 'Sector 30', 'Chiloda', 'Uvarsad', 'Sector 2', 'Bhaijipura', 'Zundal', 'Motera', 'Sabarmati', 'Chandkheda', 'Tapovan', 'Vasna'];
  const routes = RT0.map(r => ({ code: r[0], name: r[1], area: r[2], stops: r[3], students: r[4], vehicle: r[5], model: r[6], start: r[7], end: r[8], status: 'Active', color: rcol(r[0]), school: 'City Plus School' }));
  for (let i = 11; i <= 28; i++) {
    const a = areas[i - 11], v = vehicles[i + 10] || vehicles[i];
    routes.push({ code: `R-${i}`, name: `${a} Route`, area: a, stops: 9 + Math.floor(R() * 8), students: 20 + Math.floor(R() * 22), vehicle: v.no, model: v.model, start: `07:${String(Math.floor(R() * 3) * 5).padStart(2, '0')} AM`, end: `08:${String(10 + Math.floor(R() * 4) * 5)} AM`, status: i > 25 ? 'Inactive' : 'Active', color: rcol(`R-${i}`), school: R() > .5 ? 'City Plus School' : 'Gandhinagar International School' });
  }

  /* ---------- Students (1,245) ---------- */
  const S0 = [
    ['Aarav Sharma', 'ADM001', 'Grade 5', 'R-01', 'City Plus School', 'Rajesh Sharma', '98765 43210', 'Active', 'm'],
    ['Ananya Patel', 'ADM002', 'Grade 8', 'R-03', 'Infocity Stop', 'Kiran Patel', '98765 43211', 'Active', 'f'],
    ['Dev Mehta', 'ADM003', 'Grade 6', 'R-02', 'Koba Circle', 'Sneha Mehta', '98765 43212', 'Active', 'm'],
    ['Diya Shah', 'ADM004', 'Grade 4', 'R-01', 'Gandhinagar North', 'Amit Shah', '98765 43213', 'Active', 'f'],
    ['Krish Patel', 'ADM005', 'Grade 7', 'R-05', 'Sargasan', 'Neha Patel', '98765 43214', 'Active', 'm'],
    ['Mira Joshi', 'ADM006', 'Grade 9', 'R-06', 'Adalaj', 'Rakesh Joshi', '98765 43215', 'Active', 'f'],
    ['Rohan Desai', 'ADM007', 'Grade 5', 'R-04', 'Raysan', 'Pooja Desai', '98765 43216', 'Absent', 'm'],
    ['Siya Trivedi', 'ADM008', 'Grade 3', 'R-07', 'Koba, Gift City', 'Manish Trivedi', '98765 43217', 'Active', 'f'],
    ['Vedant Singh', 'ADM009', 'Grade 8', 'R-02', 'Koba Circle', 'Alka Singh', '98765 43218', 'Active', 'm'],
    ['Zara Khan', 'ADM010', 'Grade 6', 'R-08', 'Infocity', 'Imran Khan', '98765 43219', 'Active', 'f'],
  ];
  const stopsPool = ['Infocity Stop', 'Koba Circle', 'Sargasan', 'Raysan', 'Adalaj', 'Kudasan Stop', 'Gandhinagar North', 'City Mall', 'Sector 21', 'Indroda Circle', 'Gift City Gate', 'Pethapur Chowk'];
  const students = S0.map(s => ({ name: s[0], adm: s[1], cls: s[2], route: s[3], stop: s[4], parent: s[5], phone: s[6], status: s[7], g: s[8] }));
  for (let i = students.length + 1; i <= 1245; i++) {
    const f = R() > .5, fn = f ? pick(FN_F) : pick(FN_M), ln = pick(LN), rc = `R-${String(1 + Math.floor(R() * 10)).padStart(2, '0')}`;
    const pf = R() > .5;
    students.push({ name: `${fn} ${ln}`, adm: `ADM${String(i).padStart(3, '0')}`, cls: `Grade ${1 + Math.floor(R() * 10)}`, route: i > 1180 ? '-' : rc, stop: pick(stopsPool), parent: `${pf ? pick(['Hetal', 'Bhavna', 'Komal', 'Dhara', 'Nidhi', 'Payal']) : pick(['Rajesh', 'Mahesh', 'Hitesh', 'Paresh', 'Nilesh', 'Alpesh'])} ${ln}`, phone: `9${8000 + Math.floor(R() * 1999)} ${10000 + Math.floor(R() * 89999)}`, status: i > 1180 ? 'Inactive' : (R() > .99 ? 'Absent' : 'Active'), g: f ? 'f' : 'm' });
  }

  /* ---------- Drivers (48) ---------- */
  const D0 = [
    ['Ramesh Patel', 'DRV001', '98765 43210', 'GJ01DL1234', 'On Duty', 'GJ01AB1234', 'Tata Starbus'],
    ['Vikram Singh', 'DRV002', '98765 43211', 'GJ01DL5678', 'Off Duty', 'GJ01CD5678', 'Ashok Leyland'],
    ['Imran Shaikh', 'DRV003', '98765 43212', 'GJ01DL9012', 'On Leave', null, null],
    ['Suresh Kumar', 'DRV004', '98765 43213', 'GJ01DL3456', 'On Duty', 'GJ01EF9012', 'Tata Starbus'],
    ['Manoj Das', 'DRV005', '98765 43214', 'GJ01DL7890', 'On Duty', 'GJ01GH3456', 'Eicher'],
    ['Ajay Mehta', 'DRV006', '98765 43215', 'GJ01DL1122', 'Inactive', null, null],
    ['Prakash Yadav', 'DRV007', '98765 43216', 'GJ01DL3344', 'On Duty', 'GJ01IJ7890', 'Tata Ultra'],
    ['Deepak Joshi', 'DRV008', '98765 43217', 'GJ01DL5566', 'On Duty', 'GJ01KL1122', 'Mahindra'],
    ['Ketan Parmar', 'DRV009', '98765 43218', 'GJ01DL7788', 'Off Duty', 'GJ01MN3344', 'Tata Starbus'],
    ['Nitin Chauhan', 'DRV010', '98765 43219', 'GJ01DL9900', 'On Duty', 'GJ01OP5566', 'Eicher'],
  ];
  const drivers = D0.map(d => ({ name: d[0], id: d[1], phone: d[2], lic: d[3], status: d[4], vehicle: d[5], model: d[6] }));
  for (let i = 11; i <= 48; i++) {
    const v = vehicles[i + 2];
    const st = i <= 42 ? (i % 9 === 0 ? 'Off Duty' : 'On Duty') : i <= 45 ? 'On Leave' : 'Inactive';
    drivers.push({ name: `${DRV_FN[i % DRV_FN.length]} ${LN[(i * 7) % LN.length]}`, id: `DRV${String(i).padStart(3, '0')}`, phone: `98765 4${3209 + i}`, lic: `GJ01DL${1000 + Math.floor(R() * 8999)}`, status: st, vehicle: st === 'On Duty' && v ? v.no : null, model: st === 'On Duty' && v ? v.model : null });
  }
  const mmanagers = [['Rajiv Bhatt', 'MM001', 'Active'], ['Sanjay Rathod', 'MM002', 'Active'], ['Hitesh Vyas', 'MM003', 'Active'], ['Kamlesh Solanki', 'MM004', 'Active'], ['Bharat Thakkar', 'MM005', 'Active'], ['Naresh Modi', 'MM006', 'Inactive']].map(m => ({ name: m[0], id: m[1], phone: '98250 ' + (11220 + parseInt(m[1].slice(2))), status: m[2], depot: m[1] === 'MM001' ? 'Main Depot' : 'East Depot' }));
  const admins = Array.from({ length: 12 }, (_, i) => ({ name: ['Super Admin', 'Nirav Shah', 'Rupal Desai', 'Alpesh Patel', 'Jinal Mehta', 'Kunal Joshi', 'Payal Trivedi', 'Harsh Pandya', 'Dhara Bhatt', 'Mitesh Rana', 'Komal Vyas', 'Chirag Modi'][i], id: `ADM-U${String(i + 1).padStart(2, '0')}`, role: i === 0 ? 'Full Administrator' : ['Transport Coordinator', 'School Admin', 'Finance Officer', 'Safety Officer'][i % 4], status: i === 11 ? 'Inactive' : 'Active' }));

  const kindOf = name => {
    if (OLD[name]) return OLD[name];
    const st = studentSet.has(name) || KIDS.has(name);
    const fn = name.split(' ')[0];
    const f = FEMALE.has(fn) || name.startsWith('Mrs');
    if (st) return f ? 'kid-f' : 'kid-m';
    if (f) return 'f';
    return BEARD.has(name) ? 'mb' : 'm';
  };
  const KIDS = new Set(['Aarav Sharma', 'Ananya Sharma', 'Ananya Patel', 'Rohan Desai', 'Diya Shah', 'Kavya Mehta', 'Arjun Verma', 'Diya Patel', 'Ishaan Patel', 'Myra Shah', 'Vihaan Joshi', 'Saanvi Desai', 'Kabir Mehta', 'Riya Trivedi']);
  const studentSet = new Set(students.slice(0, 40).map(s => s.name));

  /* ---------- Maintenance ---------- */
  const serviceSchedule = [
    { v: 'GJ01AB1234', model: 'Tata Starbus', seats: '42 Seater', type: 'Oil Change', last: '15 Aug 2026', next: '15 Sep 2026', due: '19 days', status: 'Scheduled' },
    { v: 'GJ01CD5678', model: 'Ashok Leyland', seats: '52 Seater', type: 'Tire Inspection', last: '10 Jul 2026', next: '10 Sep 2026', due: 'Overdue', status: 'Overdue' },
    { v: 'GJ01EF9012', model: 'Tata Starbus', seats: '52 Seater', type: 'Brake Inspection', last: '02 Aug 2026', next: '02 Oct 2026', due: '6 days', status: 'Due Soon' },
    { v: 'GJ01GH3456', model: 'Eicher', seats: '40 Seater', type: 'Engine Service', last: '18 Jun 2026', next: '18 Sep 2026', due: 'Overdue', status: 'Overdue' },
    { v: 'GJ01IJ7890', model: 'Tata Ultra', seats: '35 Seater', type: 'AC Service', last: '22 Aug 2026', next: '22 Oct 2026', due: '26 days', status: 'Scheduled' },
    { v: 'GJ01KL1122', model: 'Mahindra', seats: '30 Seater', type: 'General Checkup', last: '01 Sep 2026', next: '01 Sep 2026', due: 'Today', status: 'Due Today' },
  ];
  const mmVehicles = [
    { no: 'GJ01EF9012', model: 'Tata Starbus (52 Seater)', status: 'Active', last: '12 Aug 2026', km: '42,580 km' },
    { no: 'GJ01EF9013', model: 'Tata Starbus (52 Seater)', status: 'In Service', last: '05 Jul 2026', km: '38,220 km' },
    { no: 'GJ01EF9014', model: 'Ashok Leyland (40 Seater)', status: 'Active', last: '20 Aug 2026', km: '55,120 km' },
    { no: 'GJ01EF9015', model: 'Tata Starbus (52 Seater)', status: 'Active', last: '02 Sep 2026', km: '48,760 km' },
    { no: 'GJ01EF9016', model: 'Eicher (40 Seater)', status: 'Out of Service', last: '15 Jun 2026', km: '61,230 km' },
  ];
  for (let i = 0; i < 19; i++) mmVehicles.push({ no: `GJ01EF${9017 + i}`, model: pick(['Tata Starbus (52 Seater)', 'Ashok Leyland (40 Seater)', 'Eicher (40 Seater)', 'Tata Ultra (35 Seater)']), status: i < 16 ? 'Active' : 'In Service', last: `${String(1 + Math.floor(R() * 27)).padStart(2, '0')} ${pick(['Jun', 'Jul', 'Aug', 'Sep'])} 2026`, km: `${(30 + Math.floor(R() * 40))},${String(Math.floor(R() * 999)).padStart(3, '0')} km` });
  const vehSchedule = [
    { t: 'Engine Oil Change', i: '5,000 km / 3 months', d: '12 Oct 2026', st: 'Due Soon', km: '42,580 km', ic: 'oil', c: '#f59e0b' },
    { t: 'Brake Inspection', i: '10,000 km / 6 months', d: '15 Nov 2026', st: 'Scheduled', km: '42,580 km', ic: 'disc', c: '#ef4444' },
    { t: 'Tire Rotation', i: '10,000 km / 6 months', d: '15 Nov 2026', st: 'Scheduled', km: '42,580 km', ic: 'tyre', c: '#2563eb' },
    { t: 'General Service', i: '20,000 km / 12 months', d: '12 Feb 2027', st: 'Scheduled', km: '42,580 km', ic: 'tools', c: '#f97316' },
    { t: 'Emission Check', i: '12 months', d: '12 Aug 2026', st: 'Overdue', km: '42,580 km', ic: 'leaf', c: '#16a34a' },
  ];
  const vehCompleted = [
    { t: 'General Service', i: '5,000 km service', d: '12 Aug 2026', st: 'Completed', km: '42,580 km', ic: 'tools', c: '#2563eb' },
    { t: 'Brake Repair', i: 'Brake pad replacement', d: '05 Jul 2026', st: 'Completed', km: '38,220 km', ic: 'disc', c: '#ef4444' },
    { t: 'Tire Replacement', i: '2 tyres replaced', d: '12 Jun 2026', st: 'Completed', km: '35,140 km', ic: 'tyre', c: '#2563eb' },
  ];
  const maintHistory = [
    { d: '12 Aug 2026', t: 'General Service', wo: 'WO-2026-014', desc: '5,000 km service completed', cost: 12500, cat: 'Service', ic: 'tools', c: '#2563eb', bg: '#eaf1ff' },
    { d: '05 Jul 2026', t: 'Brake Repair', wo: 'WO-2026-011', desc: 'Brake pad replacement', cost: 8300, cat: 'Repair', ic: 'disc', c: '#ef4444', bg: '#fde8e8' },
    { d: '12 Jun 2026', t: 'Tire Replacement', wo: 'WO-2026-009', desc: '2 tyres replaced', cost: 18000, cat: 'Repair', ic: 'tyre', c: '#2563eb', bg: '#eaf1ff' },
    { d: '20 Apr 2026', t: 'Routine Inspection', wo: 'WO-2026-006', desc: 'Quarterly inspection', cost: 2500, cat: 'Inspection', ic: 'clipcheck', c: '#7c3aed', bg: '#f1ecff' },
    { d: '15 Jan 2026', t: 'Engine Service', wo: 'WO-2026-001', desc: '10,000 km service', cost: 14200, cat: 'Service', ic: 'oil', c: '#f59e0b', bg: '#fff3df' },
  ];
  const parts = [
    { name: 'Engine Oil 15W-40', code: 'OIL-001', cat: 'Engine', qty: 120, reorder: 50, cost: 450, art: 'oil', brand: 'Castrol', unit: 'Liter', supplier: 'ABC Auto Parts', desc: 'Engine oil 15W-40 for diesel engines', compat: 'Tata Starbus, Ashok Leyland' },
    { name: 'Brake Pad Set', code: 'BRK-002', cat: 'Brakes', qty: 8, reorder: 20, cost: 2500, art: 'pad', brand: 'Brembo', unit: 'Set', supplier: 'MotorCare Pvt Ltd', desc: 'Front brake pad set', compat: 'Tata Starbus' },
    { name: 'Air Filter', code: 'FLT-003', cat: 'Filters', qty: 0, reorder: 10, cost: 800, art: 'filter', brand: 'Bosch', unit: 'Piece', supplier: 'Auto Supply Co.', desc: 'Engine air filter element', compat: 'All vehicles' },
    { name: 'Fuel Filter', code: 'FLT-004', cat: 'Filters', qty: 25, reorder: 10, cost: 600, art: 'fuelf', brand: 'Bosch', unit: 'Piece', supplier: 'Auto Supply Co.', desc: 'Diesel fuel filter', compat: 'All vehicles' },
    { name: 'Tyre 295/80R22.5', code: 'TYR-005', cat: 'Tyres', qty: 12, reorder: 8, cost: 18000, art: 'tyre', brand: 'MRF', unit: 'Piece', supplier: 'Reliable Parts', desc: 'Radial tubeless tyre', compat: 'Tata Starbus, Ashok Leyland' },
    { name: 'Headlight Bulb', code: 'LGT-006', cat: 'Electrical', qty: 5, reorder: 10, cost: 350, art: 'bulb', brand: 'Philips', unit: 'Piece', supplier: 'ABC Auto Parts', desc: 'H4 halogen headlight bulb', compat: 'All vehicles' },
    { name: 'Battery 12V 150Ah', code: 'BAT-007', cat: 'Electrical', qty: 4, reorder: 5, cost: 12500, art: 'battery', brand: 'Exide', unit: 'Piece', supplier: 'MotorCare Pvt Ltd', desc: 'Heavy duty battery', compat: 'All vehicles' },
    { name: 'Wiper Blade 24"', code: 'WIP-008', cat: 'Body', qty: 32, reorder: 10, cost: 300, art: 'wiper', brand: 'Bosch', unit: 'Piece', supplier: 'Reliable Parts', desc: '24 inch wiper blade', compat: 'All vehicles' },
  ];
  const purchases = [
    ['26 Sep 2026', 'ABC Auto Parts', 'SUP-2026-014', 50, 450], ['12 Aug 2026', 'MotorCare Pvt Ltd', 'SUP-2026-011', 100, 440], ['15 Jun 2026', 'Auto Supply Co.', 'SUP-2026-008', 50, 430], ['20 Apr 2026', 'ABC Auto Parts', 'SUP-2026-005', 100, 425], ['10 Feb 2026', 'Reliable Parts', 'SUP-2026-002', 50, 420],
  ];
  const suppliers = [
    { name: 'ABC Auto Parts', type: 'Auto Parts Supplier', phone: '+91 98765 43210', email: 'abc@autoparts.in' },
    { name: 'MotorCare Pvt Ltd', type: 'Vehicle Parts Distributor', phone: '+91 98765 11111', email: 'sales@motorcare.in' },
    { name: 'Auto Supply Co.', type: 'Spare Parts Dealer', phone: '+91 98765 22222', email: 'info@autosupply.in' },
    { name: 'Reliable Parts', type: 'Industrial & Auto Parts', phone: '+91 98765 33333', email: 'contact@reliableparts.in', city: 'Rajkot, Gujarat' },
  ];
  const documents = [
    { name: 'Service_Invoice_INV-2026-014.pdf', type: 'Service Invoice', v: 'GJ01EF9012', wo: 'WO-2026-014', d: '26 Sep 2026', by: 'Ramesh Patel', kind: 'pdf', view: 'service' },
    { name: 'Repair_Invoice_INV-2026-011.pdf', type: 'Repair Invoice', v: 'GJ01EF9012', wo: 'WO-2026-011', d: '12 Sep 2026', by: 'Ramesh Patel', kind: 'pdf', view: 'repair' },
    { name: 'Warranty_Certificate.pdf', type: 'Warranty', v: 'GJ01EF9013', wo: '-', d: '05 Aug 2026', by: 'Admin', kind: 'pdf', view: 'warranty' },
    { name: 'Inspection_Report_IR-2026-022.pdf', type: 'Inspection Report', v: 'GJ01EF9012', wo: 'WO-2026-014', d: '26 Sep 2026', by: 'Ramesh Patel', kind: 'pdfb', view: 'inspection' },
    { name: 'Engine_Before.jpg', type: 'Photos', v: 'GJ01EF9012', wo: 'WO-2026-014', d: '26 Sep 2026', by: 'Ramesh Patel', kind: 'img', view: 'photos' },
    { name: 'Engine_After.jpg', type: 'Photos', v: 'GJ01EF9012', wo: 'WO-2026-014', d: '26 Sep 2026', by: 'Ramesh Patel', kind: 'img', view: 'photos' },
    { name: 'Tyre_Condition.jpg', type: 'Photos', v: 'GJ01EF9012', wo: 'WO-2026-034', d: '26 Sep 2026', by: 'Ramesh Patel', kind: 'img', view: 'photos' },
  ];

  /* ---------- Driver app (mobile) ---------- */
  const drvStops = [
    { name: 'Depot', time: '07:05 AM', exp: 0 }, { name: 'Infocity Stop', time: '07:12 AM', exp: 12 }, { name: 'Raysan Stop', time: '07:20 AM', exp: 8 },
    { name: 'Koba Circle', time: '07:28 AM', exp: 6 }, { name: 'Kudasan Stop', time: '07:35 AM', exp: 10 }, { name: 'City Plus School', time: '08:00 AM', exp: 0 },
  ];
  const boardKids = [
    ['S001', 'Aarav Sharma', 'Grade 5'], ['S002', 'Ananya Patel', 'Grade 4'], ['S003', 'Rohan Desai', 'Grade 6'], ['S004', 'Diya Shah', 'Grade 3'], ['S005', 'Kavya Mehta', 'Grade 5'], ['S006', 'Arjun Verma', 'Grade 4'],
    ['S007', 'Ishaan Patel', 'Grade 6'], ['S008', 'Myra Shah', 'Grade 2'], ['S009', 'Vihaan Joshi', 'Grade 7'], ['S010', 'Saanvi Desai', 'Grade 3'], ['S011', 'Kabir Mehta', 'Grade 5'], ['S012', 'Riya Trivedi', 'Grade 4'],
  ].map(k => ({ id: k[0], name: k[1], cls: k[2] }));
  const inspectionItems = [
    { k: 'tyres', t: 'Tyres', s: 'Check condition and pressure', ic: 'tyre', c: '#0f766e', scene: 'tyre', lead: 'Check condition, pressure and visible damage.', checks: ['Tyre condition (no cracks/cuts)', 'Adequate tread depth', 'Correct air pressure', 'No bulges or deformities', 'All tyres including spare checked'], photos: ['tyre', 'brake', 'tyre'] },
    { k: 'brakes', t: 'Brakes', s: 'Test brake response', ic: 'disc', c: '#1e3a8a', scene: 'brake', lead: 'Test brake response and check for leaks.', checks: ['Service brakes working', 'Parking brake working', 'No air/leakage issues', 'Brake fluid levels (if applicable)', 'No unusual noise'], photos: ['brake', 'brakepad'] },
    { k: 'lights', t: 'Lights', s: 'Headlights, indicators, brake lights', ic: 'sun', c: '#1e3a8a', scene: 'lights', lead: 'Check all lights and indicators.', checks: ['Headlights (Low/High beam)', 'Indicators (Front/Rear)', 'Brake lights', 'Reverse lights', 'Hazard lights', 'Interior lights', 'Number plate light'], photos: ['lights', 'lights', 'lights'] },
    { k: 'emkit', t: 'Emergency Kit', s: 'Check availability', ic: 'camera', c: '#1e3a8a', scene: 'emkit', lead: 'Check availability and condition.', checks: ['Reflective triangle', 'Warning cones (if applicable)', 'Tool kit', 'Torch/flashlight', 'Towing cable/rope', 'Jumper cable', 'Other emergency items'], photos: ['emkit', 'emkit', 'emkit'] },
    { k: 'fire', t: 'Fire Extinguisher', s: 'Check expiry date', ic: 'extinguisher', c: '#dc2626', scene: 'extinguisher', lead: 'Check availability, expiry and condition.', checks: ['Present and properly mounted', 'Expiry date valid|Expires: Dec 2026', 'Pressure gauge in green zone', 'Seal intact', 'No physical damage'], photos: ['extinguisher', 'extinguisher', 'extinguisher', 'extinguisher'] },
    { k: 'firstaid', t: 'First Aid', s: 'Check availability', ic: 'firstaid', c: '#dc2626', scene: 'firstaid', lead: 'Check availability and expiry.', checks: ['First aid box present', 'Core items available (bandages, antiseptic)', 'Expiry dates valid|Checked: Sep 2026', 'No missing critical items', 'Box in good condition'], photos: ['firstaid', 'firstaid', 'firstaid'] },
    { k: 'other', t: 'Other Checks', s: 'Cleanliness, mirrors, wipers etc.', ic: 'clip', c: '#1e3a8a', scene: 'interior', lead: 'Configurable checks.', checks: [], photos: [] },
  ];
  const otherChecks = [['Windshield & Wipers', 'No cracks, wipers working', 'windshield'], ['Mirrors', 'Clean and properly adjusted', 'mirror'], ['Doors', 'Open/close properly', 'door'], ['Seat Belts (if applicable)', 'Working condition', 'seat'], ['Horn', 'Working properly', 'horn'], ['GPS/Tracking Device', 'Functioning', 'radio'], ['Cleanliness', 'Interior and exterior clean', 'sparkle']];

  /* ---------- Driver portal (desktop) ---------- */
  const portalStops = [
    ['Shanti Nagar', '07:00 AM', '23.0221, 72.5714', '5 / 5'], ['Vishwakarma Society', '07:08 AM', '23.0245, 72.5721', '3 / 3'], ['Riverdale Apartments', '07:15 AM', '23.0281, 72.5730', '4 / 4'],
    ['Sunrise Park', '07:22 AM', '23.0321, 72.5756', '6 / 6'], ['City Mall Stop', '07:30 AM', '23.0345, 72.5782', '8 / 8'], ['Green Valley', '07:36 AM', '23.0372, 72.5801', '2 / 2'],
    ['Sector 21 Circle', '07:42 AM', '23.0398, 72.5826', '3 / 3'], ['Pathika Ashram', '07:47 AM', '23.0421, 72.5843', '2 / 2'], ['Ghh Circle', '07:52 AM', '23.0449, 72.5867', '1 / 1'],
    ['Sector 11', '07:57 AM', '23.0472, 72.5889', '2 / 2'], ['Mahatma Mandir', '08:03 AM', '23.0498, 72.5904', '2 / 2'], ['City Plus School', '08:35 AM', '23.0521, 72.5932', '0 / 0'],
  ];

  /* ---------- Parent app ---------- */
  const children = [
    { name: 'Aarav Sharma', cls: 'Grade 5 (Class V)', school: 'Greenwood International School', route: 'R-03', routeName: 'R-03 (East Zone)', bus: 'BUS-12', busModel: 'Tata Starbus (52 Seater)', stop: 'Maple Street', stopFull: 'Maple Street (Stop 4)', sid: 'STU00125', dob: '12 Mar 2014', gender: 'Male', roll: '25', adm: 'AD202400125', blood: 'B+', allergies: 'Peanuts (Mild)', med: 'None' },
    { name: 'Ananya Sharma', cls: 'Grade 2 (Class II)', school: 'Greenwood International School', route: 'R-05', routeName: 'R-05 (Central Zone)', bus: 'BUS-08', busModel: 'Tata Starbus (42 Seater)', stop: 'Lake View', stopFull: 'Lake View (Stop 5)', sid: 'STU00211', dob: '04 Jul 2018', gender: 'Female', roll: '12', adm: 'AD202400211', blood: 'O+', allergies: 'None', med: 'None' },
  ];
  const pStops = [
    { t: '07:10 AM', n: 'Trip Started', s: 'Depot - East Zone' }, { t: '07:18 AM', n: 'Sunrise Apartments', s: '2 students boarded' }, { t: '07:28 AM', n: 'Maple Street', s: 'Next Stop' },
    { t: '07:35 AM', n: 'Lake View Society', s: '5 students' }, { t: '07:42 AM', n: 'Riverside Park', s: '3 students' }, { t: '07:50 AM', n: 'City Mall', s: '4 students' },
    { t: '08:05 AM', n: 'Greenwood International School', s: '12 students|(Estimated Arrival)' },
  ];

  return { vehicles, routes, students, drivers, mmanagers, admins, kindOf, rcol, serviceSchedule, mmVehicles, vehSchedule, vehCompleted, maintHistory, parts, purchases, suppliers, documents, drvStops, boardKids, inspectionItems, otherChecks, portalStops, children, pStops };
})();

function seedState() {
  return {
    v: 3,
    lang: 'English',
    settings: { notif: true, sms: true, lowstock: 10, overdue: true },
    added: { vehicles: [], students: [], drivers: [], routes: [] },
    alertsRead: [],
    notifRead: { admin: false, maint: false, driver: false, parent: false },
    admin: {
      incidents: [
        { time: '09:12 AM', type: 'Near Miss', desc: 'Vehicle sudden brake near school', route: 'R-03', status: 'Open' },
        { time: '08:45 AM', type: 'Minor Injury', desc: 'Student slipped while boarding', route: 'R-01', status: 'In Progress' },
        { time: '07:32 AM', type: 'SOS Alert', desc: 'Student pressed SOS button', route: 'R-05', status: 'Resolved' },
        { time: '06:55 AM', type: 'Harassment', desc: 'Misbehavior reported', route: 'R-02', status: 'Open' },
        { time: '06:30 AM', type: 'Medical', desc: 'Student felt unwell', route: 'R-04', status: 'Resolved' },
      ],
      sos: [
        { name: 'Diya Patel', cls: 'Grade 3', route: 'R-05 | Seat 12', status: 'Active', time: '07:32 AM' },
        { name: 'Aarav Sharma', cls: 'Grade 5', route: 'R-02 | Seat 08', status: 'Acknowledged', time: '08:10 AM' },
      ],
      delays: [
        { time: '07:42 AM', v: 'GJ01EF9012', r: 'R-03', issue: 'Traffic Delay', delay: '+12 min', act: 'Take Action', done: false },
        { time: '07:38 AM', v: 'GJ01MN3344', r: 'R-07', issue: 'Route Deviation', delay: '-', act: 'View on Map', done: false },
        { time: '07:25 AM', v: 'GJ01CD5678', r: 'R-02', issue: 'Late Departure', delay: '+5 min', act: 'Contact Driver', done: false },
      ],
      fees: {},
      audit: [],
    },
    maint: {
      workOrders: [
        { id: 'WO-2026-014', type: 'Preventive Maintenance', v: 'GJ01EF9012', model: 'Tata Starbus', desc: 'Engine oil and filter change as per 5,000 km schedule.', priority: 'Normal', by: 'Maintenance Manager', on: '26 Sep 2026, 09:15 AM', exp: '12 Oct 2026', to: 'Ramesh Patel (Mechanic)', step: 3, status: 'In Progress', comments: [] },
        { id: 'WO-2026-013', type: 'Repair', v: 'GJ01CD5678', model: 'Ashok Leyland', desc: 'Tyre inspection and replacement of worn rear tyre.', priority: 'High', by: 'Maintenance Manager', on: '24 Sep 2026, 11:20 AM', exp: '28 Sep 2026', to: 'Suresh Mistry (Mechanic)', step: 5, status: 'In Progress', comments: [] },
        { id: 'WO-2026-012', type: 'Breakdown', v: 'GJ01GH3456', model: 'Eicher', desc: 'Engine overheating, coolant leak suspected.', priority: 'Critical', by: 'Maintenance Manager', on: '22 Sep 2026, 08:05 AM', exp: '27 Sep 2026', to: 'Ramesh Patel (Mechanic)', step: 1, status: 'Pending', comments: [] },
        { id: 'WO-2026-011', type: 'Repair', v: 'GJ01EF9012', model: 'Tata Starbus', desc: 'Brake pads and discs replaced.', priority: 'Normal', by: 'Maintenance Manager', on: '10 Sep 2026, 10:00 AM', exp: '12 Sep 2026', to: 'Ramesh Patel (Mechanic)', step: 7, status: 'Closed', comments: [] },
      ],
      partQty: {}, consumption: [], extraPurchases: [], extraSuppliers: [], extraDocs: [], schedule: [], extraHistory: [],
      inspection: { 'Brake fluid level': false, 'Brake fluid leaks': false },
      serviceDone: [],
    },
    driver: {
      phase: 'assigned', photo: false, attendance: true, insp: {}, other: {}, issue: null, inspDone: false, stop: 1,
      board: {}, boardTime: {}, noShowReason: {}, stopsDone: [], incidents: [
        { id: 'INC-2026-013', type: 'Breakdown', loc: 'R-03 East Zone', time: '26 Sep 2026, 09:15 AM', status: 'In Progress', desc: 'Engine overheated. Vehicle stopped on the side of the road.' },
        { id: 'INC-2026-012', type: 'Minor Accident', loc: 'NH-8 Highway', time: '25 Sep 2026, 07:40 AM', status: 'Resolved', desc: 'Minor collision with another vehicle. No major injuries.' },
        { id: 'INC-2026-011', type: 'Student Issue', loc: 'City Plus School', time: '24 Sep 2026, 04:20 PM', status: 'Resolved', desc: 'Student was not following instructions at the stop.' },
        { id: 'INC-2026-010', type: 'Traffic / Route Issue', loc: 'R-03 East Zone', time: '23 Sep 2026, 08:10 AM', status: 'Resolved', desc: 'Heavy traffic on NH-8.' },
        { id: 'INC-2026-009', type: 'Medical Emergency', loc: 'Kudasan Stop', time: '22 Sep 2026, 11:30 AM', status: 'Resolved', desc: 'Student felt dizzy, first aid given.' },
      ],
      draft: { photos: [], notes: '' }, msgs: {}, tripNotes: '', portalStep: 6, portalStarted: false, portalCompleted: false,
    },
    parent: {
      child: 0, homeConfirmed: null, requests: [], authorized: [
        { name: 'Priya Sharma', rel: 'Mother', phone: '+91 98765 43210', primary: true },
        { name: 'Rohit Sharma', rel: 'Father', phone: '+91 98765 43211' },
        { name: 'Neha Patel', rel: 'Grandmother', phone: '+91 98765 43212' },
      ], notifRead: [], msgs: {}, reported: [], tracking: 0, confirmStart: null,
    },
  };
}
