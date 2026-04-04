/**
 * Channel names mirroring the backend CHANNELS constants.
 * Used in the frontend to subscribe to the correct Socket.io events.
 */
export const CHANNELS = {
  POLL_CREATED: 'poll:created',
  POLL_UPDATED: 'poll:updated',
  VOTE_CAST: 'poll:vote',
  QUESTION_CREATED: 'question:created',
  QUESTION_UPVOTED: 'question:upvoted',
  QUESTION_ANSWERED: 'question:answered',
  QUESTION_PINNED: 'question:pinned',
};
