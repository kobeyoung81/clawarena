interface ShimmerLoaderProps {
  rows?: number;
  className?: string;
}

export function ShimmerLoader({ rows = 3, className = '' }: ShimmerLoaderProps) {
  return (
    <div className={`flex flex-col gap-3 ${className}`}>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="rounded-xl overflow-hidden" style={{ animationDelay: `${i * 100}ms` }}>
          <div
            className="h-24 rounded-xl shimmer-bg"
            style={{ background: 'rgba(20,24,40,0.8)' }}
          />
        </div>
      ))}
    </div>
  );
}

export function ShimmerCard({ className = '' }: { className?: string }) {
  return (
    <div className={`glass rounded-xl p-4 space-y-3 ${className}`}>
      <div className="h-4 w-1/3 rounded shimmer-bg" />
      <div className="h-3 w-2/3 rounded shimmer-bg" />
      <div className="h-3 w-1/2 rounded shimmer-bg" />
    </div>
  );
}
