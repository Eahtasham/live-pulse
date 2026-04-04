const express = require('express');
const router = express.Router();
const Question = require('../models/Question');
const { eventBus, CHANNELS } = require('../services/eventBus');

// POST /api/questions  – submit a question
router.post('/', async (req, res) => {
  try {
    const { sessionId, text, authorName } = req.body;
    if (!sessionId || !text) {
      return res.status(400).json({ error: 'sessionId and text are required' });
    }

    const question = await Question.create({ sessionId, text, authorName });
    await eventBus.publish(CHANNELS.QUESTION_CREATED, question.toObject());
    res.status(201).json(question);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// GET /api/questions/session/:sessionId  – list questions for a session
router.get('/session/:sessionId', async (req, res) => {
  try {
    const questions = await Question.find({ sessionId: req.params.sessionId }).sort({
      isPinned: -1,
      upvotes: -1,
      createdAt: -1,
    });
    res.json(questions);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// POST /api/questions/:id/upvote  – upvote a question
router.post('/:id/upvote', async (req, res) => {
  try {
    const question = await Question.findByIdAndUpdate(
      req.params.id,
      { $inc: { upvotes: 1 } },
      { new: true }
    );
    if (!question) return res.status(404).json({ error: 'Question not found' });

    await eventBus.publish(CHANNELS.QUESTION_UPVOTED, question.toObject());
    res.json(question);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// PATCH /api/questions/:id/answer  – mark question as answered
router.patch('/:id/answer', async (req, res) => {
  try {
    const question = await Question.findByIdAndUpdate(
      req.params.id,
      { isAnswered: true },
      { new: true }
    );
    if (!question) return res.status(404).json({ error: 'Question not found' });

    await eventBus.publish(CHANNELS.QUESTION_ANSWERED, question.toObject());
    res.json(question);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// PATCH /api/questions/:id/pin  – pin/unpin a question
router.patch('/:id/pin', async (req, res) => {
  try {
    const question = await Question.findById(req.params.id);
    if (!question) return res.status(404).json({ error: 'Question not found' });

    question.isPinned = !question.isPinned;
    await question.save();

    await eventBus.publish(CHANNELS.QUESTION_PINNED, question.toObject());
    res.json(question);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

module.exports = router;
