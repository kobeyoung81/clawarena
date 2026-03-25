const ROLE_EMOJI: Record<string, string> = {
  clawedwolf: '🐺',
  seer:       '👁',
  guard:      '🛡',
  villager:   '👤',
  witch:      '🧙',
};

const ROLE_COLORS: Record<string, { bg: string; border: string; text: string }> = {
  clawedwolf: { bg: 'rgba(255,45,107,0.2)', border: 'rgba(255,45,107,0.6)', text: '#ff2d6b' },
  seer:     { bg: 'rgba(179,136,255,0.2)', border: 'rgba(179,136,255,0.6)', text: '#b388ff' },
  guard:    { bg: 'rgba(0,230,118,0.2)', border: 'rgba(0,230,118,0.6)', text: '#00e676' },
  villager: { bg: 'rgba(100,181,246,0.2)', border: 'rgba(100,181,246,0.5)', text: '#64b5f6' },
  witch:    { bg: 'rgba(224,64,251,0.2)', border: 'rgba(224,64,251,0.6)', text: '#e040fb' },
};

interface RoleRevealProps {
  role: string;
  revealed: boolean;
}

export function RoleReveal({ role, revealed }: RoleRevealProps) {
  const colors = ROLE_COLORS[role] ?? { bg: 'rgba(0,229,255,0.15)', border: 'rgba(0,229,255,0.4)', text: '#00e5ff' };
  const emoji = ROLE_EMOJI[role] ?? '❓';

  return (
    <div
      className="relative w-14 h-14 rounded-full flex items-center justify-center text-2xl"
      style={{
        background: revealed ? colors.bg : 'rgba(20,24,40,0.8)',
        border: `2px solid ${revealed ? colors.border : 'rgba(255,255,255,0.1)'}`,
        boxShadow: revealed ? `0 0 12px ${colors.border}` : 'none',
        animation: revealed ? 'roleReveal 0.6s ease' : 'none',
        transformStyle: 'preserve-3d',
      }}
    >
      {revealed ? (
        <span style={{ filter: `drop-shadow(0 0 4px ${colors.text})` }}>{emoji}</span>
      ) : (
        <span className="text-text-muted text-sm">?</span>
      )}
    </div>
  );
}
