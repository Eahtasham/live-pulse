export interface QAEntry {
  id: string;
  sessionId: string;
  authorUid: string;
  entryType: "question" | "comment";
  body: string;
  score: number;
  status: "visible" | "answered" | "pinned" | "archived";
  isHidden: boolean;
  createdAt: string;
}
