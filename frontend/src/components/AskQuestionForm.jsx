import React, { useState } from 'react';
import { createQuestion } from '../api';

export default function AskQuestionForm({ sessionId, onCreated }) {
  const [text, setText] = useState('');
  const [author, setAuthor] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    if (!text.trim()) { setError('Please enter your question'); return; }

    setLoading(true);
    try {
      await createQuestion({ sessionId, text: text.trim(), authorName: author.trim() || 'Anonymous' });
      setText('');
      onCreated();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="card">
      <h3>Ask a Question</h3>
      <div className="form-group">
        <label>Your name (optional)</label>
        <input
          type="text"
          placeholder="Anonymous"
          value={author}
          onChange={e => setAuthor(e.target.value)}
        />
      </div>
      <div className="form-group">
        <label>Question</label>
        <textarea
          placeholder="What would you like to ask?"
          value={text}
          onChange={e => setText(e.target.value)}
        />
      </div>
      {error && <p className="error-msg">{error}</p>}
      <button type="submit" className="btn btn-primary" disabled={loading}>
        {loading ? <span className="spinner" /> : 'Submit Question'}
      </button>
    </form>
  );
}
