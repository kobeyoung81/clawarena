import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getRoomHistory, getGameHistory } from '../api/client';
import type { EventHistoryResponse } from '../types';

export function useReplay(roomId: number, gameId?: number, startAtEnd = false) {
  const [step, setStep] = useState(0);
  const initializedStepRef = useRef(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const playTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { data: history, isLoading, error } = useQuery<EventHistoryResponse>({
    queryKey: ['roomHistory', roomId, gameId],
    queryFn: () => gameId ? getGameHistory(gameId) : getRoomHistory(roomId),
  });

  const total = history?.events?.length ?? 0;

  useEffect(() => {
    initializedStepRef.current = false;
    setStep(0);
  }, [roomId, gameId]);

  useEffect(() => {
    if (!history || initializedStepRef.current) return;
    initializedStepRef.current = true;
    if (startAtEnd) {
      setStep(Math.max(0, (history.events?.length ?? 1) - 1));
    } else {
      setStep(0);
    }
  }, [history, startAtEnd]);

  const goNext = useCallback(() => {
    setStep(s => Math.min(s + 1, total - 1));
  }, [total]);

  const goPrev = useCallback(() => {
    setStep(s => Math.max(s - 1, 0));
  }, []);

  const goTo = useCallback((s: number) => {
    setStep(Math.max(0, Math.min(s, total - 1)));
  }, [total]);

  const togglePlay = useCallback(() => {
    setIsPlaying(p => !p);
  }, []);

  useEffect(() => {
    if (!isPlaying || !history) return;

    const events = history.events;
    if (step >= total - 1) {
      setIsPlaying(false);
      return;
    }

    const currentEntry = events[step];
    const nextEntry = events[step + 1];

    let delayMs: number;
    if (currentEntry?.created_at && nextEntry?.created_at) {
      const realDelta = new Date(nextEntry.created_at).getTime() - new Date(currentEntry.created_at).getTime();
      const clampedDelta = Math.max(100, Math.min(realDelta, 10000));
      delayMs = Math.round(clampedDelta / speed);
    } else {
      delayMs = Math.round(1000 / speed);
    }
    // Ensure minimum 50ms even after speed division
    delayMs = Math.max(50, delayMs);

    playTimer.current = setTimeout(() => {
      setStep(s => s + 1);
    }, delayMs);

    return () => {
      if (playTimer.current) clearTimeout(playTimer.current);
    };
  }, [isPlaying, step, speed, total, history]);

  return { history, step, total, isPlaying, speed, setSpeed, isLoading, error, goNext, goPrev, goTo, togglePlay };
}
