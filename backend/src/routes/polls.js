const express = require('express');
const router = express.Router();
const Poll = require('../models/Poll');
const { eventBus, CHANNELS } = require('../services/eventBus');

// POST /api/polls  – create a poll in a session
router.post('/', async (req, res) => {
  try {
    const { sessionId, question, options } = req.body;
    if (!sessionId || !question || !Array.isArray(options) || options.length < 2) {
      return res.status(400).json({ error: 'sessionId, question, and at least 2 options are required' });
    }

    const poll = await Poll.create({
      sessionId,
      question,
      options: options.map(text => ({ text })),
    });

    await eventBus.publish(CHANNELS.POLL_CREATED, poll.toObject());
    res.status(201).json(poll);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// GET /api/polls/session/:sessionId  – list polls for a session
router.get('/session/:sessionId', async (req, res) => {
  try {
    const polls = await Poll.find({ sessionId: req.params.sessionId }).sort({ createdAt: 1 });
    res.json(polls);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// GET /api/polls/:id  – get a single poll
router.get('/:id', async (req, res) => {
  try {
    const poll = await Poll.findById(req.params.id);
    if (!poll) return res.status(404).json({ error: 'Poll not found' });
    res.json(poll);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// POST /api/polls/:id/vote  – cast a vote
router.post('/:id/vote', async (req, res) => {
  try {
    const { optionIndex } = req.body;
    if (optionIndex === undefined || optionIndex === null) {
      return res.status(400).json({ error: 'optionIndex is required' });
    }

    const poll = await Poll.findById(req.params.id);
    if (!poll) return res.status(404).json({ error: 'Poll not found' });
    if (!poll.isActive) return res.status(400).json({ error: 'Poll is closed' });
    if (optionIndex < 0 || optionIndex >= poll.options.length) {
      return res.status(400).json({ error: 'Invalid option index' });
    }

    poll.options[optionIndex].votes += 1;
    poll.totalVotes += 1;
    await poll.save();

    await eventBus.publish(CHANNELS.VOTE_CAST, poll.toObject());
    res.json(poll);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// PATCH /api/polls/:id/close  – close a poll
router.patch('/:id/close', async (req, res) => {
  try {
    const poll = await Poll.findByIdAndUpdate(req.params.id, { isActive: false }, { new: true });
    if (!poll) return res.status(404).json({ error: 'Poll not found' });

    await eventBus.publish(CHANNELS.POLL_UPDATED, poll.toObject());
    res.json(poll);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

module.exports = router;
