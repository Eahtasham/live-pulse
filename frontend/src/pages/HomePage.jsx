import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { createSession, getSessionByCode } from '../api';

export default function HomePage() {
  const navigate = useNavigate();
  const [tab, setTab] = useState('host'); // 'host' | 'join'

  // Host form
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [hostError, setHostError] = useState('');
  const [hostLoading, setHostLoading] = useState(false);

  // Join form
  const [code, setCode] = useState('');
  const [joinError, setJoinError] = useState('');
  const [joinLoading, setJoinLoading] = useState(false);

  async function handleHost(e) {
    e.preventDefault();
    setHostError('');
    if (!title.trim()) { setHostError('Session title is required'); return; }
    setHostLoading(true);
    try {
      const hostId = `host-${Date.now()}`;
      const session = await createSession({ title: title.trim(), description: description.trim(), hostId });
      // Store hostId so the session page knows we're the host
      localStorage.setItem(`host:${session._id}`, hostId);
      navigate(`/session/${session._id}?host=true`);
    } catch (err) {
      setHostError(err.message);
    } finally {
      setHostLoading(false);
    }
  }

  async function handleJoin(e) {
    e.preventDefault();
    setJoinError('');
    if (!code.trim()) { setJoinError('Enter a session code'); return; }
    setJoinLoading(true);
    try {
      const session = await getSessionByCode(code.trim());
      navigate(`/session/${session._id}`);
    } catch (err) {
      setJoinError('Session not found. Check your code.');
    } finally {
      setJoinLoading(false);
    }
  }

  return (
    <>
      <div className="hero">
        <h1>Welcome to <em>LivePulse</em></h1>
        <p>
          Run real-time polls and live Q&amp;A sessions with your audience — powered by an event-driven cloud architecture.
        </p>
        <div className="hero-actions">
          <button className={`btn ${tab === 'host' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setTab('host')}>
            🎤 Host a Session
          </button>
          <button className={`btn ${tab === 'join' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setTab('join')}>
            🔗 Join a Session
          </button>
        </div>
      </div>

      <div className="container" style={{ maxWidth: 480 }}>
        {tab === 'host' && (
          <div className="card">
            <h3>Create a New Session</h3>
            <form onSubmit={handleHost}>
              <div className="form-group">
                <label>Session Title</label>
                <input
                  type="text"
                  placeholder="e.g. Engineering All-Hands Q3"
                  value={title}
                  onChange={e => setTitle(e.target.value)}
                />
              </div>
              <div className="form-group">
                <label>Description (optional)</label>
                <input
                  type="text"
                  placeholder="A brief description…"
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                />
              </div>
              {hostError && <p className="error-msg">{hostError}</p>}
              <button type="submit" className="btn btn-primary" disabled={hostLoading} style={{ width: '100%' }}>
                {hostLoading ? <span className="spinner" /> : 'Create Session →'}
              </button>
            </form>
          </div>
        )}

        {tab === 'join' && (
          <div className="card">
            <h3>Join a Session</h3>
            <form onSubmit={handleJoin}>
              <div className="form-group">
                <label>Session Code</label>
                <input
                  type="text"
                  placeholder="e.g. ABC123"
                  value={code}
                  onChange={e => setCode(e.target.value.toUpperCase())}
                  style={{ letterSpacing: '0.15em', fontWeight: 700, fontSize: '1.1rem' }}
                />
              </div>
              {joinError && <p className="error-msg">{joinError}</p>}
              <button type="submit" className="btn btn-primary" disabled={joinLoading} style={{ width: '100%' }}>
                {joinLoading ? <span className="spinner" /> : 'Join →'}
              </button>
            </form>
          </div>
        )}

        {/* Feature overview */}
        <div className="card mt-2" style={{ marginTop: '2rem' }}>
          <h3 style={{ marginBottom: '1rem' }}>Platform Features</h3>
          <ul style={{ listStyle: 'none', display: 'grid', gap: '0.6rem' }}>
            {[
              ['⚡', 'Real-time updates via WebSocket (Socket.io)'],
              ['📊', 'Live polling with animated bar charts'],
              ['🙋', 'Q&A with upvoting & host moderation'],
              ['🔔', 'Event-driven backend with Redis pub/sub'],
              ['📦', 'Horizontally scalable cloud architecture'],
            ].map(([icon, text]) => (
              <li key={text} style={{ display: 'flex', gap: '0.75rem', color: 'var(--muted)' }}>
                <span style={{ fontSize: '1.1rem' }}>{icon}</span>
                <span style={{ fontSize: '0.9rem' }}>{text}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </>
  );
}
