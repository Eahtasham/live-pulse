const mongoose = require('mongoose');

const sessionSchema = new mongoose.Schema(
  {
    title: { type: String, required: true, trim: true },
    description: { type: String, trim: true, default: '' },
    code: { type: String, required: true, unique: true, uppercase: true },
    hostId: { type: String, required: true },
    isActive: { type: Boolean, default: true },
  },
  { timestamps: true }
);

module.exports = mongoose.model('Session', sessionSchema);
