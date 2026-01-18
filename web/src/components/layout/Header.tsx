import { Link } from 'react-router-dom';
import { Box, User } from 'lucide-react';
import { useIsAuthenticated } from '@/stores/authStore';

export function Header() {
  const isAuthenticated = useIsAuthenticated();

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-14 items-center px-4">
        <Link to="/" className="flex items-center gap-2 font-semibold">
          <Box className="h-6 w-6 text-primary" />
          <span>Hoster</span>
        </Link>

        <nav className="ml-8 flex items-center gap-4 text-sm">
          <Link
            to="/marketplace"
            className="text-muted-foreground transition-colors hover:text-foreground"
          >
            Marketplace
          </Link>
          {isAuthenticated && (
            <>
              <Link
                to="/deployments"
                className="text-muted-foreground transition-colors hover:text-foreground"
              >
                My Deployments
              </Link>
              <Link
                to="/creator"
                className="text-muted-foreground transition-colors hover:text-foreground"
              >
                Creator Dashboard
              </Link>
            </>
          )}
        </nav>

        <div className="ml-auto flex items-center gap-2">
          {isAuthenticated ? (
            <button className="flex h-8 w-8 items-center justify-center rounded-full bg-muted">
              <User className="h-4 w-4" />
            </button>
          ) : (
            <span className="text-sm text-muted-foreground">
              Sign in via APIGate
            </span>
          )}
        </div>
      </div>
    </header>
  );
}
