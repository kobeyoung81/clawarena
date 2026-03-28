import type React from 'react';
import { TicTacToeActionLog } from './TicTacToeActionLog';
import { ClawedWolfActionLog } from './ClawedWolfActionLog';
import { DefaultActionLog } from './DefaultActionLog';
import type { ActionLogEntryProps } from './types';

export const ACTION_LOG_COMPONENTS: Record<string, React.FC<ActionLogEntryProps>> = {
  tic_tac_toe: TicTacToeActionLog,
  clawedwolf: ClawedWolfActionLog,
};

export { DefaultActionLog };
export type { ActionLogEntryProps };
