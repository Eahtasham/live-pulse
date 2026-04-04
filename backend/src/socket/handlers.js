const { eventBus, CHANNELS } = require('../services/eventBus');

/**
 * Register Socket.io event handlers.
 * Each client joins a "session room" identified by sessionId so that
 * real-time events are scoped to the relevant audience.
 */
function registerSocketHandlers(io) {
  // Bridge every EventBus channel into the correct Socket.io room
  const sessionChannels = [
    CHANNELS.POLL_CREATED,
    CHANNELS.POLL_UPDATED,
    CHANNELS.VOTE_CAST,
    CHANNELS.QUESTION_CREATED,
    CHANNELS.QUESTION_UPVOTED,
    CHANNELS.QUESTION_ANSWERED,
    CHANNELS.QUESTION_PINNED,
  ];

  sessionChannels.forEach(channel => {
    eventBus.on(channel, payload => {
      const sessionId = String(payload.sessionId);
      io.to(`session:${sessionId}`).emit(channel, payload);
    });
  });

  io.on('connection', socket => {
    console.log(`Socket connected: ${socket.id}`);

    // Client sends { sessionId } to join the real-time room for that session
    socket.on('join-session', ({ sessionId } = {}) => {
      if (!sessionId) return;
      socket.join(`session:${sessionId}`);
      socket.emit('joined', { sessionId });
      console.log(`Socket ${socket.id} joined session:${sessionId}`);
    });

    socket.on('leave-session', ({ sessionId } = {}) => {
      if (!sessionId) return;
      socket.leave(`session:${sessionId}`);
    });

    socket.on('disconnect', () => {
      console.log(`Socket disconnected: ${socket.id}`);
    });
  });
}

module.exports = { registerSocketHandlers };
