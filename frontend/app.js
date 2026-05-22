const $ = (id) => document.getElementById(id);
let chats = [];

async function loadStatus() {
  try {
    const r = await fetch('/api/status');
    const s = await r.json();
    const el = $('status');
    if (s.connected && s.loggedIn) {
      el.textContent = 'connected: ' + (s.jid || '');
      el.classList.add('ok');
    } else {
      el.textContent = 'disconnected';
      el.classList.remove('ok');
    }
  } catch (e) {
    $('status').textContent = 'status error';
  }
}

async function loadChats() {
  const r = await fetch('/api/chats');
  if (!r.ok) {
    $('chats').innerHTML = `<li class="empty">error loading chats: ${r.status}</li>`;
    return;
  }
  chats = await r.json();
  render();
}

function render() {
  const q = $('filter').value.trim().toLowerCase();
  const kind = $('kind').value;
  const ul = $('chats');
  const filtered = chats.filter(c => {
    if (kind === 'dm' && !(c.kind === 'dm' || c.kind === 'lid')) return false;
    if (kind === 'group' && c.kind !== 'group') return false;
    if (q && !c.name.toLowerCase().includes(q)) return false;
    return true;
  });
  if (filtered.length === 0) {
    ul.innerHTML = '<li class="empty">no chats match</li>';
    return;
  }
  ul.innerHTML = '';
  for (const c of filtered) {
    const li = document.createElement('li');
    li.className = 'chat';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.checked = c.tracked;
    cb.addEventListener('click', (e) => e.stopPropagation());
    cb.addEventListener('change', () => toggle(c, cb));
    const name = document.createElement('div');
    name.className = 'name';
    name.textContent = c.name;
    const badge = document.createElement('span');
    badge.className = 'badge';
    badge.textContent = c.kind;
    li.append(cb, name, badge);
    li.addEventListener('click', () => { cb.checked = !cb.checked; toggle(c, cb); });
    ul.append(li);
  }
}

async function toggle(c, cb) {
  const tracked = cb.checked;
  try {
    const r = await fetch('/api/tracked', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ jid: c.jid, tracked }),
    });
    if (!r.ok) throw new Error(await r.text());
    c.tracked = tracked;
  } catch (e) {
    cb.checked = !tracked;
    alert('failed: ' + e.message);
  }
}

$('filter').addEventListener('input', render);
$('kind').addEventListener('change', render);

loadStatus();
loadChats();
setInterval(loadStatus, 5000);
