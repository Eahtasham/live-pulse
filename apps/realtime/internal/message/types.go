package message

// Event types for WebSocket communication.
const (
	EventVoteUpdate    = "vote_update"
	EventNewQuestion   = "new_question"
	EventNewComment    = "new_comment"
	EventQAUpdate      = "qa_update"
	EventSessionClosed = "session_closed"
	EventPing          = "ping"
)
