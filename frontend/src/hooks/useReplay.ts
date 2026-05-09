import { useCallback, useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getRoomHistory, getGameHistory } from '../api/client';
import type { EventHistoryResponse } from '../types';

interface ReplayState {
  key: string;
  step: number | null;
  isPlaying: boolean;
  speed: number;
}

export function useReplay(roomId: number, gameId?: number, startAtEnd = false) {
  const replayKey = `${roomId}:${gameId ?? 'latest'}`;
  const [replayState, setReplayState] = useState<ReplayState>({
    key: replayKey,
    step: null,
    isPlaying: false,
    speed: 1,
  });
  const playTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { data: history, isLoading, error } = useQuery<EventHistoryResponse>({
    queryKey: ['roomHistory', roomId, gameId],
    queryFn: () => gameId ? getGameHistory(gameId) : getRoomHistory(roomId),
  });

  const total = history?.events?.length ?? 0;
  const currentReplayState = replayState.key === replayKey
    ? replayState
    : { key: replayKey, step: null, isPlaying: false, speed: replayState.speed };
  const defaultStep = startAtEnd ? Math.max(0, total - 1) : 0;
  const step = currentReplayState.step ?? defaultStep;
  const isPlaying = currentReplayState.isPlaying;
  const speed = currentReplayState.speed;

  const updateReplayState = useCallback((updater: (state: ReplayState) => ReplayState) => {
    setReplayState(prev => {
      const current = prev.key === replayKey
        ? prev
        : { key: replayKey, step: null, isPlaying: false, speed: prev.speed };
      return updater(current);
    });
  }, [replayKey]);

  const goNext = useCallback(() => {
    updateReplayState(current => ({
      ...current,
      step: Math.min((current.step ?? defaultStep) + 1, total - 1),
    }));
  }, [defaultStep, total, updateReplayState]);

  const goPrev = useCallback(() => {
    updateReplayState(current => ({
      ...current,
      step: Math.max((current.step ?? defaultStep) - 1, 0),
      isPlaying: false,
    }));
  }, [defaultStep, updateReplayState]);

  const goTo = useCallback((s: number) => {
    updateReplayState(current => ({
      ...current,
      step: Math.max(0, Math.min(s, total - 1)),
      isPlaying: false,
    }));
  }, [total, updateReplayState]);

  const togglePlay = useCallback(() => {
    updateReplayState(current => {
      const currentStep = current.step ?? defaultStep;
      if (currentStep >= total - 1) {
        return { ...current, isPlaying: false };
      }
      return { ...current, isPlaying: !current.isPlaying };
    });
  }, [defaultStep, total, updateReplayState]);

  const setSpeed = useCallback((nextSpeed: number) => {
    updateReplayState(current => ({
      ...current,
      speed: nextSpeed,
    }));
  }, [updateReplayState]);

  useEffect(() => {
    if (!isPlaying || !history) return;

    const events = history.events;
    if (step >= total - 1) {
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
      updateReplayState(current => {
        const currentStep = current.step ?? defaultStep;
        const nextStep = Math.min(currentStep + 1, total - 1);
        return {
          ...current,
          step: nextStep,
          isPlaying: nextStep < total - 1,
        };
      });
    }, delayMs);

    return () => {
      if (playTimer.current) clearTimeout(playTimer.current);
    };
  }, [defaultStep, history, isPlaying, speed, step, total, updateReplayState]);

  return { history, step, total, isPlaying, speed, setSpeed, isLoading, error, goNext, goPrev, goTo, togglePlay };
}
