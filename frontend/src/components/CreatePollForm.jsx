import React, { useState } from 'react';
import { createPoll } from '../api';

export default function CreatePollForm({ sessionId, onCreated }) {
  const [question, setQuestion] = useState('');
  const [options, setOptions] = useState(['', '']);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  function addOption() {
    setOptions([...options, '']);
  }

  function updateOption(idx, val) {
    const updated = [...options];
    updated[idx] = val;
    setOptions(updated);
  }

  function removeOption(idx) {
    if (options.length <= 2) return;
    setOptions(options.filter((_, i) => i !== idx));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    const filtered = options.map(o => o.trim()).filter(Boolean);
    if (!question.trim()) { setError('Question is required'); return; }
    if (filtered.length < 2) { setError('At least 2 non-empty options required'); return; }

    setLoading(true);
    try {
      await createPoll({ sessionId, question: question.trim(), options: filtered });
      setQuestion('');
      setOptions(['', '']);
      onCreated();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="card">
      <h3>Create a Poll</h3>
      <div className="form-group">
        <label>Question</label>
        <input
          type="text"
          placeholder="Ask a question…"
          value={question}
          onChange={e => setQuestion(e.target.value)}
        />
      </div>
      <div className="form-group">
        <label>Options</label>
        {options.map((opt, idx) => (
          <div key={idx} className="add-option-row">
            <input
              type="text"
              placeholder={`Option ${idx + 1}`}
              value={opt}
              onChange={e => updateOption(idx, e.target.value)}
            />
            {options.length > 2 && (
              <button type="button" className="btn btn-sm btn-secondary" onClick={() => removeOption(idx)}>
                ✕
              </button>
            )}
          </div>
        ))}
        <button type="button" className="btn btn-sm btn-secondary mt-1" onClick={addOption}>
          + Add option
        </button>
      </div>
      {error && <p className="error-msg">{error}</p>}
      <button type="submit" className="btn btn-primary" disabled={loading}>
        {loading ? <span className="spinner" /> : 'Launch Poll'}
      </button>
    </form>
  );
}
