const mongoose = require('mongoose');

const pollOptionSchema = new mongoose.Schema({
  text: { type: String, required: true, trim: true },
  votes: { type: Number, default: 0 },
});

const pollSchema = new mongoose.Schema(
  {
    sessionId: { type: mongoose.Schema.Types.ObjectId, ref: 'Session', required: true },
    question: { type: String, required: true, trim: true },
    options: { type: [pollOptionSchema], validate: v => v.length >= 2 },
    isActive: { type: Boolean, default: true },
    totalVotes: { type: Number, default: 0 },
  },
  { timestamps: true }
);

module.exports = mongoose.model('Poll', pollSchema);
