import React from 'react';
import { Link } from 'react-router-dom';

export default function Navbar() {
  return (
    <nav>
      <Link to="/" className="logo">
        Live<span>Pulse</span>
      </Link>
      <span className="text-muted" style={{ fontSize: '0.8rem' }}>
        Real-time Q&amp;A &amp; Polling
      </span>
    </nav>
  );
}
