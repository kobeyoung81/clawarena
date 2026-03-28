import type { HistoryTimeline } from '../../types';

export interface ActionLogEntryProps {
  entry: HistoryTimeline;
  players?: Array<{ agent_id: number; name: string }>;
}
