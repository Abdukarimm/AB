const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const PORT = process.env.PORT || 3000;

app.use(bodyParser.json());
app.use(express.static(__dirname)); // serve index.html

let tasks = [];
let idCounter = 1;

app.get('/api/tasks', (req, res) => {
  res.json(tasks);
});

app.post('/api/tasks', (req, res) => {
  const text = (req.body.text || '').trim();
  if (!text) return res.status(400).json({ error: 'Task text required' });
  const task = { id: idCounter++, text };
  tasks.push(task);
  res.json(task);
});

app.delete('/api/tasks/:id', (req, res) => {
  const id = parseInt(req.params.id, 10);
  tasks = tasks.filter(t => t.id !== id);
  res.json({});
});

app.listen(PORT, () => {
  console.log(`Server listening on port ${PORT}`);
});
