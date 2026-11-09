// Fahrradverleih Frontend – reines JS, spricht nur mit der Go-API (SPEC.md §4).
'use strict';

const state = { meta: null, items: [], persons: [], rentals: [], prices: [], stats: null };
const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

// ---------- Hilfsfunktionen ----------
async function api(method, path, body) {
  const opt = { method, headers: {} };
  if (body instanceof FormData) opt.body = body;
  else if (body !== undefined) { opt.body = JSON.stringify(body); opt.headers['Content-Type'] = 'application/json'; }
  const res = await fetch(path, opt);
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `Fehler ${res.status}`);
  return data;
}
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const euro = (cents) => (cents / 100).toLocaleString('de-AT', { style: 'currency', currency: 'EUR' });
const fmtDate = (iso) => new Date(iso).toLocaleString('de-AT', { weekday: 'short', day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
function fmtHours(h) {
  if (h < 24) return `${h} Std.`;
  const d = Math.floor(h / 24), r = h % 24;
  return `${d} ${d === 1 ? 'Tag' : 'Tage'}${r ? ` ${r} Std.` : ''}`;
}
function toLocalInput(d) {
  const p = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}
function toast(msg, kind = '') {
  const el = document.createElement('div');
  el.className = `toast ${kind}`;
  el.textContent = msg;
  $('#toasts').append(el);
  setTimeout(() => el.remove(), 3500);
}
function closeDialog(btn) { btn.closest('dialog').close(); }
function showFormError(form, msg) { const e = $('.form-error', form); e.textContent = msg; e.hidden = !msg; }
function options(list, selected) { return list.map((v) => `<option ${v === selected ? 'selected' : ''}>${esc(v)}</option>`).join(''); }

const itemLabel = (it) => `${it.brand} ${it.model}`;
const itemById = (id) => state.items.find((i) => i.id === id);
const personById = (id) => state.persons.find((p) => p.id === id);
const personName = (p) => (p ? `${p.firstName} ${p.lastName}` : '–');
const openRentals = () => state.rentals.filter((r) => r.status !== 'abgeschlossen');
function currentRentalFor(itemId) {
  return state.rentals.find((r) => (r.status === 'aktiv' || r.status === 'überfällig') && r.itemIds.includes(itemId));
}

async function loadAll() {
  [state.items, state.persons, state.rentals, state.prices, state.stats] = await Promise.all([
    api('GET', CFG.apiPath), api('GET', '/api/persons'), api('GET', '/api/rentals'),
    api('GET', '/api/prices'), api('GET', '/api/stats'),
  ]);
}

// ---------- Router ----------
const pages = { dashboard: renderDashboard, items: renderItems, persons: renderPersons, rentals: renderRentals, prices: renderPrices };
const ui = { itemQuery: '', itemType: '', personQuery: '', rentalFilter: 'offen' };

async function render() {
  const page = (location.hash.slice(1) || 'dashboard');
  $$('.nav a').forEach((a) => a.classList.toggle('active', a.getAttribute('href') === `#${page}`));
  document.body.classList.remove('nav-open');
  try {
    await loadAll();
    (pages[page] || renderDashboard)();
  } catch (e) {
    $('#app').innerHTML = `<div class="card"><p class="form-error">Backend nicht erreichbar: ${esc(e.message)}</p></div>`;
  }
}
window.addEventListener('hashchange', render);

// ---------- Dashboard (A-1, A-2, A-3) ----------
function renderDashboard() {
  const s = state.stats;
  const open = openRentals().sort((a, b) => {
    const rank = { 'überfällig': 0, aktiv: 1, geplant: 2 };
    return rank[a.status] - rank[b.status] || new Date(a.end) - new Date(b.end);
  });
  $('#app').innerHTML = `
    <section class="card">
      <div class="card-title"><h1>Dashboard</h1><span class="hint">${new Date().toLocaleString('de-AT', { dateStyle: 'full', timeStyle: 'short' })}</span></div>
      <div class="stats">
        <a class="stat g1" href="#items"><div class="label">Verfügbare ${esc(CFG.items)}</div><div class="value">${s.availableNow}</div></a>
        <a class="stat g2" href="#rentals"><div class="label">Ausgeliehene ${esc(CFG.items)}</div><div class="value">${s.rentedNow}</div></a>
        <a class="stat g3" href="#items"><div class="label">Gesamt ${esc(CFG.items)}</div><div class="value">${s.totalItems}</div></a>
        <a class="stat g4" href="#persons"><div class="label">Registrierte Personen</div><div class="value">${s.persons}</div></a>
      </div>
      <div class="mini-stats">
        <span class="mini ${s.overdueRentals ? 'warn' : ''}">Überfällig: <b>${s.overdueRentals}</b></span>
        <span class="mini">Aktive Ausleihen: <b>${s.activeRentals}</b></span>
        <span class="mini">Geplant: <b>${s.plannedRentals}</b></span>
        <span class="mini">Defekt: <b>${s.defective}</b></span>
      </div>
      <h3>Schnellzugriff</h3>
      <div class="quick">
        <a class="btn btn-blue" href="#items">${esc(CFG.items)} verwalten</a>
        <a class="btn btn-blue" href="#persons">Personen verwalten</a>
        <button class="btn btn-green" onclick="openRentalDialog()">Neue Ausleihe</button>
      </div>
    </section>
    <section class="card">
      <div class="card-title"><h1>Offene Ausleihen</h1><a href="#rentals">Alle anzeigen →</a></div>
      ${rentalTable(open, true)}
    </section>`;
}

// ---------- Gegenstände (A-4) ----------
function renderItems() {
  const q = ui.itemQuery.toLowerCase();
  const list = state.items.filter((it) =>
    (!ui.itemType || it.type === ui.itemType) &&
    (!q || `${it.brand} ${it.model} ${it.id}`.toLowerCase().includes(q)));
  $('#app').innerHTML = `
    <section class="card">
      <div class="card-title"><h1>${esc(CFG.itemsNav)}</h1><button class="btn btn-blue" onclick="openItemDialog()">+ ${esc(CFG.item)} hinzufügen</button></div>
      <div class="toolbar">
        <input id="q" placeholder="Suchen (Marke, Modell, Nr.) …" value="${esc(ui.itemQuery)}" class="grow">
        <select id="t"><option value="">Alle Typen</option>${options(state.meta.itemTypes, ui.itemType)}</select>
      </div>
      <div class="table-wrap"><table>
        <thead><tr><th>Nr.</th><th>Marke / Modell</th><th>Typ</th><th>${esc(CFG.size.label)}</th><th>Zustand</th><th>Status</th><th></th></tr></thead>
        <tbody>${list.map((it) => {
          const r = currentRentalFor(it.id);
          const status = it.condition === 'Defekt' ? '' : r
            ? `<span class="badge b-verliehen" title="bis ${esc(fmtDate(r.end))}">verliehen an ${esc(personName(personById(r.personId)))}</span>`
            : '<span class="badge b-frei">frei</span>';
          return `<tr>
            <td>#${it.id}</td><td><strong>${esc(itemLabel(it))}</strong></td><td>${esc(it.type)}</td>
            <td>${esc(CFG.size.format(it[CFG.size.field]))}</td>
            <td><span class="badge b-${esc(it.condition)}">${esc(it.condition)}</span></td><td>${status}</td>
            <td class="actions-cell">
              ${!r && it.condition !== 'Defekt' ? `<button class="btn btn-green btn-sm" onclick="openRentalDialog({itemId:${it.id}})">Verleihen</button>` : ''}
              <button class="btn-icon" title="Bearbeiten" onclick="openItemDialog(${it.id})">✏️</button>
              <button class="btn-icon" title="Löschen" onclick="deleteItem(${it.id})">🗑️</button>
            </td></tr>`;
        }).join('') || `<tr><td colspan="7" class="empty">Keine ${esc(CFG.items)} gefunden.</td></tr>`}</tbody>
      </table></div>
    </section>`;
  $('#q').addEventListener('input', (e) => { ui.itemQuery = e.target.value; rerenderKeepFocus(renderItems, '#q'); });
  $('#t').addEventListener('change', (e) => { ui.itemType = e.target.value; renderItems(); });
}

function rerenderKeepFocus(fn, sel) {
  const pos = $(sel).selectionStart;
  fn();
  const el = $(sel); el.focus(); el.setSelectionRange(pos, pos);
}

function openItemDialog(id) {
  const it = id ? itemById(id) : null;
  const f = $('#itemForm');
  f.reset(); showFormError(f, '');
  $('#itemDialogTitle').textContent = it ? `${CFG.item} #${it.id} bearbeiten` : `${CFG.item} hinzufügen`;
  f.type.innerHTML = options(state.meta.itemTypes, it?.type);
  f.condition.innerHTML = options(state.meta.conditions, it?.condition || 'Gut');
  f.brand.value = it?.brand || '';
  f.model.value = it?.model || '';
  f.size.value = it ? it[CFG.size.field] : (CFG.size.input.default ?? '');
  f.dataset.id = id || '';
  $('#itemDialog').showModal();
}

function setupItemForm() {
  const inp = CFG.size.input;
  $('#sizeLabel').firstChild.textContent = `${CFG.size.label}${CFG.size.unit ? ` (${CFG.size.unit})` : ''}* `;
  $('#sizeInput').innerHTML = inp.type === 'select'
    ? `<select name="size" required>${options(inp.options)}</select>`
    : `<input name="size" type="number" min="${inp.min}" max="${inp.max}" step="${inp.step}" required>`;
  $('#itemForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const f = e.target;
    const body = { brand: f.brand.value, model: f.model.value, type: f.type.value, condition: f.condition.value };
    body[CFG.size.field] = inp.type === 'number' ? Number(f.size.value) : f.size.value;
    try {
      if (f.dataset.id) await api('PUT', `${CFG.apiPath}/${f.dataset.id}`, body);
      else await api('POST', CFG.apiPath, body);
      $('#itemDialog').close();
      toast(`${CFG.item} gespeichert`, 'ok');
      render();
    } catch (err) { showFormError(f, err.message); }
  });
}

async function deleteItem(id) {
  const it = itemById(id);
  if (!confirm(`${CFG.item} „${itemLabel(it)}“ (#${id}) wirklich löschen?`)) return;
  try { await api('DELETE', `${CFG.apiPath}/${id}`); toast(`${CFG.item} gelöscht`, 'ok'); render(); }
  catch (e) { toast(e.message, 'err'); }
}

// ---------- Personen (A-5, A-6, A-7) ----------
function renderPersons() {
  const q = ui.personQuery.toLowerCase();
  const list = state.persons.filter((p) => !q ||
    `${p.firstName} ${p.lastName} ${p.email} ${p.idDocument.number}`.toLowerCase().includes(q));
  $('#app').innerHTML = `
    <section class="card">
      <div class="card-title"><h1>Personen</h1><button class="btn btn-blue" onclick="openPersonDialog()">+ Person hinzufügen</button></div>
      <div class="toolbar"><input id="q" placeholder="Suchen (Name, E-Mail, Ausweisnummer) …" value="${esc(ui.personQuery)}" class="grow"></div>
      <div class="table-wrap"><table>
        <thead><tr><th>Name</th><th>Kontakt</th><th>Ausweis</th><th>Scan</th><th class="num">Offene Ausleihen</th><th></th></tr></thead>
        <tbody>${list.map((p) => {
          const n = openRentals().filter((r) => r.personId === p.id).length;
          const d = p.idDocument;
          return `<tr>
            <td><strong>${esc(personName(p))}</strong></td>
            <td>${esc(p.email)}<br><span class="hint">${esc(p.phone)}</span></td>
            <td>${esc(d.type)}<br><code>${esc(d.number)}</code></td>
            <td>${d.hasFile
              ? `<a class="doc-ok" href="/api/persons/${p.id}/id-document" target="_blank" title="${esc(d.fileName)}">✔ ansehen</a>`
              : `<button class="btn-icon doc-missing" onclick="openPersonDialog(${p.id})" title="Scan hochladen">⚠ fehlt</button>`}</td>
            <td class="num">${n}</td>
            <td class="actions-cell">
              <button class="btn btn-green btn-sm" onclick="openRentalDialog({personId:${p.id}})">Ausleihe</button>
              <button class="btn-icon" title="Bearbeiten" onclick="openPersonDialog(${p.id})">✏️</button>
              <button class="btn-icon" title="Löschen" onclick="deletePerson(${p.id})">🗑️</button>
            </td></tr>`;
        }).join('') || '<tr><td colspan="6" class="empty">Keine Personen gefunden.</td></tr>'}</tbody>
      </table></div>
    </section>`;
  $('#q').addEventListener('input', (e) => { ui.personQuery = e.target.value; rerenderKeepFocus(renderPersons, '#q'); });
}

let personFromRental = false;
function openPersonDialog(id, fromRental = false) {
  personFromRental = fromRental;
  const p = id ? personById(id) : null;
  const f = $('#personForm');
  f.reset(); showFormError(f, '');
  $('#personDialogTitle').textContent = p ? `${personName(p)} bearbeiten` : 'Person hinzufügen';
  f.docType.innerHTML = options(state.meta.docTypes, p?.idDocument.type);
  for (const k of ['firstName', 'lastName', 'email', 'phone']) f[k].value = p?.[k] || '';
  f.docNumber.value = p?.idDocument.number || '';
  f.dataset.id = id || '';
  resetDropzone();
  $('#currentFile').innerHTML = p?.idDocument.hasFile
    ? `Aktueller Scan: <a href="/api/persons/${p.id}/id-document" target="_blank">${esc(p.idDocument.fileName)}</a> – neue Datei ersetzt ihn.`
    : '';
  $('#personDialog').showModal();
}

function resetDropzone() {
  const dz = $('#dropzone');
  $('.dz-preview', dz).hidden = true;
  $('.dz-text', dz).innerHTML = 'Ausweis-Scan hierher ziehen oder <u>auswählen</u><br><small>JPG, PNG oder PDF · max. 5 MB</small>';
}

function showPickedFile(file) {
  const dz = $('#dropzone');
  if (file.size > 5 * 1024 * 1024) { toast('Datei ist größer als 5 MB', 'err'); $('#personForm').file.value = ''; return resetDropzone(); }
  $('.dz-text', dz).innerHTML = `<strong>${esc(file.name)}</strong><br><small>${(file.size / 1024).toFixed(0)} KB · wird beim Speichern hochgeladen</small>`;
  const img = $('.dz-preview', dz);
  if (file.type.startsWith('image/')) { img.src = URL.createObjectURL(file); img.hidden = false; } else img.hidden = true;
}

function setupPersonForm() {
  const f = $('#personForm');
  const dz = $('#dropzone');
  f.file.addEventListener('change', () => f.file.files[0] && showPickedFile(f.file.files[0]));
  dz.addEventListener('dragover', (e) => { e.preventDefault(); dz.classList.add('over'); });
  dz.addEventListener('dragleave', () => dz.classList.remove('over'));
  dz.addEventListener('drop', (e) => {
    e.preventDefault(); dz.classList.remove('over');
    if (e.dataTransfer.files.length) { f.file.files = e.dataTransfer.files; showPickedFile(f.file.files[0]); }
  });
  f.addEventListener('submit', async (e) => {
    e.preventDefault();
    const body = {
      firstName: f.firstName.value, lastName: f.lastName.value, email: f.email.value, phone: f.phone.value,
      idDocument: { type: f.docType.value, number: f.docNumber.value },
    };
    try {
      const p = f.dataset.id ? await api('PUT', `/api/persons/${f.dataset.id}`, body) : await api('POST', '/api/persons', body);
      f.dataset.id = p.id; // falls der Upload scheitert, nicht doppelt anlegen
      if (f.file.files[0]) {
        const fd = new FormData();
        fd.append('file', f.file.files[0]);
        await api('POST', `/api/persons/${p.id}/id-document`, fd);
      }
      $('#personDialog').close();
      toast('Person gespeichert', 'ok');
      if (personFromRental) { state.persons = await api('GET', '/api/persons'); fillPersonList(); pickPerson(p.id); }
      else render();
    } catch (err) { showFormError(f, err.message); }
  });
}

async function deletePerson(id) {
  if (!confirm(`${personName(personById(id))} wirklich löschen?`)) return;
  try { await api('DELETE', `/api/persons/${id}`); toast('Person gelöscht', 'ok'); render(); }
  catch (e) { toast(e.message, 'err'); }
}

// ---------- Ausleihen (A-8 … A-11) ----------
function rentalTable(list, compact = false) {
  if (!list.length) return '<p class="empty">Keine Ausleihen.</p>';
  return `<div class="table-wrap"><table>
    <thead><tr><th>Nr.</th><th>Person</th><th>${esc(CFG.items)}</th><th>Beginn</th><th>Ende</th><th>Dauer</th><th class="num">Preis</th><th>Status</th><th></th></tr></thead>
    <tbody>${list.map((r) => {
      const price = r.finalPriceCents > r.priceCents
        ? `${euro(r.finalPriceCents)}<br><span class="hint">gebucht ${euro(r.priceCents)}</span>` : euro(r.finalPriceCents);
      let action = '';
      if (r.status === 'aktiv' || r.status === 'überfällig') action = `<button class="btn btn-blue btn-sm" onclick="returnRental(${r.id})">Rückgabe</button>`;
      if (r.status === 'geplant') action = `<button class="btn btn-ghost btn-sm" onclick="cancelRental(${r.id})">Stornieren</button>`;
      return `<tr class="${r.status === 'überfällig' ? 'overdue' : ''}">
        <td>#${r.id}</td><td>${esc(personName(personById(r.personId)))}</td>
        <td>${r.itemIds.map((id) => { const it = itemById(id); return esc(it ? itemLabel(it) : `#${id}`); }).join('<br>')}</td>
        <td>${esc(fmtDate(r.start))}</td>
        <td>${esc(fmtDate(r.returnedAt || r.end))}${r.returnedAt ? '<br><span class="hint">zurückgegeben</span>' : ''}</td>
        <td>${fmtHours(r.hours)}</td><td class="num">${price}</td>
        <td><span class="badge b-${esc(r.status)}">${esc(r.status)}</span></td>
        <td class="actions-cell">${action}</td></tr>`;
    }).join('')}</tbody></table></div>`;
}

function renderRentals() {
  const filters = { offen: (r) => r.status !== 'abgeschlossen', 'überfällig': (r) => r.status === 'überfällig', abgeschlossen: (r) => r.status === 'abgeschlossen', alle: () => true };
  const list = state.rentals.filter(filters[ui.rentalFilter]).sort((a, b) => new Date(b.start) - new Date(a.start));
  $('#app').innerHTML = `
    <section class="card">
      <div class="card-title"><h1>Ausleihen</h1><button class="btn btn-green" onclick="openRentalDialog()">+ Neue Ausleihe</button></div>
      <div class="chips" style="margin-bottom:16px">${Object.keys(filters).map((k) =>
        `<button class="chip ${k === ui.rentalFilter ? 'on' : ''}" onclick="ui.rentalFilter='${k}';renderRentals()">${k[0].toUpperCase() + k.slice(1)} (${state.rentals.filter(filters[k]).length})</button>`).join('')}</div>
      ${rentalTable(list)}
    </section>`;
}

async function returnRental(id) {
  try {
    const r = await api('POST', `/api/rentals/${id}/return`);
    const extra = r.finalPriceCents > r.priceCents ? ` – Nachzahlung ${euro(r.finalPriceCents - r.priceCents)} (Überziehung)` : '';
    toast(`Rückgabe erfasst. Endpreis ${euro(r.finalPriceCents)}${extra}`, 'ok');
    render();
  } catch (e) { toast(e.message, 'err'); }
}

async function cancelRental(id) {
  if (!confirm(`Ausleihe #${id} stornieren?`)) return;
  try { await api('DELETE', `/api/rentals/${id}`); toast('Ausleihe storniert', 'ok'); render(); }
  catch (e) { toast(e.message, 'err'); }
}

// ----- Dialog: Neue Ausleihe -----
const rental = { personId: null, picked: new Set(), available: [] };
const DURATIONS = [[1, '1 Std.'], [2, '2 Std.'], [4, '4 Std.'], [24, '1 Tag'], [48, '2 Tage'], [72, '3 Tage'], [168, '1 Woche']];
const personOption = (p) => `${personName(p)} · ${p.idDocument.type} ${p.idDocument.number}`;

function fillPersonList() {
  $('#personList').innerHTML = state.persons.map((p) => `<option value="${esc(personOption(p))}">`).join('');
}
function pickPerson(id) {
  const p = personById(id);
  rental.personId = p ? p.id : null;
  if (p) $('#personSearch').value = personOption(p);
  $('#personPicked').innerHTML = p
    ? `✔ ${esc(personName(p))}${p.idDocument.hasFile ? '' : ' · <span class="doc-missing">Ausweis-Scan fehlt noch</span>'}`
    : '';
}

async function openRentalDialog(pre = {}) {
  const f = $('#rentalForm');
  f.reset(); showFormError(f, '');
  if (!state.meta) return;
  state.persons = await api('GET', '/api/persons');
  state.items = await api('GET', CFG.apiPath);
  fillPersonList();
  rental.picked = new Set(pre.itemId ? [pre.itemId] : []);
  $('#personSearch').value = '';
  pickPerson(pre.personId);
  const now = new Date(); now.setSeconds(0, 0); now.setMinutes(Math.floor(now.getMinutes() / 5) * 5);
  f.start.value = toLocalInput(now);
  f.hours.value = 4;
  $('#itemFilter').value = '';
  $('#itemTypeFilter').value = '';
  $('#rentalDialog').showModal();
  await refreshAvailability();
  if (!rental.personId) $('#personSearch').focus();
}

async function refreshAvailability() {
  const f = $('#rentalForm');
  const hours = Number(f.hours.value);
  $$('#durationChips .chip').forEach((c) => c.classList.toggle('on', Number(c.dataset.h) === hours));
  if (!f.start.value || !(hours >= 1 && hours <= 336)) {
    $('#endInfo').textContent = 'Dauer: 1 Stunde bis 14 Tage.';
    return;
  }
  const start = new Date(f.start.value);
  const end = new Date(start.getTime() + hours * 3600e3);
  $('#endInfo').textContent = `Rückgabe bis ${fmtDate(end.toISOString())} (${fmtHours(hours)})`;
  try {
    rental.available = await api('GET', `${CFG.apiPath}/available?start=${encodeURIComponent(start.toISOString())}&hours=${hours}`);
    const ids = new Set(rental.available.map((i) => i.id));
    for (const id of [...rental.picked]) if (!ids.has(id)) rental.picked.delete(id);
    renderPickList();
  } catch (e) { showFormError(f, e.message); }
}

function renderPickList() {
  const q = $('#itemFilter').value.toLowerCase();
  const t = $('#itemTypeFilter').value;
  const priceOf = (type) => state.prices.find((p) => p.type === type);
  const list = rental.available.filter((it) => (!t || it.type === t) && (!q || itemLabel(it).toLowerCase().includes(q) || rental.picked.has(it.id)));
  $('#pickList').innerHTML = list.map((it) => {
    const pr = priceOf(it.type);
    return `<label class="pick"><input type="checkbox" value="${it.id}" ${rental.picked.has(it.id) ? 'checked' : ''}>
      <span><strong>${esc(itemLabel(it))}</strong> <span class="hint">#${it.id}</span></span>
      <span class="meta">${esc(it.type)} · ${esc(CFG.size.format(it[CFG.size.field]))}<br>${pr ? `${euro(pr.hourlyCents)}/Std. · ${euro(pr.dailyCents)}/Tag` : ''}</span></label>`;
  }).join('') || `<p class="empty">Im gewählten Zeitraum sind keine passenden ${esc(CFG.items)} frei.</p>`;
  updateQuote();
}

let quoteTimer;
function updateQuote() {
  $('#pickCount').textContent = rental.picked.size;
  clearTimeout(quoteTimer);
  const hours = Number($('#rentalForm').hours.value);
  if (!rental.picked.size || !(hours >= 1)) { $('#quote').textContent = '–'; return; }
  quoteTimer = setTimeout(async () => {
    try {
      const q = await api('GET', `/api/rentals/quote?itemIds=${[...rental.picked].join(',')}&hours=${hours}`);
      $('#quote').textContent = euro(q.priceCents);
    } catch { $('#quote').textContent = '–'; }
  }, 150);
}

function setupRentalForm() {
  const f = $('#rentalForm');
  $('#durationChips').innerHTML = DURATIONS.map(([h, l]) => `<button type="button" class="chip" data-h="${h}">${l}</button>`).join('');
  $('#durationChips').addEventListener('click', (e) => {
    if (!e.target.dataset.h) return;
    f.hours.value = e.target.dataset.h;
    refreshAvailability();
  });
  f.start.addEventListener('change', refreshAvailability);
  f.hours.addEventListener('input', refreshAvailability);
  $('#personSearch').addEventListener('input', (e) => {
    const p = state.persons.find((x) => personOption(x) === e.target.value);
    pickPerson(p?.id);
  });
  $('#itemFilter').addEventListener('input', renderPickList);
  $('#itemTypeFilter').addEventListener('change', renderPickList);
  $('#pickList').addEventListener('change', (e) => {
    const id = Number(e.target.value);
    e.target.checked ? rental.picked.add(id) : rental.picked.delete(id);
    updateQuote();
  });
  f.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!rental.personId) return showFormError(f, 'Bitte eine Person auswählen (oder neu anlegen).');
    if (!rental.picked.size) return showFormError(f, `Bitte mindestens ${CFG.oneItem} auswählen.`);
    try {
      const r = await api('POST', '/api/rentals', {
        personId: rental.personId, itemIds: [...rental.picked],
        start: new Date(f.start.value).toISOString(), hours: Number(f.hours.value),
      });
      $('#rentalDialog').close();
      toast(`Ausleihe #${r.id} gebucht – ${euro(r.priceCents)}`, 'ok');
      render();
    } catch (err) { showFormError(f, err.message); refreshAvailability(); }
  });
}

// ---------- Preise (B-2 … B-4) ----------
function calcPrice(rule, hours) { // gleiche Formel wie im Backend (B-3), nur für die Beispielspalte
  const d = Math.floor(hours / 24), r = hours % 24;
  return d * rule.dailyCents + Math.min(r * rule.hourlyCents, rule.dailyCents);
}

function renderPrices() {
  $('#app').innerHTML = `
    <section class="card">
      <div class="card-title"><h1>Preise</h1><span class="hint">Stundenweise Abrechnung mit Tagesdeckel</span></div>
      <p class="hint" style="margin-bottom:18px">Preis pro ${esc(CFG.item)} = volle Tage × Tagespreis + Reststunden × Stundenpreis (höchstens ein Tagespreis).
        Änderungen gelten für neue Ausleihen. Bei verspäteter Rückgabe wird nach tatsächlich angefangenen Stunden nachberechnet.</p>
      <form id="priceForm">
        <div class="table-wrap"><table class="price-table">
          <thead><tr><th>Typ</th><th>Stundenpreis (€)</th><th>Tagespreis (€)</th><th>Beispiele</th></tr></thead>
          <tbody>${state.prices.map((p) => `<tr data-type="${esc(p.type)}">
            <td><strong>${esc(p.type)}</strong></td>
            <td><input type="number" name="h" min="0.01" step="0.01" value="${p.hourlyCents / 100}" required></td>
            <td><input type="number" name="d" min="0.01" step="0.01" value="${p.dailyCents / 100}" required></td>
            <td class="example"></td></tr>`).join('')}</tbody>
        </table></div>
        <div class="actions"><button class="btn btn-blue">Preise speichern</button></div>
      </form>
    </section>`;
  const f = $('#priceForm');
  const readRules = () => $$('tbody tr', f).map((tr) => ({
    type: tr.dataset.type,
    hourlyCents: Math.round(Number($('[name=h]', tr).value) * 100),
    dailyCents: Math.round(Number($('[name=d]', tr).value) * 100),
  }));
  const updateExamples = () => readRules().forEach((rule, i) => {
    $$('.example', f)[i].textContent = [2, 6, 24, 72].map((h) => `${fmtHours(h)}: ${euro(calcPrice(rule, h))}`).join(' · ');
  });
  f.addEventListener('input', updateExamples);
  updateExamples();
  f.addEventListener('submit', async (e) => {
    e.preventDefault();
    try { state.prices = await api('PUT', '/api/prices', readRules()); toast('Preise gespeichert', 'ok'); }
    catch (err) { toast(err.message, 'err'); }
  });
}

// ---------- Start ----------
async function init() {
  document.title = CFG.appName;
  $$('[data-cfg]').forEach((el) => { el.textContent = CFG[el.dataset.cfg]; });
  try { state.meta = await api('GET', '/api/meta'); } catch { /* render() zeigt den Fehler */ }
  if (state.meta) $('#itemTypeFilter').innerHTML += options(state.meta.itemTypes);
  setupItemForm();
  setupPersonForm();
  setupRentalForm();
  render();
}
init();
