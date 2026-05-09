import { useEffect, useRef, useState, useCallback } from 'react';
import type { GameEvent } from '../types';

interface SSEState {
  events: GameEvent[];
  latestEvent: GameEvent | null;
  isConnected: boolean;
  error: string | null;
}

interface RoomSSEState extends SSEState {
  roomId: number | null;
}

const EMPTY_STATE: SSEState = {
  events: [],
  latestEvent: null,
  isConnected: false,
  error: null,
};

function createRoomState(roomId: number | null): RoomSSEState {
  return { roomId, ...EMPTY_STATE };
}

export function useSSE(roomId: number | null) {
  const [state, setState] = useState<RoomSSEState>(() => createRoomState(roomId));
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const seenSeqs = useRef<Set<number>>(new Set());

  const currentState = state.roomId === roomId ? state : createRoomState(roomId);

  const appendEvent = useCallback((evt: GameEvent, currentRoomId: number) => {
    if (seenSeqs.current.has(evt.seq)) return;
    seenSeqs.current.add(evt.seq);
    setState(prev => {
      const base = prev.roomId === currentRoomId ? prev : createRoomState(currentRoomId);
      return {
        ...base,
        events: [...base.events, evt],
        latestEvent: evt,
        isConnected: true,
      };
    });
  }, []);

  useEffect(() => {
    if (roomId == null) return;

    seenSeqs.current.clear();

    const baseURL = '';
    const url = `${baseURL}/api/v1/rooms/${roomId}/watch`;
    const currentRoomId = roomId;

    function connect() {
      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => setState(prev => {
        const base = prev.roomId === currentRoomId ? prev : createRoomState(currentRoomId);
        return { ...base, isConnected: true, error: null };
      });

      // Named event listeners for the new event-sourced backend
      es.addEventListener('game_event', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data) as GameEvent;
          appendEvent(data, currentRoomId);
        } catch {
          // ignore parse errors
        }
      });

      es.addEventListener('room_event', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data) as GameEvent;
          appendEvent(data, currentRoomId);
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
            appendEvent(data as GameEvent, currentRoomId);
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
            appendEvent(pseudo, currentRoomId);
          }
        } catch {
          // ignore parse errors
        }
      };

      es.onerror = () => {
        es.close();
        esRef.current = null;
        setState(prev => {
          const base = prev.roomId === currentRoomId ? prev : createRoomState(currentRoomId);
          return { ...base, isConnected: false, error: 'Connection lost' };
        });
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

  return currentState;
}
