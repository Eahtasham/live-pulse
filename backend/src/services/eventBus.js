/**
 * EventBus: Redis-backed publish/subscribe event bus for horizontal scalability.
 * Falls back to an in-process EventEmitter when Redis is unavailable (e.g. tests).
 */
const { EventEmitter } = require('events');

const CHANNELS = {
  POLL_UPDATED: 'poll:updated',
  POLL_CREATED: 'poll:created',
  VOTE_CAST: 'poll:vote',
  QUESTION_CREATED: 'question:created',
  QUESTION_UPVOTED: 'question:upvoted',
  QUESTION_ANSWERED: 'question:answered',
  QUESTION_PINNED: 'question:pinned',
};

class EventBus extends EventEmitter {
  constructor() {
    super();
    this._publisher = null;
    this._subscriber = null;
    this._redisEnabled = false;
  }

  /** Attempt to connect to Redis. If it fails, the bus stays in local mode. */
  async connect(redisUrl) {
    try {
      const Redis = require('ioredis');
      const opts = { lazyConnect: true, enableOfflineQueue: false, maxRetriesPerRequest: 1 };

      this._publisher = new Redis(redisUrl, opts);
      this._subscriber = new Redis(redisUrl, opts);

      await Promise.all([this._publisher.connect(), this._subscriber.connect()]);

      this._subscriber.on('message', (channel, message) => {
        try {
          const payload = JSON.parse(message);
          super.emit(channel, payload);
        } catch {
          // ignore malformed messages
        }
      });

      this._redisEnabled = true;
      console.log('EventBus: Redis connected');
    } catch (err) {
      console.warn('EventBus: Redis unavailable, using in-process events –', err.message);
      this._redisEnabled = false;
    }
  }

  async publish(channel, payload) {
    if (this._redisEnabled) {
      await this._publisher.publish(channel, JSON.stringify(payload));
    } else {
      // local fallback: emit synchronously
      super.emit(channel, payload);
    }
  }

  async subscribe(channel) {
    if (this._redisEnabled) {
      await this._subscriber.subscribe(channel);
    }
    // In local mode the EventEmitter handles delivery; nothing extra needed.
  }

  async close() {
    if (this._publisher) await this._publisher.quit().catch(() => {});
    if (this._subscriber) await this._subscriber.quit().catch(() => {});
  }
}

const eventBus = new EventBus();

module.exports = { eventBus, CHANNELS };
