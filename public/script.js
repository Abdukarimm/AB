const themeToggle = document.getElementById('theme-toggle');
const root = document.documentElement;

function setTheme(theme) {
  if (theme === 'dark') {
    root.setAttribute('data-theme', 'dark');
  } else if (theme === 'light') {
    root.setAttribute('data-theme', 'light');
  } else {
    root.removeAttribute('data-theme');
  }
  localStorage.setItem('theme', theme);
}

const savedTheme = localStorage.getItem('theme');
if (savedTheme) setTheme(savedTheme);

themeToggle.addEventListener('click', () => {
  const current = root.getAttribute('data-theme');
  setTheme(current === 'dark' ? 'light' : 'dark');
});

const languageSelect = document.getElementById('language');
fetch('/languages').then(r => r.json()).then(langs => {
  langs.forEach(l => {
    const opt = document.createElement('option');
    opt.value = l; opt.textContent = l; languageSelect.appendChild(opt);
  });
});

const ws = new WebSocket(`ws://${location.host}/ws`);
ws.onmessage = ev => {
  const resp = JSON.parse(ev.data);
  document.getElementById('output').textContent = resp.output || resp.error || '';
};

document.getElementById('run').addEventListener('click', () => {
  const req = {
    language: languageSelect.value,
    code: document.getElementById('code').value
  };
  ws.send(JSON.stringify(req));
});
