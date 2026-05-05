import Link from "next/link";
import { ArrowLeft, SearchX } from "lucide-react";

import { Brand } from "@/components/brand";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function NotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-10">
      <div className="mx-auto flex w-full max-w-lg flex-col gap-6">
        <Brand href="/" />

        <Card className="border-border/70 bg-card/90 shadow-lg">
          <CardHeader className="space-y-4 text-center">
            <div className="mx-auto flex size-14 items-center justify-center rounded-2xl bg-primary/10 text-primary">
              <SearchX className="size-7" />
            </div>
            <CardTitle className="text-3xl tracking-tight">Page not found</CardTitle>
            <CardDescription className="text-base leading-7">
              The route you requested does not exist or has moved.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 sm:flex-row">
            <Button asChild className="h-11 flex-1 rounded-2xl">
              <Link href="/dashboard">
                <ArrowLeft className="size-4" />
                Go to dashboard
              </Link>
            </Button>
            <Button asChild variant="outline" className="h-11 flex-1 rounded-2xl">
              <Link href="/">Back to home</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </main>
  );
}