import { useI18n } from '../../i18n';
import type { ClawedWolfPlayer } from '../../types';

export interface BoardProps {
  state: Record<string, unknown>;
  gameType?: string;
  players?: ClawedWolfPlayer[];
  isReplay?: boolean;
}

interface TicTacToeState {
  board: string[];
  winner: string | null;
  is_draw: boolean;
}

function getWinningLine(board: string[]): number[] | null {
  const lines = [
    [0, 1, 2], [3, 4, 5], [6, 7, 8],
    [0, 3, 6], [1, 4, 7], [2, 5, 8],
    [0, 4, 8], [2, 4, 6],
  ];
  for (const [a, b, c] of lines) {
    if (board[a] && board[a] === board[b] && board[a] === board[c]) return [a, b, c];
  }
  return null;
}

export function TicTacToeBoard({ state, players }: BoardProps) {
  const { t } = useI18n();
  const s = state as unknown as TicTacToeState;
  const board = s?.board ?? Array(9).fill('');
  const winner = s?.winner ?? null;
  const isDraw = s?.is_draw ?? false;
  const winLine = winner ? getWinningLine(board) : null;

  const allFilled = board.every(cell => cell !== '');
  const xCount = board.filter(c => c === 'X').length;
  const oCount = board.filter(c => c === 'O').length;

  const xName = players?.[0]?.name ?? 'X';
  const oName = players?.[1]?.name ?? 'O';

  let statusMsg = '';
  if (winner) statusMsg = winner === 'X' ? `${xName} (X) wins! 🎉` : `${oName} (O) wins! 🎉`;
  else if (isDraw || allFilled) statusMsg = t('ttt.draw');
  else statusMsg = xCount === oCount ? `${xName}'s turn (X)` : `${oName}'s turn (O)`;

  return (
    <div className="flex flex-col items-center gap-4 p-4">
      <div className="text-lg font-semibold text-white">{statusMsg}</div>
      <div className="grid grid-cols-3 gap-2">
        {board.map((cell, idx) => {
          const isWin = winLine?.includes(idx) ?? false;
          let cellClass = 'w-20 h-20 flex items-center justify-center text-3xl font-bold rounded-lg ';
          if (isWin) cellClass += 'bg-yellow-600 ';
          else if (!cell) cellClass += 'bg-gray-700 ';
          else cellClass += 'bg-gray-600 ';

          let textClass = '';
          if (cell === 'X') textClass = 'text-blue-400';
          else if (cell === 'O') textClass = 'text-red-400';

          return (
            <div key={idx} className={cellClass}>
              <span className={textClass}>{cell}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
