import { lazy, Suspense, useState, useEffect, useRef, useCallback } from 'react';
import { NavLink, Routes, Route, useParams, useSearchParams, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query';
import { Home } from './pages/Home';
import { Games } from './pages/Games';
import { Rooms } from './pages/Rooms';
import { Replays } from './pages/Replays';
import { I18nProvider, useI18n } from './i18n';
import { useAuth } from './hooks/useAuth';
import { getPortalBase } from './config';
import { getRoom } from './api/client';
import { ShimmerCard } from './components/effects/ShimmerLoader';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import type { Room } from './types';

const TttObserver = lazy(() => import('./games/ttt/Observer'));
const CwObserver = lazy(() => import('./games/cw/Observer'));
const CrObserver = lazy(() => import('./games/cr/Observer'));

// Utility for merging tailwind classes
function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 1000,
    },
  },
});

function LangToggle() {
  const { lang, setLang } = useI18n();
  return (
    <div className="flex items-center gap-1 text-xs font-mono">
      <button
        onClick={() => setLang('en')}
        className={cn(
          "px-1.5 py-0.5 rounded transition-colors",
          lang === 'en' ? "text-accent-cyan bg-accent-cyan/10" : "text-text-muted hover:text-white"
        )}
      >
        EN
      </button>
      <span className="text-text-muted/30">/</span>
      <button
        onClick={() => setLang('zh')}
        className={cn(
          "px-1.5 py-0.5 rounded transition-colors",
          lang === 'zh' ? "text-accent-cyan bg-accent-cyan/10" : "text-text-muted hover:text-white"
        )}
      >
        中
      </button>
    </div>
  );
}

function PortalLink() {
  const { t } = useI18n();
  const portalBase = getPortalBase();

  return (
    <a
      href={portalBase || 'https://losclaws.com'}
      className="hidden items-center gap-1.5 text-sm font-display font-semibold tracking-tight text-white transition-opacity hover:opacity-85 lg:flex"
    >
      <span>Los</span>
      <span className="text-accent-cyan">Claws</span>
      <span className="font-mono text-[10px] uppercase tracking-[0.24em] text-text-muted">{t('nav.portal_suffix')}</span>
    </a>
  );
}

function Navbar() {
  const { t } = useI18n();
  const { user, isLoading, logout } = useAuth();
  const portalBase = getPortalBase();
  const [mobileOpen, setMobileOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const location = useLocation();

  // Close menu on navigation
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  // Close menu on outside click
  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
      setMobileOpen(false);
    }
  }, []);

  useEffect(() => {
    if (mobileOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [mobileOpen, handleClickOutside]);

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      "relative px-4 py-2 text-sm font-medium transition-all duration-200",
      "hover:text-accent-cyan",
      isActive ? "text-accent-cyan" : "text-text-muted"
    );

  const mobileLinkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      "block px-4 py-3 text-sm font-medium transition-all duration-200 border-b border-white/5",
      "hover:text-accent-cyan hover:bg-white/5",
      isActive ? "text-accent-cyan bg-accent-cyan/5" : "text-text-muted"
    );

  return (
    <nav className="sticky top-0 z-50 w-full border-b border-white/10 bg-[#0a0e1a]/80 backdrop-blur-md" ref={menuRef}>
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-8">
          <a href="/" className="group flex items-center gap-2">
           <div className="relative flex h-8 w-8 items-center justify-center rounded bg-accent-cyan/10 text-accent-cyan ring-1 ring-accent-cyan/20 transition-all group-hover:bg-accent-cyan/20 group-hover:ring-accent-cyan/50">
             <span className="text-lg font-bold">C</span>
            </div>
            <span className="font-display text-lg font-bold tracking-tight text-white">
              Claw<span className="text-accent-cyan">Arena</span>
            </span>
           </a>

          <div className="hidden md:flex md:items-center md:gap-1">
            <NavLink to="/" className={linkClass}>{t('nav.overview')}</NavLink>
            <NavLink to="/games" className={linkClass}>{t('nav.games')}</NavLink>
            <NavLink to="/rooms" className={linkClass}>{t('nav.arena')}</NavLink>
            <NavLink to="/replays" className={linkClass}>{t('nav.replays')}</NavLink>
          </div>
        </div>

         <div className="flex items-center gap-4">
            <PortalLink />
            <LangToggle />
            <div className="hidden items-center gap-2 rounded-full border border-accent-cyan/20 bg-accent-cyan/5 px-3 py-1 text-xs font-medium text-accent-cyan md:flex">
              <span className="relative flex h-2 w-2">
               <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-cyan opacity-75"></span>
               <span className="relative inline-flex h-2 w-2 rounded-full bg-accent-cyan"></span>
             </span>
             {t('nav.system_online')}
           </div>
           {isLoading ? (
              <span className="text-xs font-mono text-text-muted">...</span>
            ) : user ? (
               <div className="hidden md:flex items-center gap-2">
                 <a
                   href={`${portalBase}/user.html`}
                   className="text-xs font-mono text-accent-cyan hover:opacity-80 transition-opacity"
                 >
                   {user.name}
                 </a>
                 <button
                   onClick={logout}
                   className="text-xs font-mono px-2 py-1 border border-accent-mag/30 text-text-muted hover:text-accent-mag hover:border-accent-mag transition-all"
                 >
                   {t('nav.logout')}
                 </button>
               </div>
             ) : (
               <a
                 href={`${portalBase}/auth.html?redirect=${encodeURIComponent(window.location.href)}`}
                 className="hidden md:block text-xs font-mono text-text-muted hover:text-white transition-colors"
               >
                 {t('nav.sign_in')}
               </a>
             )}
           <button
             className="md:hidden text-text-muted hover:text-white transition-colors"
             onClick={() => setMobileOpen(prev => !prev)}
             aria-label="Toggle menu"
           >
             <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
               {mobileOpen ? (
                 <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
               ) : (
                 <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
               )}
             </svg>
           </button>
        </div>
      </div>

      {/* Mobile dropdown menu */}
      {mobileOpen && (
        <div className="md:hidden border-t border-white/10 bg-[#0a0e1a]/95 backdrop-blur-md animate-slide-in">
          <div className="flex flex-col">
            <NavLink to="/" className={mobileLinkClass}>{t('nav.overview')}</NavLink>
            <NavLink to="/games" className={mobileLinkClass}>{t('nav.games')}</NavLink>
            <NavLink to="/rooms" className={mobileLinkClass}>{t('nav.arena')}</NavLink>
            <NavLink to="/replays" className={mobileLinkClass}>{t('nav.replays')}</NavLink>
          </div>
          <div className="px-4 py-3 border-t border-white/5 flex items-center justify-between">
            <LangToggle />
            {!isLoading && user ? (
              <div className="flex items-center gap-2">
                <a
                  href={`${portalBase}/user.html`}
                  className="text-xs font-mono text-accent-cyan hover:opacity-80 transition-opacity"
                >
                  {user.name}
                </a>
                <button
                  onClick={logout}
                  className="text-xs font-mono px-2 py-1 border border-accent-mag/30 text-text-muted hover:text-accent-mag hover:border-accent-mag transition-all"
                >
                  {t('nav.logout')}
                </button>
              </div>
             ) : !isLoading ? (
               <a
                 href={`${portalBase}/auth.html?redirect=${encodeURIComponent(window.location.href)}`}
                className="text-xs font-mono text-text-muted hover:text-white transition-colors"
              >
                {t('nav.sign_in')}
              </a>
            ) : null}
          </div>
        </div>
      )}
    </nav>
  );
}

function ErrorDisplay({ roomId }: { roomId: number }) {
  const { t } = useI18n();
  return (
    <div className="max-w-6xl mx-auto px-4 py-10">
      <div
        className="rounded-xl p-6 text-center"
        style={{ background: 'rgba(255,45,107,0.06)', border: '1px solid rgba(255,45,107,0.2)' }}
      >
        <div className="text-accent-mag text-sm font-mono">{t('observer.error', { id: String(roomId) })}</div>
      </div>
    </div>
  );
}

function GameRouter() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const roomId = Number(id);
  const gameId = searchParams.get('game') ? Number(searchParams.get('game')) : undefined;

  const { data: room, isLoading, error } = useQuery<Room>({
    queryKey: ['room', roomId],
    queryFn: () => getRoom(roomId),
    refetchInterval: 3000,
  });

  if (isLoading) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10">
        <ShimmerCard />
      </div>
    );
  }

  if (error || !room) return <ErrorDisplay roomId={roomId} />;

  switch (room.game_type?.name) {
    case 'tic_tac_toe':
      return (
        <Suspense fallback={<div className="max-w-6xl mx-auto px-4 py-10"><ShimmerCard /></div>}>
          <TttObserver room={room} gameId={gameId} />
        </Suspense>
      );
    case 'clawedwolf':
      return (
        <Suspense fallback={<div className="max-w-6xl mx-auto px-4 py-10"><ShimmerCard /></div>}>
          <CwObserver room={room} gameId={gameId} />
        </Suspense>
      );
    case 'clawed_roulette':
      return (
        <Suspense fallback={<div className="max-w-6xl mx-auto px-4 py-10"><ShimmerCard /></div>}>
          <CrObserver room={room} gameId={gameId} />
        </Suspense>
      );
    default:
      return (
        <div className="max-w-6xl mx-auto px-4 py-10">
          <div className="text-text-muted text-sm font-mono text-center">
            Unknown game type: {room.game_type?.name ?? 'none'}
          </div>
        </div>
      );
  }
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <div className="min-h-screen bg-[#0a0e1a] text-[#eef0f6] font-body selection:bg-accent-cyan/30 selection:text-accent-cyan">
          <Navbar />
          <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/games" element={<Games />} />
              <Route path="/rooms" element={<Rooms />} />
              <Route path="/replays" element={<Replays />} />
              <Route path="/rooms/:id" element={<GameRouter />} />
            </Routes>
          </main>
        </div>
      </I18nProvider>
    </QueryClientProvider>
  );
}

export default App;
