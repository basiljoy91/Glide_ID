# Enterprise Attendance Frontend - Next.js

Next.js frontend application for the Enterprise Facial Recognition Attendance & Identity System.

## Features

- **Next.js App Router**: Modern React framework with App Router
- **Tailwind CSS**: Utility-first CSS with dark mode support
- **Zustand**: Lightweight state management with persistence
- **WebRTC Camera**: Face capture with liveness detection
- **Offline Support**: IndexedDB queue with asymmetric encryption
- **Silent Ping**: Backend pre-warming hooks
- **Ambient Light Detection**: Automatic flashlight overlay for dark environments
- **PWA Support**: Progressive Web App for kiosk deployment
- **Responsive Design**: Mobile-first with data cards

## Setup

### 1. Install Dependencies

```bash
npm install
# or
yarn install
# or
pnpm install
```

### 2. Configure Environment

Copy `.env.example` to `.env.local`:

```bash
cp .env.example .env.local
```

Configure:
- `NEXT_PUBLIC_API_URL`: Golang backend URL
- `NEXT_PUBLIC_AI_SERVICE_URL`: Python AI service URL
- `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY`: Public key for offline encryption

### 3. Run Development Server

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000)

## Architecture

```
frontend-nextjs/
├── app/                    # Next.js App Router pages
│   ├── layout.tsx         # Root layout with providers
│   ├── page.tsx           # Home page
│   ├── dashboard/          # Admin dashboard
│   └── kiosk/[code]/      # Kiosk check-in portal
├── components/            # React components
│   ├── camera/           # WebRTC camera components
│   ├── data/             # Data cards and tables
│   └── theme-provider.tsx
├── hooks/                # Custom React hooks
│   ├── useSilentPing.tsx  # Backend pre-warming
│   ├── useAmbientLight.ts # Light detection
│   └── useOfflineQueue.ts # Offline queue management
├── lib/                  # Utilities
│   ├── config.ts         # Configuration
│   └── offline-queue.ts  # IndexedDB queue
├── store/                # Zustand stores
│   └── useStore.ts       # State management
└── public/               # Static assets
    └── manifest.json     # PWA manifest
```

## Key Components

### FaceCamera

WebRTC-based camera component with:
- Face detection guide overlay
- Liveness detection (passive/active)
- Ambient light detection
- Flashlight overlay for dark environments
- Camera permission handling
- User feedback

### Offline Queue

IndexedDB-based offline queue with:
- Asymmetric encryption (public key)
- Automatic sync when online
- Retry mechanism
- Queue statistics

### Silent Ping

Automatically pings backend services:
- Golang API health check
- Python AI service health check
- Runs on page load and every 30 seconds
- Keeps free-tier backends warm

## Kiosk Mode

The kiosk portal (`/kiosk/[code]`) provides:
- Full-screen face capture interface
- Offline support with automatic sync
- PIN fallback option
- Monotonic clock for time reconciliation

## PWA Configuration

The app is configured as a Progressive Web App:
- Installable on mobile devices
- Works offline
- App-like experience
- Manifest file included

## Dark Mode

Dark mode is supported via:
- Tailwind CSS dark mode classes
- System preference detection
- Manual toggle
- Persistent preference storage

## Responsive Design

- Mobile-first approach
- Data cards collapse on mobile
- Touch-friendly interface
- Optimized for kiosk screens

## Deployment

### Vercel

1. Connect your Git repository
2. Set environment variables
3. Deploy automatically on push

### Manual Build

```bash
npm run build
npm start
```

## Browser Support

- Chrome/Edge (recommended)
- Firefox
- Safari (with limitations)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Security

- Offline data encrypted with public key
- HTTPS required for production
- Secure token storage
- Camera permissions handled gracefully

## Troubleshooting

### Camera Not Working

- Check browser permissions
- Ensure HTTPS (required for WebRTC)
- Try different browser

### Offline Queue Not Syncing

- Check network connection
- Verify API URL configuration
- Check browser console for errors

### Dark Mode Not Working

- Clear browser cache
- Check system preferences
- Verify Tailwind configuration

## License

Proprietary - Enterprise Attendance System

