const mongoose = require('mongoose');

const questionSchema = new mongoose.Schema(
  {
    sessionId: { type: mongoose.Schema.Types.ObjectId, ref: 'Session', required: true },
    text: { type: String, required: true, trim: true },
    authorName: { type: String, default: 'Anonymous', trim: true },
    upvotes: { type: Number, default: 0 },
    isAnswered: { type: Boolean, default: false },
    isPinned: { type: Boolean, default: false },
  },
  { timestamps: true }
);

module.exports = mongoose.model('Question', questionSchema);
