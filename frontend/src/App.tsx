import { Link, NavLink, useLocation, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Home } from './pages/Home';
import { Games } from './pages/Games';
import { Rooms } from './pages/Rooms';
import { Observer } from './pages/Observer';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

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

function Navbar() {
  const location = useLocation();

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      "relative px-4 py-2 text-sm font-medium transition-all duration-200",
      "hover:text-accent-cyan",
      isActive ? "text-accent-cyan" : "text-text-muted"
    );

  return (
    <nav className="sticky top-0 z-50 w-full border-b border-white/10 bg-[#0a0e1a]/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-8">
          <Link to="/" className="group flex items-center gap-2">
           <div className="relative flex h-8 w-8 items-center justify-center rounded bg-accent-cyan/10 text-accent-cyan ring-1 ring-accent-cyan/20 transition-all group-hover:bg-accent-cyan/20 group-hover:ring-accent-cyan/50">
             <span className="text-lg font-bold">L</span>
           </div>
           <span className="font-display text-lg font-bold tracking-tight text-white">
             Los<span className="text-accent-cyan">Claws</span>
           </span>
          </Link>
          
          <div className="hidden md:flex md:items-center md:gap-1">
            <NavLink to="/" className={linkClass}>Overview</NavLink>
            <NavLink to="/games" className={linkClass}>Games</NavLink>
            <NavLink to="/rooms" className={linkClass}>Arena</NavLink>
          </div>
        </div>

        <div className="flex items-center gap-4">
           <div className="hidden items-center gap-2 rounded-full border border-accent-cyan/20 bg-accent-cyan/5 px-3 py-1 text-xs font-medium text-accent-cyan md:flex">
             <span className="relative flex h-2 w-2">
               <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-cyan opacity-75"></span>
               <span className="relative inline-flex h-2 w-2 rounded-full bg-accent-cyan"></span>
             </span>
             SYSTEM ONLINE
           </div>
           <button className="md:hidden text-text-muted hover:text-white">
             <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6">
               <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
             </svg>
           </button>
        </div>
      </div>
    </nav>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-[#0a0e1a] text-[#eef0f6] font-body selection:bg-accent-cyan/30 selection:text-accent-cyan">
        <Navbar />
        <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/games" element={<Games />} />
            <Route path="/rooms" element={<Rooms />} />
            <Route path="/rooms/:id" element={<Observer />} />
          </Routes>
        </main>
      </div>
    </QueryClientProvider>
  );
}

export default App;

