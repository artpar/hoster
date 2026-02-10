import { NavLink } from 'react-router-dom';
import { Store, Layers, Server, KeyRound, LayoutDashboard, Package } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useIsAuthenticated } from '@/stores/authStore';

interface NavItem {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  requiresAuth?: boolean;
}

const navItems: NavItem[] = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard', requiresAuth: true },
  { to: '/marketplace', icon: Store, label: 'Marketplace' },
  { to: '/deployments', icon: Layers, label: 'My Deployments', requiresAuth: true },
  { to: '/templates', icon: Package, label: 'App Templates', requiresAuth: true },
  { to: '/nodes', icon: Server, label: 'My Nodes', requiresAuth: true },
  { to: '/ssh-keys', icon: KeyRound, label: 'SSH Keys', requiresAuth: true },
];

interface SidebarProps {
  open: boolean;
  onClose: () => void;
}

export function Sidebar({ open, onClose }: SidebarProps) {
  const isAuthenticated = useIsAuthenticated();

  const visibleItems = navItems.filter(
    (item) => !item.requiresAuth || isAuthenticated
  );

  const navContent = (
    <nav className="flex flex-col gap-1 p-4">
      {visibleItems.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          onClick={onClose}
          className={({ isActive }) =>
            cn(
              'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
              isActive
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
            )
          }
        >
          <item.icon className="h-4 w-4" />
          {item.label}
        </NavLink>
      ))}
    </nav>
  );

  return (
    <>
      {/* Desktop sidebar — always visible */}
      <aside className="hidden w-56 shrink-0 border-r border-border bg-muted/30 md:block">
        {navContent}
      </aside>

      {/* Mobile overlay sidebar */}
      {open && (
        <div className="fixed inset-0 z-40 md:hidden">
          {/* Backdrop */}
          <div className="fixed inset-0 bg-black/50" onClick={onClose} />
          {/* Sidebar panel */}
          <aside className="fixed inset-y-0 left-0 w-56 border-r border-border bg-background">
            {navContent}
          </aside>
        </div>
      )}
    </>
  );
}
