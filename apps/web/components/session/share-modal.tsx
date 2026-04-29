"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Share2Icon,
  CopyIcon,
  CheckIcon,
  SmartphoneIcon,
} from "lucide-react";

interface ShareModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sessionCode: string;
  sessionTitle: string;
  /** When true, shows the "Thanks for voting!" header */
  afterVote?: boolean;
}

function getSessionUrl(code: string) {
  if (typeof window === "undefined") return "";
  return `${window.location.origin}/session/${code}`;
}

function getShareText(title: string, code: string) {
  return `Join my LivePulse session "${title}" and share your opinion! \n\nSession code: ${code}`;
}

export function ShareModal({
  open,
  onOpenChange,
  sessionCode,
  sessionTitle,
  afterVote = false,
}: ShareModalProps) {
  const [copied, setCopied] = useState(false);
  const [canNativeShare, setCanNativeShare] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    setCanNativeShare(typeof navigator !== "undefined" && !!navigator.share);
    setIsMobile(
      typeof window !== "undefined" &&
      /Android|iPhone|iPad|iPod|Opera Mini|IEMobile|WPDesktop/i.test(
        navigator.userAgent
      )
    );
  }, []);

  const url = getSessionUrl(sessionCode);
  const shareText = getShareText(sessionTitle, sessionCode);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback
      const textarea = document.createElement("textarea");
      textarea.value = url;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [url]);

  const handleNativeShare = useCallback(async () => {
    try {
      await navigator.share({
        title: `LivePulse: ${sessionTitle}`,
        text: shareText,
        url,
      });
    } catch {
      // User cancelled or not supported
    }
  }, [sessionTitle, shareText, url]);

  const handleWhatsApp = useCallback(() => {
    const text = encodeURIComponent(`${shareText}\n\n${url}`);
    window.open(`https://wa.me/?text=${text}`, "_blank");
  }, [shareText, url]);

  const handleTwitter = useCallback(() => {
    const text = encodeURIComponent(
      `Join my LivePulse session "${sessionTitle}" 🗳️`
    );
    window.open(
      `https://twitter.com/intent/tweet?text=${text}&url=${encodeURIComponent(url)}`,
      "_blank"
    );
  }, [sessionTitle, url]);

  const handleFacebook = useCallback(() => {
    window.open(
      `https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(url)}`,
      "_blank"
    );
  }, [url]);

  const handleLinkedIn = useCallback(() => {
    window.open(
      `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(url)}`,
      "_blank"
    );
  }, [url]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {afterVote ? (
              <>
                <span className="text-xl">🎉</span>
                <span>Thanks for voting!</span>
              </>
            ) : (
              <>
                <Share2Icon className="size-5" />
                <span>Share this session</span>
              </>
            )}
          </DialogTitle>
          <DialogDescription>
            {afterVote
              ? "Invite your friends to vote too — the more the merrier!"
              : `Invite others to join "${sessionTitle}"`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 min-w-0">
          {/* Session link display */}
          <div className="flex items-center gap-2 rounded-lg border border-border bg-muted/50 px-3 py-2.5 min-w-0">
            <code className="flex-1 min-w-0 truncate text-sm font-mono text-foreground">
              {url}
            </code>
            <Button
              size="sm"
              variant="ghost"
              onClick={handleCopy}
              className="shrink-0"
              id="share-copy-link"
            >
              {copied ? (
                <CheckIcon className="size-4 text-green-500" />
              ) : (
                <CopyIcon className="size-4" />
              )}
              {copied ? "Copied!" : "Copy"}
            </Button>
          </div>

          {/* WhatsApp — dedicated prominent button */}
          <button
            onClick={handleWhatsApp}
            id="share-whatsapp"
            className="flex w-full items-center justify-center gap-3 rounded-xl px-4 py-3.5 text-sm font-semibold text-white transition-all hover:brightness-110 active:scale-[0.98]"
            style={{ backgroundColor: "#25D366" }}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="22"
              height="22"
              viewBox="0 0 24 24"
              fill="currentColor"
            >
              <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z" />
            </svg>
            Share on WhatsApp
          </button>

          {/* Native share button for mobile — prominent */}
          {canNativeShare && isMobile && (
            <button
              onClick={handleNativeShare}
              id="share-native"
              className="flex w-full items-center justify-center gap-3 rounded-xl border-2 border-primary bg-primary/10 px-4 py-3.5 text-sm font-semibold text-primary transition-all hover:bg-primary/20 active:scale-[0.98]"
            >
              <SmartphoneIcon className="size-5" />
              Share via your apps
            </button>
          )}

          {/* Other social media grid */}
          <div className="grid grid-cols-3 gap-2">
            <button
              onClick={handleTwitter}
              id="share-twitter"
              className="flex flex-col items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-3 text-xs font-medium text-foreground transition-all hover:bg-muted active:scale-[0.97]"
            >
              <div className="flex size-9 items-center justify-center rounded-full bg-black dark:bg-white">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  className="text-white dark:text-black"
                >
                  <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
                </svg>
              </div>
              X / Twitter
            </button>

            <button
              onClick={handleFacebook}
              id="share-facebook"
              className="flex flex-col items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-3 text-xs font-medium text-foreground transition-all hover:bg-muted active:scale-[0.97]"
            >
              <div
                className="flex size-9 items-center justify-center rounded-full"
                style={{ backgroundColor: "#1877F2" }}
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  className="text-white"
                >
                  <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
                </svg>
              </div>
              Facebook
            </button>

            <button
              onClick={handleLinkedIn}
              id="share-linkedin"
              className="flex flex-col items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-3 text-xs font-medium text-foreground transition-all hover:bg-muted active:scale-[0.97]"
            >
              <div
                className="flex size-9 items-center justify-center rounded-full"
                style={{ backgroundColor: "#0A66C2" }}
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  className="text-white"
                >
                  <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z" />
                </svg>
              </div>
              LinkedIn
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

/** Small share button for the session header bar */
export function ShareSessionButton({
  sessionCode,
  sessionTitle,
}: {
  sessionCode: string;
  sessionTitle: string;
}) {
  const [shareOpen, setShareOpen] = useState(false);

  const url = getSessionUrl(sessionCode);
  const shareText = getShareText(sessionTitle, sessionCode);

  const handleWhatsApp = useCallback(() => {
    const text = encodeURIComponent(`${shareText}\n\n${url}`);
    window.open(`https://wa.me/?text=${text}`, "_blank");
  }, [shareText, url]);

  return (
    <>
      <div className="flex items-center gap-1.5 shrink-0">
        {/* WhatsApp quick-share — icon-only to stay compact */}
        <button
          onClick={handleWhatsApp}
          id="header-share-whatsapp"
          title="Share on WhatsApp"
          className="inline-flex size-8 items-center justify-center rounded-lg text-white transition-all hover:brightness-110 active:scale-[0.96]"
          style={{ backgroundColor: "#25D366" }}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="currentColor"
          >
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413Z" />
          </svg>
        </button>

        {/* General share button */}
        <Button
          size="icon"
          variant="outline"
          onClick={() => setShareOpen(true)}
          title="Share session"
          id="header-share-button"
        >
          <Share2Icon className="size-4" />
        </Button>
      </div>

      <ShareModal
        open={shareOpen}
        onOpenChange={setShareOpen}
        sessionCode={sessionCode}
        sessionTitle={sessionTitle}
      />
    </>
  );
}
