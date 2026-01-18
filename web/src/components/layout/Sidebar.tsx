import { NavLink } from 'react-router-dom';
import { Store, Layers, LayoutDashboard } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useIsAuthenticated } from '@/stores/authStore';

interface NavItem {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  requiresAuth?: boolean;
}

const navItems: NavItem[] = [
  { to: '/marketplace', icon: Store, label: 'Marketplace' },
  { to: '/deployments', icon: Layers, label: 'My Deployments', requiresAuth: true },
  { to: '/creator', icon: LayoutDashboard, label: 'Creator', requiresAuth: true },
];

export function Sidebar() {
  const isAuthenticated = useIsAuthenticated();

  const visibleItems = navItems.filter(
    (item) => !item.requiresAuth || isAuthenticated
  );

  return (
    <aside className="hidden w-56 shrink-0 border-r border-border bg-muted/30 md:block">
      <nav className="flex flex-col gap-1 p-4">
        {visibleItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
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
    </aside>
  );
}
