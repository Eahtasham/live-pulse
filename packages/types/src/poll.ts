export interface Poll {
  id: string;
  sessionId: string;
  question: string;
  answerMode: "single" | "multi";
  timeLimitSec: number | null;
  status: "draft" | "active" | "closed";
  createdAt: string;
}

export interface PollOption {
  id: string;
  pollId: string;
  label: string;
  position: number;
}

export interface PollOptionWithCount extends PollOption {
  count: number;
}
