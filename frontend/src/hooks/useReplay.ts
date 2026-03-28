import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getRoomHistory } from '../api/client';
import type { HistoryResponse } from '../types';

export function useReplay(roomId: number) {
  const [step, setStep] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const playTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { data: history, isLoading, error } = useQuery<HistoryResponse>({
    queryKey: ['roomHistory', roomId],
    queryFn: () => getRoomHistory(roomId),
  });

  const total = history?.timeline.length ?? 0;

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

    const timeline = history.timeline;
    if (step >= total - 1) {
      setIsPlaying(false);
      return;
    }

    const currentEntry = timeline[step];
    const nextEntry = timeline[step + 1];

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
