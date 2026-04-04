const BASE = '/api';

async function request(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(`${BASE}${path}`, opts);
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Request failed');
  return data;
}

// Sessions
export const createSession = (payload) => request('POST', '/sessions', payload);
export const getSessionByCode = (code) => request('GET', `/sessions/code/${code}`);
export const getSessionById = (id) => request('GET', `/sessions/${id}`);
export const closeSession = (id) => request('PATCH', `/sessions/${id}/close`);

// Polls
export const createPoll = (payload) => request('POST', '/polls', payload);
export const getPollsBySession = (sessionId) => request('GET', `/polls/session/${sessionId}`);
export const votePoll = (pollId, optionIndex) =>
  request('POST', `/polls/${pollId}/vote`, { optionIndex });
export const closePoll = (pollId) => request('PATCH', `/polls/${pollId}/close`);

// Questions
export const createQuestion = (payload) => request('POST', '/questions', payload);
export const getQuestionsBySession = (sessionId) =>
  request('GET', `/questions/session/${sessionId}`);
export const upvoteQuestion = (questionId) => request('POST', `/questions/${questionId}/upvote`);
export const answerQuestion = (questionId) => request('PATCH', `/questions/${questionId}/answer`);
export const pinQuestion = (questionId) => request('PATCH', `/questions/${questionId}/pin`);
