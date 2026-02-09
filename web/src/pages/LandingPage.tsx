import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { useAuthStore } from '@/stores/authStore';
import {
  Rocket,
  Shield,
  Zap,
  Globe,
  Package,
  BarChart3,
  ArrowRight,
  CheckCircle2
} from 'lucide-react';

const features = [
  {
    icon: Rocket,
    title: 'One-Click Deployments',
    description: 'Deploy applications from our marketplace with a single click. No complex configurations required.',
  },
  {
    icon: Shield,
    title: 'Secure by Default',
    description: 'Automatic HTTPS, network isolation, and secure authentication for all your deployments.',
  },
  {
    icon: Zap,
    title: 'Instant Scaling',
    description: 'Scale your deployments up or down based on demand. Pay only for what you use.',
  },
  {
    icon: Globe,
    title: 'Global Edge Network',
    description: 'Deploy close to your users with our distributed infrastructure for low latency.',
  },
  {
    icon: Package,
    title: 'Template Marketplace',
    description: 'Choose from hundreds of pre-built templates or create your own for the community.',
  },
  {
    icon: BarChart3,
    title: 'Usage Analytics',
    description: 'Monitor your deployments with real-time metrics and comprehensive analytics.',
  },
];

const pricingPlans = [
  {
    name: 'Starter',
    price: 'Free',
    description: 'Perfect for trying out Hoster',
    features: [
      '1 deployment',
      '1 vCPU',
      '1 GB RAM',
      '5 GB storage',
      'Community support',
    ],
    cta: 'Get Started',
    highlighted: false,
  },
  {
    name: 'Pro',
    price: '$29/mo',
    description: 'For growing teams and projects',
    features: [
      '10 deployments',
      '4 vCPUs',
      '8 GB RAM',
      '50 GB storage',
      'Priority support',
      'Custom domains',
    ],
    cta: 'Start Free Trial',
    highlighted: true,
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    description: 'For large organizations',
    features: [
      'Unlimited deployments',
      'Dedicated resources',
      'SLA guarantee',
      '24/7 support',
      'SSO integration',
      'On-premise option',
    ],
    cta: 'Contact Sales',
    highlighted: false,
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
              <Link to="/deployments">
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

      {/* Hero Section */}
      <section className="container mx-auto px-4 py-20 text-center">
        <div className="mx-auto max-w-3xl">
          <h1 className="text-5xl font-bold tracking-tight sm:text-6xl">
            Deploy Applications
            <span className="text-primary"> in Seconds</span>
          </h1>
          <p className="mt-6 text-xl text-muted-foreground">
            Hoster is a self-hosted deployment platform with a template marketplace.
            One-click deploy any application onto your own infrastructure.
          </p>
          <div className="mt-10 flex items-center justify-center gap-4">
            <Link to="/marketplace">
              <Button size="lg" className="gap-2">
                Explore Marketplace <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Link to="/signup">
              <Button size="lg" variant="outline">
                Start Free
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="container mx-auto px-4 py-20">
        <div className="text-center">
          <h2 className="text-3xl font-bold">Everything You Need</h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Powerful features to deploy and manage your applications
          </p>
        </div>
        <div className="mt-16 grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <Card key={feature.title} className="border-0 bg-card/50 backdrop-blur">
              <CardHeader>
                <feature.icon className="h-12 w-12 text-primary" />
                <CardTitle className="mt-4">{feature.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <CardDescription className="text-base">
                  {feature.description}
                </CardDescription>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Pricing Section */}
      <section className="container mx-auto px-4 py-20">
        <div className="text-center">
          <h2 className="text-3xl font-bold">Simple, Transparent Pricing</h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Choose the plan that works for you
          </p>
        </div>
        <div className="mt-16 grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {pricingPlans.map((plan) => (
            <Card
              key={plan.name}
              className={plan.highlighted ? 'border-primary shadow-lg' : ''}
            >
              <CardHeader>
                <CardTitle>{plan.name}</CardTitle>
                <div className="mt-4">
                  <span className="text-4xl font-bold">{plan.price}</span>
                </div>
                <CardDescription>{plan.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <ul className="space-y-3">
                  {plan.features.map((feature) => (
                    <li key={feature} className="flex items-center gap-2">
                      <CheckCircle2 className="h-4 w-4 text-primary" />
                      <span className="text-sm">{feature}</span>
                    </li>
                  ))}
                </ul>
                <Button
                  className="mt-6 w-full"
                  variant={plan.highlighted ? 'default' : 'outline'}
                >
                  {plan.cta}
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* CTA Section */}
      <section className="container mx-auto px-4 py-20">
        <Card className="bg-primary text-primary-foreground">
          <CardContent className="py-16 text-center">
            <h2 className="text-3xl font-bold">Ready to Get Started?</h2>
            <p className="mx-auto mt-4 max-w-xl text-lg opacity-90">
              Join thousands of developers deploying applications with Hoster.
              Start free and scale as you grow.
            </p>
            <div className="mt-8 flex items-center justify-center gap-4">
              <Link to="/signup">
                <Button size="lg" variant="secondary" className="gap-2">
                  Create Free Account <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
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
