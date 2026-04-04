/**
 * Unit tests for the EventBus service.
 * These tests use the in-process fallback mode (no Redis required).
 */
const { eventBus, CHANNELS } = require('../services/eventBus');

afterAll(async () => {
  await eventBus.close();
});

describe('EventBus (in-process mode)', () => {
  it('publishes and receives a VOTE_CAST event', done => {
    const payload = { sessionId: 'test-session', pollId: 'poll-1', optionIndex: 0 };

    eventBus.once(CHANNELS.VOTE_CAST, received => {
      expect(received).toEqual(payload);
      done();
    });

    eventBus.publish(CHANNELS.VOTE_CAST, payload);
  });

  it('publishes and receives a QUESTION_CREATED event', done => {
    const payload = { sessionId: 'test-session', text: 'How does this work?', authorName: 'Alice' };

    eventBus.once(CHANNELS.QUESTION_CREATED, received => {
      expect(received).toEqual(payload);
      done();
    });

    eventBus.publish(CHANNELS.QUESTION_CREATED, payload);
  });

  it('publishes QUESTION_UPVOTED with correct data', done => {
    const payload = { sessionId: 's1', _id: 'q1', upvotes: 5 };

    eventBus.once(CHANNELS.QUESTION_UPVOTED, received => {
      expect(received.upvotes).toBe(5);
      done();
    });

    eventBus.publish(CHANNELS.QUESTION_UPVOTED, payload);
  });

  it('exposes all expected channel names', () => {
    expect(CHANNELS.POLL_CREATED).toBe('poll:created');
    expect(CHANNELS.POLL_UPDATED).toBe('poll:updated');
    expect(CHANNELS.VOTE_CAST).toBe('poll:vote');
    expect(CHANNELS.QUESTION_CREATED).toBe('question:created');
    expect(CHANNELS.QUESTION_UPVOTED).toBe('question:upvoted');
    expect(CHANNELS.QUESTION_ANSWERED).toBe('question:answered');
    expect(CHANNELS.QUESTION_PINNED).toBe('question:pinned');
  });
});
