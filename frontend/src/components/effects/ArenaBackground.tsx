export function ArenaBackground({ className = '' }: { className?: string }) {
  return (
    <div className={`absolute inset-0 pointer-events-none overflow-hidden ${className}`}>
      {/* City skyline silhouette */}
      <svg
        className="absolute bottom-0 left-0 right-0 w-full"
        viewBox="0 0 1200 200"
        preserveAspectRatio="none"
        fill="none"
      >
        <path
          d="M0 200 L0 140 L40 140 L40 100 L60 100 L60 120 L80 120 L80 80 L100 80 L100 120 L110 120 L110 60 L130 60 L130 120 L150 120 L150 90 L170 90 L170 50 L185 50 L185 30 L200 30 L200 50 L215 50 L215 90 L230 90 L230 70 L250 70 L250 110 L270 110 L270 80 L290 80 L290 100 L310 100 L310 60 L330 60 L330 80 L360 80 L360 100 L380 100 L380 70 L400 70 L400 40 L415 40 L415 20 L430 20 L430 40 L445 40 L445 70 L470 70 L470 90 L500 90 L500 60 L520 60 L520 80 L540 80 L540 50 L560 50 L560 80 L580 80 L580 100 L610 100 L610 70 L630 70 L630 90 L660 90 L660 60 L680 60 L680 40 L695 40 L695 15 L710 15 L710 40 L725 40 L725 60 L750 60 L750 80 L770 80 L770 100 L800 100 L800 70 L820 70 L820 90 L850 90 L850 60 L870 60 L870 80 L900 80 L900 50 L920 50 L920 80 L940 80 L940 100 L970 100 L970 70 L990 70 L990 90 L1020 90 L1020 60 L1040 60 L1040 80 L1060 80 L1060 50 L1080 50 L1080 30 L1095 30 L1095 10 L1110 10 L1110 30 L1125 30 L1125 50 L1150 50 L1150 80 L1170 80 L1170 100 L1200 100 L1200 200 Z"
          fill="rgba(0, 229, 255, 0.04)"
          stroke="rgba(0, 229, 255, 0.12)"
          strokeWidth="0.5"
        />
        {/* Window glows */}
        {[190, 420, 700, 1100].map((x, i) => (
          <rect key={i} x={x - 3} y={22} width={6} height={4} fill="#00e5ff" opacity="0.6" />
        ))}
        {[185, 415, 695, 1095].map((x, i) => (
          <rect key={i} x={x - 2} y={36} width={4} height={3} fill="#00e5ff" opacity="0.4" />
        ))}
      </svg>

      {/* Scanning line */}
      <div
        className="absolute left-0 right-0 h-px"
        style={{
          background: 'linear-gradient(90deg, transparent, rgba(0,229,255,0.3), transparent)',
          animation: 'fadeUp 4s ease-in-out infinite alternate',
          top: '30%',
        }}
      />
    </div>
  );
}
