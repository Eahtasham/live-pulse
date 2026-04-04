require('dotenv').config();
const http = require('http');
const express = require('express');
const cors = require('cors');
const rateLimit = require('express-rate-limit');
const { Server } = require('socket.io');

const connectDB = require('./config/database');
const { eventBus } = require('./services/eventBus');
const { registerSocketHandlers } = require('./socket/handlers');

const sessionRoutes = require('./routes/sessions');
const pollRoutes = require('./routes/polls');
const questionRoutes = require('./routes/questions');

const app = express();
const server = http.createServer(app);

const corsOrigin = process.env.CORS_ORIGIN || 'http://localhost:5173';

// --- Middleware ---
app.use(cors({ origin: corsOrigin, credentials: true }));
app.use(express.json());

// Rate limiting – protect all API routes
const apiLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 200,                  // max 200 requests per window per IP
  standardHeaders: true,
  legacyHeaders: false,
  message: { error: 'Too many requests, please try again later.' },
});

const voteLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 30,             // max 30 votes per minute per IP
  standardHeaders: true,
  legacyHeaders: false,
  message: { error: 'Too many votes, please slow down.' },
});

const questionLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 10,             // max 10 question submissions per minute per IP
  standardHeaders: true,
  legacyHeaders: false,
  message: { error: 'Too many questions submitted, please slow down.' },
});

// --- REST API ---
app.use('/api/sessions', apiLimiter, sessionRoutes);
app.use('/api/polls', apiLimiter, pollRoutes);
app.use('/api/polls/:id/vote', voteLimiter);        // tighter limit for voting
app.use('/api/questions', apiLimiter, questionRoutes);
app.use('/api/questions', questionLimiter);          // tighter limit for Q&A submissions

app.get('/api/health', (_req, res) => res.json({ status: 'ok', timestamp: new Date().toISOString() }));

// --- Socket.io ---
const io = new Server(server, {
  cors: { origin: corsOrigin, methods: ['GET', 'POST'] },
});
registerSocketHandlers(io);

// --- Start ---
const PORT = process.env.PORT || 4000;

async function start() {
  await connectDB();
  await eventBus.connect(process.env.REDIS_URL || 'redis://localhost:6379');
  server.listen(PORT, () => console.log(`LivePulse backend listening on port ${PORT}`));
}

start().catch(err => {
  console.error('Startup error:', err);
  process.exit(1);
});

module.exports = { app, server }; // exported for tests
