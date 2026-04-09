import { SessionJoinView } from "@/components/session/session-join-view";

export default async function SessionPage({
  params,
}: {
  params: Promise<{ code: string }>;
}) {
  const { code } = await params;

  return <SessionJoinView code={code} />;
}
