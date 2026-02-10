import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { useAuthStore } from '@/stores/authStore';
import {
  Rocket,
  ArrowRight,
  Server,
  Package,
  Play,
  Monitor,
  KeyRound,
  Shield,
} from 'lucide-react';

const capabilities = [
  {
    icon: Play,
    title: 'Deploy from the Marketplace',
    description:
      'Pick an app — WordPress, Gitea, n8n, Metabase, and more. Click deploy. It launches on your infrastructure with a URL, monitoring, and lifecycle controls.',
  },
  {
    icon: Server,
    title: 'Bring Your Own Servers',
    description:
      'Register any Linux server with SSH access. Hoster connects via SSH, installs Docker, and deploys your apps remotely. AWS, DigitalOcean, bare metal — anything works.',
  },
  {
    icon: Package,
    title: 'Create Your Own Templates',
    description:
      'Define a docker-compose spec, set a price, and publish it to the marketplace. Other users can deploy your template with one click.',
  },
];

const highlights = [
  {
    icon: Monitor,
    title: 'Real-time Monitoring',
    description: 'CPU, memory, network stats. Container logs. Lifecycle events. All built in.',
  },
  {
    icon: KeyRound,
    title: 'SSH Key Management',
    description: 'Encrypted key storage. Per-node key assignment. No passwords on your servers.',
  },
  {
    icon: Shield,
    title: 'Self-Hosted & Private',
    description: 'Your data stays on your servers. No vendor lock-in. Full control over everything.',
  },
];

export function LandingPage() {
  const { isAuthenticated } = useAuthStore();

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted/50">
      {/* Navigation */}
      <nav className="container mx-auto px-4 py-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Rocket className="h-8 w-8 text-primary" />
            <span className="text-2xl font-bold">Hoster</span>
          </div>
          <div className="flex items-center gap-4">
            <Link to="/marketplace">
              <Button variant="ghost">Marketplace</Button>
            </Link>
            {isAuthenticated ? (
              <Link to="/dashboard">
                <Button>Dashboard</Button>
              </Link>
            ) : (
              <>
                <Link to="/login">
                  <Button variant="ghost">Sign In</Button>
                </Link>
                <Link to="/signup">
                  <Button>Get Started</Button>
                </Link>
              </>
            )}
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="container mx-auto px-4 py-16 text-center sm:py-24">
        <div className="mx-auto max-w-3xl">
          <h1 className="text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
            Deploy apps to
            <span className="text-primary"> your own servers</span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground">
            Hoster is a self-hosted deployment platform. Pick an app from the marketplace,
            connect your servers, and deploy with one click. You own the infrastructure.
          </p>
          <div className="mt-10 flex items-center justify-center gap-4">
            <Link to="/marketplace">
              <Button size="lg" className="gap-2">
                Browse Apps <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Link to="/signup">
              <Button size="lg" variant="outline">
                Create Account
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* How it works — 3 capabilities */}
      <section className="container mx-auto px-4 py-16">
        <h2 className="text-center text-2xl font-bold sm:text-3xl">How it works</h2>
        <p className="mx-auto mt-3 max-w-xl text-center text-muted-foreground">
          Three things you can do with Hoster, each in under a minute.
        </p>
        <div className="mt-12 grid gap-8 sm:grid-cols-3">
          {capabilities.map((cap, i) => (
            <div key={cap.title} className="rounded-lg border bg-background p-6">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <cap.icon className="h-5 w-5 text-primary" />
              </div>
              <div className="mt-1 text-xs font-medium text-muted-foreground">Step {i + 1}</div>
              <h3 className="mt-2 text-lg font-semibold">{cap.title}</h3>
              <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
                {cap.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* Highlights */}
      <section className="container mx-auto px-4 py-16">
        <div className="rounded-lg border bg-muted/30 px-6 py-10 sm:px-10">
          <div className="grid gap-8 sm:grid-cols-3">
            {highlights.map((h) => (
              <div key={h.title} className="flex gap-4">
                <h.icon className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
                <div>
                  <h3 className="font-medium">{h.title}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{h.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="container mx-auto px-4 py-16">
        <div className="rounded-lg bg-primary px-6 py-14 text-center text-primary-foreground">
          <h2 className="text-2xl font-bold sm:text-3xl">Ready to deploy?</h2>
          <p className="mx-auto mt-3 max-w-lg text-base opacity-90">
            Sign up, connect a server, and deploy your first app. Free to start.
          </p>
          <div className="mt-8 flex items-center justify-center gap-4">
            <Link to="/signup">
              <Button size="lg" variant="secondary" className="gap-2">
                Create Free Account <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t bg-muted/50 py-12">
        <div className="container mx-auto px-4">
          <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
            <div className="flex items-center gap-2">
              <Rocket className="h-6 w-6 text-primary" />
              <span className="font-semibold">Hoster</span>
            </div>
            <p className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()} Hoster. All rights reserved.
            </p>
            <div className="flex gap-4 text-sm text-muted-foreground">
              <a href="#" className="hover:text-foreground">Privacy</a>
              <a href="#" className="hover:text-foreground">Terms</a>
              <a href="#" className="hover:text-foreground">Docs</a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
