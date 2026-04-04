import React from 'react';
import { votePoll, closePoll } from '../api';

export default function PollCard({ poll, isHost, onUpdate }) {
  async function handleVote(idx) {
    if (!poll.isActive) return;
    await votePoll(poll._id, idx);
    onUpdate();
  }

  async function handleClose() {
    await closePoll(poll._id);
    onUpdate();
  }

  const total = poll.totalVotes || 1;

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.75rem' }}>
        <h3 style={{ margin: 0 }}>{poll.question}</h3>
        <span className={`badge ${poll.isActive ? 'badge-green' : 'badge-gray'}`}>
          {poll.isActive ? 'Live' : 'Closed'}
        </span>
      </div>

      {poll.options.map((opt, idx) => {
        const pct = poll.totalVotes > 0 ? Math.round((opt.votes / poll.totalVotes) * 100) : 0;
        return (
          <div key={idx} className="poll-option">
            <div className="poll-option-label">
              <button
                onClick={() => handleVote(idx)}
                disabled={!poll.isActive}
                style={{
                  background: 'none',
                  border: 'none',
                  color: poll.isActive ? 'var(--text)' : 'var(--muted)',
                  cursor: poll.isActive ? 'pointer' : 'default',
                  padding: 0,
                  fontWeight: 600,
                  fontSize: '0.9rem',
                  textAlign: 'left',
                }}
              >
                {opt.text}
              </button>
              <span style={{ color: 'var(--muted)', fontSize: '0.85rem' }}>
                {opt.votes} ({pct}%)
              </span>
            </div>
            <div className="poll-bar-track">
              <div className="poll-bar-fill" style={{ width: `${pct}%` }} />
            </div>
          </div>
        );
      })}

      <p className="text-muted mt-1" style={{ fontSize: '0.8rem' }}>
        {poll.totalVotes} vote{poll.totalVotes !== 1 ? 's' : ''}
      </p>

      {isHost && poll.isActive && (
        <div className="mt-1">
          <button className="btn btn-sm btn-danger" onClick={handleClose}>
            Close Poll
          </button>
        </div>
      )}
    </div>
  );
}
