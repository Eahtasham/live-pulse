export type WSEventType =
  | "vote_update"
  | "new_question"
  | "new_comment"
  | "qa_update"
  | "session_closed"
  | "ping";

export interface WSMessage<T = unknown> {
  type: WSEventType;
  payload: T;
}

export interface VoteUpdatePayload {
  pollId: string;
  options: { id: string; count: number }[];
}

export interface NewQuestionPayload {
  id: string;
  body: string;
  score: number;
  status: string;
}

export interface NewCommentPayload {
  id: string;
  body: string;
  authorUid: string;
}

export interface QAUpdatePayload {
  id: string;
  status: string;
  score: number;
}

export interface SessionClosedPayload {
  code: string;
  closedAt: string;
}
