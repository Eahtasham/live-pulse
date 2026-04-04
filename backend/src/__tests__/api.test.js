/**
 * Unit tests for the REST API routes using jest mocks.
 * No real database or Redis connection is needed.
 */

const request = require('supertest');
const express = require('express');

// ── Mock Mongoose models ──────────────────────────────────────────────────────
jest.mock('../models/Session');
jest.mock('../models/Poll');
jest.mock('../models/Question');

// ── Mock EventBus so no Redis is needed ──────────────────────────────────────
jest.mock('../services/eventBus', () => ({
  eventBus: { publish: jest.fn().mockResolvedValue(undefined) },
  CHANNELS: {
    POLL_CREATED: 'poll:created',
    POLL_UPDATED: 'poll:updated',
    VOTE_CAST: 'poll:vote',
    QUESTION_CREATED: 'question:created',
    QUESTION_UPVOTED: 'question:upvoted',
    QUESTION_ANSWERED: 'question:answered',
    QUESTION_PINNED: 'question:pinned',
  },
}));

const Session = require('../models/Session');
const Poll = require('../models/Poll');
const Question = require('../models/Question');

const sessionRoutes = require('../routes/sessions');
const pollRoutes = require('../routes/polls');
const questionRoutes = require('../routes/questions');

const app = express();
app.use(express.json());
app.use('/api/sessions', sessionRoutes);
app.use('/api/polls', pollRoutes);
app.use('/api/questions', questionRoutes);

afterEach(() => jest.clearAllMocks());

// ─── Sessions ────────────────────────────────────────────────────────────────

describe('POST /api/sessions', () => {
  it('creates a session and returns 201', async () => {
    Session.exists.mockResolvedValue(null);
    const created = { _id: 'sid1', title: 'Test Session', code: 'ABC123', isActive: true };
    Session.create.mockResolvedValue(created);

    const res = await request(app).post('/api/sessions').send({
      title: 'Test Session',
      description: 'A test',
      hostId: 'host-001',
    });
    expect(res.status).toBe(201);
    expect(res.body.title).toBe('Test Session');
    expect(res.body.isActive).toBe(true);
  });

  it('returns 400 when title is missing', async () => {
    const res = await request(app).post('/api/sessions').send({ hostId: 'host-001' });
    expect(res.status).toBe(400);
  });
});

describe('GET /api/sessions/code/:code', () => {
  it('returns session by join code', async () => {
    const session = { _id: 'sid2', code: 'MYCODE', title: 'Coded' };
    Session.findOne.mockResolvedValue(session);

    const res = await request(app).get('/api/sessions/code/MYCODE');
    expect(res.status).toBe(200);
    expect(res.body._id).toBe('sid2');
  });

  it('returns 404 for unknown code', async () => {
    Session.findOne.mockResolvedValue(null);
    const res = await request(app).get('/api/sessions/code/XXXXXX');
    expect(res.status).toBe(404);
  });
});

// ─── Polls ───────────────────────────────────────────────────────────────────

describe('POST /api/polls', () => {
  it('creates a poll with options', async () => {
    const created = {
      _id: 'p1',
      sessionId: 'sid1',
      question: 'Favourite colour?',
      options: [{ text: 'Red', votes: 0 }, { text: 'Blue', votes: 0 }, { text: 'Green', votes: 0 }],
      totalVotes: 0,
      toObject: function () { return this; },
    };
    Poll.create.mockResolvedValue(created);

    const res = await request(app).post('/api/polls').send({
      sessionId: 'sid1',
      question: 'Favourite colour?',
      options: ['Red', 'Blue', 'Green'],
    });
    expect(res.status).toBe(201);
    expect(res.body.options).toHaveLength(3);
    expect(res.body.totalVotes).toBe(0);
  });

  it('returns 400 when fewer than 2 options given', async () => {
    const res = await request(app).post('/api/polls').send({
      sessionId: 'sid1',
      question: 'Single option?',
      options: ['Only one'],
    });
    expect(res.status).toBe(400);
  });
});

describe('POST /api/polls/:id/vote', () => {
  it('increments vote count for chosen option', async () => {
    const pollDoc = {
      _id: 'p1',
      isActive: true,
      options: [{ text: 'Yes', votes: 0 }, { text: 'No', votes: 0 }],
      totalVotes: 0,
      save: jest.fn().mockResolvedValue(true),
      toObject: function () { return { _id: this._id, options: this.options, totalVotes: this.totalVotes }; },
    };
    Poll.findById.mockResolvedValue(pollDoc);

    const res = await request(app).post('/api/polls/p1/vote').send({ optionIndex: 0 });
    expect(res.status).toBe(200);
    expect(res.body.options[0].votes).toBe(1);
    expect(res.body.totalVotes).toBe(1);
  });

  it('returns 400 for an invalid option index', async () => {
    const pollDoc = {
      _id: 'p1',
      isActive: true,
      options: [{ text: 'A', votes: 0 }, { text: 'B', votes: 0 }],
      totalVotes: 0,
      save: jest.fn(),
    };
    Poll.findById.mockResolvedValue(pollDoc);

    const res = await request(app).post('/api/polls/p1/vote').send({ optionIndex: 99 });
    expect(res.status).toBe(400);
  });
});

// ─── Questions ───────────────────────────────────────────────────────────────

describe('POST /api/questions', () => {
  it('creates a question and returns 201', async () => {
    const created = {
      _id: 'q1',
      sessionId: 'sid1',
      text: 'What is WebSockets?',
      authorName: 'Bob',
      upvotes: 0,
      isAnswered: false,
      toObject: function () { return this; },
    };
    Question.create.mockResolvedValue(created);

    const res = await request(app)
      .post('/api/questions')
      .send({ sessionId: 'sid1', text: 'What is WebSockets?', authorName: 'Bob' });
    expect(res.status).toBe(201);
    expect(res.body.upvotes).toBe(0);
    expect(res.body.isAnswered).toBe(false);
  });
});

describe('POST /api/questions/:id/upvote', () => {
  it('increments upvote count', async () => {
    const updated = { _id: 'q1', upvotes: 1, toObject: function () { return this; } };
    Question.findByIdAndUpdate.mockResolvedValue(updated);

    const res = await request(app).post('/api/questions/q1/upvote');
    expect(res.status).toBe(200);
    expect(res.body.upvotes).toBe(1);
  });
});

describe('PATCH /api/questions/:id/answer', () => {
  it('marks a question as answered', async () => {
    const updated = { _id: 'q1', isAnswered: true, toObject: function () { return this; } };
    Question.findByIdAndUpdate.mockResolvedValue(updated);

    const res = await request(app).patch('/api/questions/q1/answer');
    expect(res.status).toBe(200);
    expect(res.body.isAnswered).toBe(true);
  });
});
