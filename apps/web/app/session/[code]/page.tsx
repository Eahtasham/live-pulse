export default async function SessionPage({
  params,
}: {
  params: Promise<{ code: string }>;
}) {
  const { code } = await params;

  return (
    <div className="flex min-h-screen flex-col items-center justify-center">
      <h1 className="text-3xl font-bold">Session: {code}</h1>
      <p className="mt-2 text-gray-500">Audience view — polls &amp; Q&amp;A will appear here</p>
    </div>
  );
}
