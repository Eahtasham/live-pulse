export interface Session {
  id: string;
  hostId: string;
  code: string;
  title: string;
  status: "active" | "archived";
  createdAt: string;
  closedAt: string | null;
}
