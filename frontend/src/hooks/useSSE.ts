import { useEffect, useRef, useState, useCallback } from 'react';
import type { GameEvent } from '../types';

interface SSEState {
  events: GameEvent[];
  latestEvent: GameEvent | null;
  isConnected: boolean;
  error: string | null;
}

export function useSSE(roomId: number | null) {
  const [state, setState] = useState<SSEState>({
    events: [],
    latestEvent: null,
    isConnected: false,
    error: null,
  });
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const seenSeqs = useRef<Set<number>>(new Set());

  const appendEvent = useCallback((evt: GameEvent) => {
    if (seenSeqs.current.has(evt.seq)) return;
    seenSeqs.current.add(evt.seq);
    setState(s => ({
      ...s,
      events: [...s.events, evt],
      latestEvent: evt,
      isConnected: true,
    }));
  }, []);

  useEffect(() => {
    if (roomId == null) return;

    // Reset on roomId change
    setState({ events: [], latestEvent: null, isConnected: false, error: null });
    seenSeqs.current.clear();

    const baseURL = '';
    const url = `${baseURL}/api/v1/rooms/${roomId}/watch`;

    function connect() {
      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => setState(s => ({ ...s, isConnected: true, error: null }));

      // Named event listeners for the new event-sourced backend
      es.addEventListener('game_event', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data) as GameEvent;
          appendEvent(data);
        } catch {
          // ignore parse errors
        }
      });

      es.addEventListener('room_event', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data) as GameEvent;
          appendEvent(data);
        } catch {
          // ignore parse errors
        }
      });

      // Backward compat: handle unnamed messages (old backend format)
      es.onmessage = (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          // If it looks like a GameEvent (has seq), treat it as one
          if (typeof data.seq === 'number') {
            appendEvent(data as GameEvent);
          } else {
            // Legacy format: wrap as a pseudo-event
            const pseudo: GameEvent = {
              seq: Date.now(),
              source: 'system',
              event_type: 'legacy',
              state: (data.state as Record<string, unknown>) ?? data,
              visibility: 'public',
              pending_action: data.pending_action ?? null,
              agents: data.agents ?? undefined,
              current_agent_id: data.current_agent_id ?? undefined,
            };
            appendEvent(pseudo);
          }
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
  }, [roomId, appendEvent]);

  return state;
}
