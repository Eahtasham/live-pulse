const express = require('express');
const router = express.Router();
const { randomBytes } = require('crypto');
const Session = require('../models/Session');

/** Generate a cryptographically-random 6-char alphanumeric join code */
function generateCode() {
  // Each byte gives values 0-255; modulo 36 maps to [0-9, a-z]
  const chars = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  return Array.from(randomBytes(6))
    .map(b => chars[b % chars.length])
    .join('');
}

// POST /api/sessions  – create a new session
router.post('/', async (req, res) => {
  try {
    const { title, description, hostId } = req.body;
    if (!title || !hostId) {
      return res.status(400).json({ error: 'title and hostId are required' });
    }

    let code;
    let attempts = 0;
    do {
      code = generateCode();
      attempts++;
      if (attempts > 10) return res.status(500).json({ error: 'Could not generate unique code' });
    } while (await Session.exists({ code }));

    const session = await Session.create({ title, description, hostId, code });
    res.status(201).json(session);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// GET /api/sessions/:id  – get by ObjectId
router.get('/:id', async (req, res) => {
  try {
    const session = await Session.findById(req.params.id);
    if (!session) return res.status(404).json({ error: 'Session not found' });
    res.json(session);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// GET /api/sessions/code/:code  – get by join code
router.get('/code/:code', async (req, res) => {
  try {
    const session = await Session.findOne({ code: req.params.code.toUpperCase() });
    if (!session) return res.status(404).json({ error: 'Session not found' });
    res.json(session);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// PATCH /api/sessions/:id/close  – close a session
router.patch('/:id/close', async (req, res) => {
  try {
    const session = await Session.findByIdAndUpdate(
      req.params.id,
      { isActive: false },
      { new: true }
    );
    if (!session) return res.status(404).json({ error: 'Session not found' });
    res.json(session);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

module.exports = router;
