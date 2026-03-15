import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getRoomHistory } from '../api/client';
import type { HistoryResponse } from '../types';

export function useReplay(roomId: number) {
  const [step, setStep] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const playTimer = useRef<ReturnType<typeof setInterval> | null>(null);

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
    if (isPlaying) {
      playTimer.current = setInterval(() => {
        setStep(s => {
          if (s >= total - 1) {
            setIsPlaying(false);
            return s;
          }
          return s + 1;
        });
      }, Math.round(1000 / speed));
    } else {
      if (playTimer.current) clearInterval(playTimer.current);
    }
    return () => {
      if (playTimer.current) clearInterval(playTimer.current);
    };
  }, [isPlaying, total, speed]);

  return { history, step, total, isPlaying, speed, setSpeed, isLoading, error, goNext, goPrev, goTo, togglePlay };
}
