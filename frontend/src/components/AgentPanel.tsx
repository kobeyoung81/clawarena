import { useI18n } from '../i18n';
import type { RoomAgent, PendingAction, HistoryPlayer } from '../types';

const ROLE_EMOJI: Record<string, string> = {
  clawedwolf: '🐺',
  seer:       '👁',
  guard:      '🛡',
  villager:   '👤',
  witch:      '🧙',
};

const ROLE_COLOR: Record<string, { bar: string; glow: string; text: string }> = {
  clawedwolf: { bar: '#ff2d6b', glow: 'rgba(255,45,107,0.35)', text: '#ff2d6b' },
  seer:     { bar: '#b388ff', glow: 'rgba(179,136,255,0.35)', text: '#b388ff' },
  guard:    { bar: '#00e676', glow: 'rgba(0,230,118,0.35)', text: '#00e676' },
  villager: { bar: '#64b5f6', glow: 'rgba(100,181,246,0.30)', text: '#64b5f6' },
  witch:    { bar: '#e040fb', glow: 'rgba(224,64,251,0.35)', text: '#e040fb' },
};

const DEFAULT_COLOR = { bar: '#00e5ff', glow: 'rgba(0,229,255,0.25)', text: '#00e5ff' };

interface AgentPanelProps {
  agents: RoomAgent[];
  pendingAction: PendingAction | null;
  replayPlayers?: HistoryPlayer[];
}

export function AgentPanel({ agents, pendingAction, replayPlayers }: AgentPanelProps) {
  const { t } = useI18n();
  const roleMap = replayPlayers
    ? Object.fromEntries(replayPlayers.map(p => [p.agent_id, p.role]))
    : {};

  return (
    <div className="glass rounded-xl border-white/8 overflow-hidden">
      <div className="px-3 py-2 border-b border-white/6 flex items-center gap-2">
        <span className="text-xs font-mono font-semibold text-text-muted uppercase tracking-widest">
          {t('agent_panel.title')}
        </span>
        <span className="text-text-muted/40 text-xs font-mono">{agents.length}</span>
      </div>

      <div className="p-2 flex flex-col gap-1.5">
        {agents.map(ra => {
          const isCurrentTurn = pendingAction?.player_id === ra.agent?.id;
          const role = roleMap[ra.agent?.id] ?? undefined;
          const colors = role ? (ROLE_COLOR[role] ?? DEFAULT_COLOR) : DEFAULT_COLOR;

          return (
            <div
              key={ra.id}
              className="relative rounded-lg overflow-hidden transition-all duration-300"
              style={{
                background: isCurrentTurn
                  ? 'rgba(0,229,255,0.06)'
                  : 'rgba(255,255,255,0.03)',
                border: isCurrentTurn
                  ? '1px solid rgba(0,229,255,0.35)'
                  : '1px solid rgba(255,255,255,0.06)',
                boxShadow: isCurrentTurn ? '0 0 12px rgba(0,229,255,0.15)' : 'none',
              }}
            >
              {/* Left role color bar */}
              <div
                className="absolute left-0 top-0 bottom-0 w-0.5"
                style={{ background: role ? colors.bar : 'rgba(255,255,255,0.1)' }}
              />

              <div className="flex items-center justify-between px-3 py-2 pl-4">
                {/* Left: slot + name + role */}
                <div className="flex items-center gap-2 min-w-0">
                  <span
                    className="text-[10px] font-mono w-5 text-center shrink-0"
                    style={{ color: 'rgba(255,255,255,0.3)' }}
                  >
                    #{ra.slot}
                  </span>

                  <span className="text-sm font-medium text-text-primary truncate">
                    {ra.agent?.name ?? `Agent ${ra.agent?.id}`}
                  </span>

                  {role && (
                    <span
                      className="text-xs font-mono shrink-0"
                      title={role}
                      style={{ color: colors.text }}
                    >
                      {ROLE_EMOJI[role] ?? '❓'}
                    </span>
                  )}
                </div>

                {/* Right: ELO + score + turn indicator */}
                <div className="flex items-center gap-3 shrink-0">
                  <div className="text-right">
                    <div
                      className="text-xs font-mono leading-none"
                      style={{ color: 'rgba(255,255,255,0.5)', fontFamily: '"JetBrains Mono", monospace' }}
                    >
                      {ra.agent?.elo_rating ?? '—'}
                    </div>
                    <div className="text-[9px] text-text-muted/30 font-mono mt-0.5">ELO</div>
                  </div>

                  {ra.score !== undefined && ra.score !== 0 && (
                    <div className="text-right">
                      <div className="text-xs font-mono text-accent-cyan/70 leading-none">
                        +{ra.score}
                      </div>
                      <div className="text-[9px] text-text-muted/30 font-mono mt-0.5">pts</div>
                    </div>
                  )}

                  {isCurrentTurn && (
                    <div
                      className="flex items-center gap-1 text-[10px] font-mono font-semibold px-1.5 py-0.5 rounded"
                      style={{
                        background: 'rgba(0,229,255,0.12)',
                        color: '#00e5ff',
                        animation: 'speakerPulse 1.5s ease infinite',
                      }}
                    >
                      ⚡
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        })}

        {agents.length === 0 && (
          <div className="py-6 text-center text-text-muted/30 text-xs font-mono italic">
            {t('agent_panel.empty')}
          </div>
        )}
      </div>
    </div>
  );
}
