import React from 'react';
import { upvoteQuestion, answerQuestion, pinQuestion } from '../api';

export default function QuestionList({ questions, isHost, onUpdate }) {
  async function handleUpvote(qId) {
    await upvoteQuestion(qId);
    onUpdate();
  }

  async function handleAnswer(qId) {
    await answerQuestion(qId);
    onUpdate();
  }

  async function handlePin(qId) {
    await pinQuestion(qId);
    onUpdate();
  }

  if (questions.length === 0) {
    return <p className="text-muted text-center" style={{ padding: '2rem 0' }}>No questions yet. Be the first to ask!</p>;
  }

  return (
    <div>
      {questions.map(q => (
        <div
          key={q._id}
          className={`question-item${q.isAnswered ? ' answered' : ''}${q.isPinned ? ' pinned' : ''}`}
        >
          <button className="upvote-btn" onClick={() => handleUpvote(q._id)} disabled={q.isAnswered}>
            <span className="arrow">▲</span>
            <span className="count">{q.upvotes}</span>
          </button>
          <div style={{ flex: 1 }}>
            <p className="question-text">
              {q.isPinned && <span style={{ marginRight: 6 }}>📌</span>}
              {q.text}
            </p>
            <p className="question-meta">
              {q.authorName} &middot;{' '}
              {q.isAnswered ? (
                <span style={{ color: 'var(--success)' }}>✓ Answered</span>
              ) : (
                'Unanswered'
              )}
            </p>
            {isHost && !q.isAnswered && (
              <div className="question-actions">
                <button className="btn btn-sm btn-success" onClick={() => handleAnswer(q._id)}>
                  Mark Answered
                </button>
                <button className="btn btn-sm btn-secondary" onClick={() => handlePin(q._id)}>
                  {q.isPinned ? 'Unpin' : 'Pin'}
                </button>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
