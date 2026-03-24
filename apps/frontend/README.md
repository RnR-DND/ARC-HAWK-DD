# ARC-Hawk Frontend

The official dashboard for ARC-Hawk, built with **Next.js 14**, **TypeScript 5.3**, **Tailwind CSS**, and **ReactFlow**.

## 🌟 Features

- **Dashboard Overview**: Real-time risk summary, scan status, and key metrics
- **Findings Explorer**: Comprehensive data grid with advanced filtering (Risk, Asset, Status, PII Type)
- **Lineage Graph**: Interactive **ReactFlow** visualization of data movement across systems
- **Compliance Center**: DPDPA 2023 readiness tracking and consent management
- **Remediation**: One-click actions to mask or delete sensitive data
- **Asset Inventory**: Complete view of all data sources and assets
- **Real-Time Updates**: WebSocket connection for live scan progress
- **Responsive Design**: Mobile-friendly interface

## 🛠️ Tech Stack

- **Framework**: [Next.js 14](https://nextjs.org/) (App Router)
- **Language**: [TypeScript 5.3](https://www.typescriptlang.org/)
- **Styling**: [Tailwind CSS 3.4](https://tailwindcss.com/) + CSS Modules
- **State Management**: React Hooks + Context API
- **UI Components**: Custom components + [Radix UI](https://www.radix-ui.com/)
- **Visualization**: [ReactFlow](https://reactflow.dev/) (lineage graphs), [Recharts](https://recharts.org/) (charts)
- **Icons**: [Lucide React](https://lucide.dev/)
- **Utilities**: [date-fns](https://date-fns.org/), [clsx](https://github.com/lukeed/clsx), [tailwind-merge](https://github.com/dcastil/tailwind-merge)

## 📂 Project Structure

```
apps/frontend/
├── app/                          # Next.js App Router
│   ├── page.tsx                  # Dashboard home
│   ├── layout.tsx                # Root layout
│   ├── globals.css               # Global styles
│   ├── findings/                 # Findings explorer
│   │   └── page.tsx
│   ├── lineage/                  # Lineage visualization
│   │   └── page.tsx
│   ├── compliance/               # Compliance center
│   │   └── page.tsx
│   ├── remediation/              # Remediation actions
│   │   └── page.tsx
│   ├── scans/                    # Scan management
│   │   └── page.tsx
│   ├── asset-inventory/          # Asset inventory
│   │   └── page.tsx
│   ├── analytics/                # Analytics & reporting
│   │   └── page.tsx
│   └── settings/                 # Settings
│       └── page.tsx
├── components/                   # React components
│   ├── ui/                       # UI kit (Button, Card, Modal, etc.)
│   ├── layout/                   # Layout components
│   │   ├── GlobalLayout.tsx
│   │   ├── Navigation.tsx
│   │   └── Sidebar.tsx
│   ├── dashboard/                # Dashboard-specific widgets
│   ├── findings/                 # Finding components
│   ├── lineage/                  # Lineage components
│   ├── remediation/              # Remediation modals
│   └── sources/                  # Connection wizards
├── services/                     # API clients
│   ├── api.ts                    # Base API client
│   ├── dashboard.api.ts          # Dashboard endpoints
│   ├── findings.api.ts           # Findings endpoints
│   ├── lineage.api.ts            # Lineage endpoints
│   ├── remediation.api.ts        # Remediation endpoints
│   ├── connections.api.ts        # Connection endpoints
│   ├── assets.api.ts             # Asset endpoints
│   ├── compliance.api.ts         # Compliance endpoints
│   └── websocket.ts              # WebSocket client
├── hooks/                        # Custom React hooks
│   ├── useScans.ts               # Scan data management
│   ├── useFindings.ts            # Findings data management
│   └── useWebSocket.ts           # WebSocket connection
├── types/                        # TypeScript definitions
│   ├── finding.types.ts          # Finding models
│   ├── asset.types.ts            # Asset models
│   ├── lineage.types.ts          # Lineage models
│   └── scan.types.ts             # Scan models
├── utils/                        # Utility functions
│   ├── formatters.ts             # Data formatters
│   └── validators.ts             # Input validators
├── design-system/                # Design tokens
│   └── tokens.ts                 # Colors, spacing, typography
├── public/                       # Static assets
└── styles/                       # Additional styles
    └── components.css
```

## 🚀 Getting Started

### Prerequisites

- **Node.js 18+** (LTS recommended)
- **npm 9+** or **yarn 1.22+** or **pnpm**
- Running **backend server** (see [backend README](../backend/README.md))

### Installation

```bash
# 1. Install dependencies
npm install

# 2. Create environment file
cp .env.example .env.local

# 3. Edit .env.local with your configuration
```

### Environment Variables

Create `.env.local`:

```bash
# API Configuration (Required)
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws

# Optional: Feature Flags
NEXT_PUBLIC_ENABLE_ANALYTICS=true
NEXT_PUBLIC_DEBUG_MODE=false

# Optional: External Services
# NEXT_PUBLIC_SENTRY_DSN=...
# NEXT_PUBLIC_GA_ID=...
```

### Development Server

```bash
# Start development server
npm run dev

# Server will start on http://localhost:3000
```

### Production Build

```bash
# Create production build
npm run build

# Start production server
npm start
```

## 📱 Pages

### Dashboard (`/`)

The main dashboard providing an overview of:
- Total findings by severity
- Active scans and their status
- Risk score summary
- Recent activity
- Quick actions

### Findings (`/findings`)

Comprehensive findings explorer with:
- Data grid with sorting and pagination
- Advanced filters (PII type, asset, status, severity)
- Bulk actions
- Export functionality
- Detailed finding view

### Lineage (`/lineage`)

Interactive lineage visualization:
- ReactFlow graph showing data flow
- Zoom and pan controls
- Node details on click
- Export to PNG/SVG
- Impact analysis

### Compliance (`/compliance`)

DPDPA 2023 compliance tracking:
- Consent status overview
- Retention policy monitoring
- Compliance score
- Report generation
- Data principal requests

### Remediation (`/remediation`)

Remediation actions:
- Available actions for findings
- Masking preview
- Execution history
- One-click remediation

### Asset Inventory (`/asset-inventory`)

Asset management:
- List of all assets
- Asset details and metadata
- Scan history per asset
- Add/edit assets

### Scans (`/scans`)

Scan management:
- Active and completed scans
- Scan details and progress
- Trigger new scans
- View scan results

## 🔧 Development

### Code Style

We use ESLint and Prettier for code formatting:

```bash
# Run linter
npm run lint

# Fix linting issues
npm run lint -- --fix

# Format code
npx prettier --write .
```

### Type Checking

```bash
# Run TypeScript compiler
npx tsc --noEmit
```

### Testing

```bash
# Run tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage
```

### Building

```bash
# Build for production
npm run build

# Analyze bundle size
npm run analyze
```

## 🎨 Design System

### Colors

```typescript
// Primary palette
primary: {
  50: '#eff6ff',
  100: '#dbeafe',
  500: '#3b82f6',
  600: '#2563eb',
  700: '#1d4ed8',
}

// Semantic colors
success: '#10b981',
warning: '#f59e0b',
error: '#ef4444',
info: '#3b82f6',
```

### Typography

- **Headings**: Inter, sans-serif
- **Body**: Inter, sans-serif
- **Monospace**: JetBrains Mono (for code)

### Components

All UI components are in `components/ui/`:

- `Button` - Action buttons with variants
- `Card` - Content containers
- `Modal` - Dialog overlays
- `Table` - Data tables with sorting
- `Input` - Form inputs
- `Select` - Dropdown selects
- `Badge` - Status indicators
- `Alert` - Notification banners

## 🔌 API Integration

### Base API Client

Located in `services/api.ts`:

```typescript
import { api } from '@/services/api';

// GET request
const findings = await api.get('/findings');

// POST request
const result = await api.post('/scans/trigger', { asset_id: '...' });

// PATCH request
await api.patch(`/findings/${id}`, { status: 'resolved' });
```

### WebSocket

Real-time updates via WebSocket:

```typescript
import { useWebSocket } from '@/hooks/useWebSocket';

function ScanProgress({ scanId }: { scanId: string }) {
  const { lastMessage } = useWebSocket(`scan:${scanId}`);
  
  return (
    <div>
      Progress: {lastMessage?.progress}%
    </div>
  );
}
```

### Error Handling

API errors are handled globally:

```typescript
// components/ErrorBoundary.tsx
// Shows user-friendly error messages
// Logs errors to console (and Sentry in production)
```

## 📊 State Management

### Local State

Use React hooks for local component state:

```typescript
const [isLoading, setIsLoading] = useState(false);
const [data, setData] = useState<Finding[]>([]);
```

### Global State

Use Context API for global state:

```typescript
// contexts/ScanContext.tsx
const { scans, refreshScans } = useScanContext();
```

### Server State

Use SWR or React Query for server state:

```typescript
import useSWR from 'swr';

const { data: findings, error } = useSWR('/api/v1/findings', fetcher);
```

## 🧪 Testing

### Unit Tests

```typescript
// components/__tests__/Button.test.tsx
import { render, screen } from '@testing-library/react';
import { Button } from '../ui/Button';

test('renders button with text', () => {
  render(<Button>Click me</Button>);
  expect(screen.getByText('Click me')).toBeInTheDocument();
});
```

### Integration Tests

```typescript
// app/__tests__/findings.test.tsx
test('loads and displays findings', async () => {
  render(<FindingsPage />);
  
  const findings = await screen.findByTestId('findings-grid');
  expect(findings).toBeInTheDocument();
});
```

### E2E Tests

```bash
# Using Playwright (optional setup)
npm run test:e2e
```

## 🚀 Deployment

### Docker

```bash
# Build Docker image
docker build -t arc-hawk-frontend .

# Run container
docker run -p 3000:3000 arc-hawk-frontend
```

### Environment-Specific Builds

```bash
# Development
npm run dev

# Staging
NEXT_PUBLIC_ENV=staging npm run build

# Production
NEXT_PUBLIC_ENV=production npm run build
```

### Static Export

```bash
# Export as static HTML
npm run build

# Output in dist/ directory
```

## 📈 Performance

### Optimization Techniques

1. **Code Splitting**: Automatic with Next.js
2. **Image Optimization**: Use `next/image`
3. **Font Optimization**: Use `next/font`
4. **Lazy Loading**: Dynamic imports for heavy components

```typescript
// Lazy load heavy components
const LineageGraph = dynamic(() => import('@/components/lineage/LineageGraph'), {
  loading: () => <LoadingSpinner />,
});
```

### Bundle Analysis

```bash
# Analyze bundle size
npm run analyze
```

## 🔒 Security

### Environment Variables

- Never commit `.env.local`
- Use `NEXT_PUBLIC_` prefix only for client-side variables
- Keep API keys and secrets server-side only

### XSS Prevention

- All user input is sanitized
- React's automatic escaping
- Content Security Policy headers

### CORS

API calls handle CORS via backend configuration.

## 🐛 Troubleshooting

### Build Errors

```bash
# Clear Next.js cache
rm -rf .next

# Clear node_modules
rm -rf node_modules
npm install
```

### API Connection Issues

```bash
# Verify backend is running
curl http://localhost:8080/api/v1/health

# Check environment variables
cat .env.local
```

### WebSocket Issues

```bash
# Check WebSocket URL in .env.local
# Verify backend WebSocket endpoint is working
```

## 📚 Additional Resources

- [Next.js Documentation](https://nextjs.org/docs)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [ReactFlow Documentation](https://reactflow.dev/docs)
- [TypeScript Documentation](https://www.typescriptlang.org/docs)

## 🤝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

## 📝 License

Apache License 2.0 - See [LICENSE](../LICENSE) for details.
