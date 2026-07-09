"use client";

import { useCallback, useEffect, useRef, useState } from "react";

interface PollState<T> {
  data: T[];
  error: string | null;
  loading: boolean;
  /** True after the first response (success or error) has landed. */
  ready: boolean;
}

/**
 * Poll a BFF endpoint that returns a JSON array on an interval. Never throws to
 * the browser — the BFF returns an error envelope, which we surface as `error`.
 * Keeps last good data on transient errors so the UI doesn't flicker empty.
 */
export function usePolling<T>(
  url: string,
  intervalMs: number
): PollState<T> & { refresh: () => void } {
  const [data, setData] = useState<T[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [ready, setReady] = useState(false);
  const mounted = useRef(true);

  const load = useCallback(async () => {
    try {
      const res = await fetch(url, { cache: "no-store" });
      const body = await res.json().catch(() => null);
      if (!mounted.current) return;
      if (!res.ok) {
        const msg =
          (body && (body.error as string)) || `request failed (${res.status})`;
        setError(msg);
      } else {
        setData(Array.isArray(body) ? (body as T[]) : []);
        setError(null);
      }
    } catch (e) {
      if (!mounted.current) return;
      setError(e instanceof Error ? e.message : "network error");
    } finally {
      if (mounted.current) {
        setLoading(false);
        setReady(true);
      }
    }
  }, [url]);

  useEffect(() => {
    mounted.current = true;
    void load();
    const id = setInterval(() => void load(), intervalMs);
    return () => {
      mounted.current = false;
      clearInterval(id);
    };
  }, [load, intervalMs]);

  return { data, error, loading, ready, refresh: () => void load() };
}
