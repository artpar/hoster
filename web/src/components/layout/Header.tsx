import { Link, useNavigate } from 'react-router-dom';
import { Box, Menu, User, LogOut } from 'lucide-react';
import { useIsAuthenticated, useUser, useAuthStore } from '@/stores/authStore';
import { Button } from '@/components/ui/Button';

interface HeaderProps {
  onMenuToggle: () => void;
}

export function Header({ onMenuToggle }: HeaderProps) {
  const isAuthenticated = useIsAuthenticated();
  const user = useUser();
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/');
  };

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-14 items-center px-4">
        <button
          onClick={onMenuToggle}
          className="mr-3 rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground md:hidden"
          aria-label="Toggle navigation menu"
        >
          <Menu className="h-5 w-5" />
        </button>

        <Link to="/" className="flex items-center gap-2 font-semibold">
          <Box className="h-6 w-6 text-primary" />
          <span>Hoster</span>
        </Link>

        <div className="ml-auto flex items-center gap-2">
          {isAuthenticated ? (
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-2 text-sm">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground">
                  <User className="h-4 w-4" />
                </div>
                <span className="font-medium">
                  {user?.name || user?.email || 'User'}
                </span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleLogout}
                className="gap-1"
              >
                <LogOut className="h-4 w-4" />
                Sign Out
              </Button>
            </div>
          ) : (
            <Link to="/login">
              <Button variant="ghost" size="sm">Sign In</Button>
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
