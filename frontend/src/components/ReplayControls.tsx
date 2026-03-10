interface ReplayControlsProps {
  step: number;
  total: number;
  isPlaying: boolean;
  onPrev: () => void;
  onNext: () => void;
  onPlay: () => void;
  onJump: (step: number) => void;
}

export function ReplayControls({ step, total, isPlaying, onPrev, onNext, onPlay, onJump }: ReplayControlsProps) {
  return (
    <div className="bg-gray-800 rounded-lg p-4 border border-gray-700 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">Replay</h3>
        <span className="text-sm text-gray-300">
          Step <span className="text-white font-bold">{step + 1}</span> / <span className="text-white font-bold">{total}</span>
        </span>
      </div>

      <div className="flex items-center gap-2">
        <button
          onClick={onPrev}
          disabled={step <= 0}
          className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 disabled:opacity-40 disabled:cursor-not-allowed rounded text-white text-sm transition-colors"
          title="Previous step"
        >
          ◀
        </button>
        <button
          onClick={onPlay}
          className={`px-3 py-1.5 rounded text-white text-sm transition-colors ${
            isPlaying ? 'bg-yellow-600 hover:bg-yellow-500' : 'bg-green-600 hover:bg-green-500'
          }`}
          title={isPlaying ? 'Pause' : 'Auto-play'}
        >
          {isPlaying ? '⏸' : '⏯'}
        </button>
        <button
          onClick={onNext}
          disabled={step >= total - 1}
          className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 disabled:opacity-40 disabled:cursor-not-allowed rounded text-white text-sm transition-colors"
          title="Next step"
        >
          ▶
        </button>
      </div>

      {total > 1 && (
        <input
          type="range"
          min={0}
          max={total - 1}
          value={step}
          onChange={e => onJump(Number(e.target.value))}
          className="w-full accent-blue-500"
        />
      )}
    </div>
  );
}
