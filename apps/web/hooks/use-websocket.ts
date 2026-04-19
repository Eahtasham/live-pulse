"use client";

import { useEffect, useRef, useState, useCallback } from "react";

export type ConnectionState = "connecting" | "connected" | "disconnected";

export interface WSMessage {
  type: string;
  payload: unknown;
}

type MessageHandler = (msg: WSMessage) => void;

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8081";
const PING_INTERVAL = 30_000;
const MAX_BACKOFF = 30_000;

export function useWebSocket(code: string) {
  const [state, setState] = useState<ConnectionState>("connecting");
  const wsRef = useRef<WebSocket | null>(null);
  const handlersRef = useRef<Set<MessageHandler>>(new Set());
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const attemptRef = useRef(0);
  const mountedRef = useRef(true);
  const reconnectCallbacksRef = useRef<Set<() => void>>(new Set());

  const connect = useCallback(() => {
    if (!mountedRef.current) return;

    setState("connecting");
    const url = `${WS_URL}/ws/${encodeURIComponent(code)}`;
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      if (!mountedRef.current) { ws.close(); return; }
      setState("connected");
      attemptRef.current = 0;

      // Start ping interval
      pingRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "ping" }));
        }
      }, PING_INTERVAL);
    };

    ws.onmessage = (event) => {
      // The server may batch multiple JSON messages into one frame,
      // separated by newlines. Parse each line individually.
      const lines = (event.data as string).split("\n");
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;
        try {
          const msg: WSMessage = JSON.parse(trimmed);
          handlersRef.current.forEach((handler) => handler(msg));
        } catch {
          // ignore non-JSON lines
        }
      }
    };

    ws.onclose = () => {
      if (!mountedRef.current) return;
      setState("disconnected");
      cleanup();
      scheduleReconnect();
    };

    ws.onerror = () => {
      // onclose will fire after onerror
    };
  }, [code]);

  function cleanup() {
    if (pingRef.current) {
      clearInterval(pingRef.current);
      pingRef.current = null;
    }
  }

  function scheduleReconnect() {
    if (!mountedRef.current) return;
    const delay = Math.min(1000 * Math.pow(2, attemptRef.current), MAX_BACKOFF);
    attemptRef.current += 1;

    reconnectRef.current = setTimeout(() => {
      if (!mountedRef.current) return;
      connect();
      // Fire reconnect callbacks after connection attempt
      // (actual state sync happens via onopen → connected state change)
      reconnectCallbacksRef.current.forEach((cb) => cb());
    }, delay);
  }

  // Subscribe to messages
  const subscribe = useCallback((handler: MessageHandler) => {
    handlersRef.current.add(handler);
    return () => { handlersRef.current.delete(handler); };
  }, []);

  // Subscribe to reconnect events (for REST fallback)
  const onReconnect = useCallback((cb: () => void) => {
    reconnectCallbacksRef.current.add(cb);
    return () => { reconnectCallbacksRef.current.delete(cb); };
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    connect();

    return () => {
      mountedRef.current = false;
      cleanup();
      if (reconnectRef.current) {
        clearTimeout(reconnectRef.current);
        reconnectRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  return { state, subscribe, onReconnect };
}
