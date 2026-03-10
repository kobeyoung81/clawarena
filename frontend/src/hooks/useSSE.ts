import { useEffect, useRef, useState } from 'react';

interface SSEState {
  latestEvent: Record<string, unknown> | null;
  isConnected: boolean;
  error: string | null;
}

export function useSSE(roomId: number | null) {
  const [state, setState] = useState<SSEState>({
    latestEvent: null,
    isConnected: false,
    error: null,
  });
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (roomId == null) return;

    const baseURL = import.meta.env.VITE_API_BASE_URL || '';
    const url = `${baseURL}/api/v1/rooms/${roomId}/watch`;

    function connect() {
      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => setState(s => ({ ...s, isConnected: true, error: null }));

      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data) as Record<string, unknown>;
          setState(s => ({ ...s, latestEvent: data, isConnected: true }));
        } catch {
          // ignore parse errors
        }
      };

      es.onerror = () => {
        es.close();
        esRef.current = null;
        setState(s => ({ ...s, isConnected: false, error: 'Connection lost' }));
        reconnectTimer.current = setTimeout(connect, 3000);
      };
    }

    connect();

    return () => {
      esRef.current?.close();
      esRef.current = null;
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
    };
  }, [roomId]);

  return state;
}
