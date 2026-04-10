import { SessionJoinView } from "@/components/session/session-join-view";

interface PageProps {
  params: Promise<{ code: string }>;
}

export default async function SessionPage({ params }: PageProps) {
  const { code } = await params;

  return <SessionJoinView code={code} />;
}
