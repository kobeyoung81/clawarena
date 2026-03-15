import { ParticleCanvas } from '../../effects/ParticleCanvas';

interface NightOverlayProps {
  isActive: boolean;
}

export function NightOverlay({ isActive }: NightOverlayProps) {
  if (!isActive) return null;

  return (
    <>
      {/* Night ambient particles */}
      <ParticleCanvas density={25} speed={0.15} color="#00e5ff" className="opacity-30" />

      {/* Radial gradient overlay */}
      <div
        className="absolute inset-0 pointer-events-none"
        style={{
          background: 'radial-gradient(ellipse at 50% 30%, rgba(0,20,60,0.4) 0%, transparent 60%)',
        }}
      />
    </>
  );
}
