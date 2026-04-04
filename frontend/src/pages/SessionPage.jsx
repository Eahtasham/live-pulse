import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import {
  getSessionById,
  getPollsBySession,
  getQuestionsBySession,
} from '../api';
import { io } from 'socket.io-client';
import PollCard from '../components/PollCard';
import QuestionList from '../components/QuestionList';
import CreatePollForm from '../components/CreatePollForm';
import AskQuestionForm from '../components/AskQuestionForm';
import { CHANNELS } from '../channels';

const SOCKET_URL = import.meta.env.VITE_API_URL || '';

export default function SessionPage() {
  const { id } = useParams();
  const [searchParams] = useSearchParams();
  const isHost = searchParams.get('host') === 'true';

  const [session, setSession] = useState(null);
  const [polls, setPolls] = useState([]);
  const [questions, setQuestions] = useState([]);
  const [activeTab, setActiveTab] = useState('qa');
  const [loading, setLoading] = useState(true);
  const socketRef = useRef(null);

  const fetchPolls = useCallback(async () => {
    if (!id) return;
    const data = await getPollsBySession(id);
    setPolls(data);
  }, [id]);

  const fetchQuestions = useCallback(async () => {
    if (!id) return;
    const data = await getQuestionsBySession(id);
    setQuestions(data);
  }, [id]);

  useEffect(() => {
    async function init() {
      const [sess] = await Promise.all([getSessionById(id), fetchPolls(), fetchQuestions()]);
      setSession(sess);
      setLoading(false);
    }
    init().catch(console.error);
  }, [id, fetchPolls, fetchQuestions]);

  // Socket.io – join the session room and react to real-time events
  useEffect(() => {
    const socket = io(SOCKET_URL, { transports: ['websocket', 'polling'] });
    socketRef.current = socket;

    socket.on('connect', () => {
      socket.emit('join-session', { sessionId: id });
    });

    // Any poll event → re-fetch polls
    [CHANNELS.POLL_CREATED, CHANNELS.POLL_UPDATED, CHANNELS.VOTE_CAST].forEach(ch => {
      socket.on(ch, () => fetchPolls());
    });

    // Any question event → re-fetch questions
    [
      CHANNELS.QUESTION_CREATED,
      CHANNELS.QUESTION_UPVOTED,
      CHANNELS.QUESTION_ANSWERED,
      CHANNELS.QUESTION_PINNED,
    ].forEach(ch => {
      socket.on(ch, () => fetchQuestions());
    });

    return () => {
      socket.emit('leave-session', { sessionId: id });
      socket.disconnect();
    };
  }, [id, fetchPolls, fetchQuestions]);

  if (loading) {
    return (
      <div className="page text-center">
        <span className="spinner" />
      </div>
    );
  }

  if (!session) {
    return <div className="page text-center text-muted">Session not found.</div>;
  }

  return (
    <div className="page">
      <div className="container">
        {/* Header */}
        <div className="card" style={{ marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <h2 style={{ fontSize: '1.5rem', marginBottom: '0.25rem' }}>{session.title}</h2>
              {session.description && <p className="text-muted">{session.description}</p>}
            </div>
            <span className={`badge ${session.isActive ? 'badge-green' : 'badge-gray'}`}>
              {session.isActive ? (
                <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span className="live-dot" /> Live
                </span>
              ) : 'Closed'}
            </span>
          </div>

          {isHost && (
            <div style={{ marginTop: '1rem', display: 'flex', alignItems: 'center', gap: '1rem', flexWrap: 'wrap' }}>
              <span className="text-muted" style={{ fontSize: '0.85rem' }}>Share code:</span>
              <span className="code-badge">{session.code}</span>
            </div>
          )}
        </div>

        {/* Tabs */}
        <div className="tabs">
          <button className={`tab-btn${activeTab === 'qa' ? ' active' : ''}`} onClick={() => setActiveTab('qa')}>
            🙋 Q&amp;A ({questions.length})
          </button>
          <button className={`tab-btn${activeTab === 'polls' ? ' active' : ''}`} onClick={() => setActiveTab('polls')}>
            📊 Polls ({polls.length})
          </button>
        </div>

        {/* Q&A Tab */}
        {activeTab === 'qa' && (
          <>
            {session.isActive && <AskQuestionForm sessionId={id} onCreated={fetchQuestions} />}
            <div className="section-header">
              <h2>Questions</h2>
              <span className="text-muted" style={{ fontSize: '0.8rem' }}>Sorted by votes</span>
            </div>
            <QuestionList questions={questions} isHost={isHost} onUpdate={fetchQuestions} />
          </>
        )}

        {/* Polls Tab */}
        {activeTab === 'polls' && (
          <>
            {isHost && session.isActive && (
              <CreatePollForm sessionId={id} onCreated={fetchPolls} />
            )}
            <div className="section-header">
              <h2>Polls</h2>
            </div>
            {polls.length === 0 ? (
              <p className="text-muted text-center" style={{ padding: '2rem 0' }}>
                {isHost ? 'Create the first poll above.' : 'No polls yet — check back soon!'}
              </p>
            ) : (
              polls.map(p => (
                <PollCard key={p._id} poll={p} isHost={isHost} onUpdate={fetchPolls} />
              ))
            )}
          </>
        )}
      </div>
    </div>
  );
}
