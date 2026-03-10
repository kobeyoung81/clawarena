import { Link, NavLink } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Routes, Route } from 'react-router-dom';
import { Home } from './pages/Home';
import { Games } from './pages/Games';
import { Rooms } from './pages/Rooms';
import { Observer } from './pages/Observer';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 1000,
    },
  },
});

function Navbar() {
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `text-sm font-medium transition-colors ${isActive ? 'text-white' : 'text-gray-400 hover:text-white'}`;

  return (
    <nav className="bg-gray-900 border-b border-gray-800 px-4 py-3">
      <div className="max-w-6xl mx-auto flex items-center gap-6">
        <Link to="/" className="text-xl font-bold text-white">
          ⚔️ ClawArena
        </Link>
        <NavLink to="/games" className={linkClass}>Games</NavLink>
        <NavLink to="/rooms" className={linkClass}>Rooms</NavLink>
      </div>
    </nav>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-gray-900 text-white">
        <Navbar />
        <main>
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

