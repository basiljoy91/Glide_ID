# Enterprise Attendance System - Implementation Plan

This document outlines the comprehensive refactoring to match the enterprise specification.

## Phase 1: Foundation & Public Landing Page ✅ (In Progress)

### 1.1 Install shadcn/ui Components
- [ ] Set up shadcn/ui configuration
- [ ] Install core components (Button, Card, Input, Toast, Skeleton, etc.)
- [ ] Configure theme system with dark/light mode

### 1.2 Public Landing Page
- [ ] Create navigation bar (About, Blog, Contact, Pricing, Admin Login)
- [ ] Hero section with "Get Started" CTA
- [ ] Features section
- [ ] Pricing page with subscription tiers
- [ ] About Us page
- [ ] Blog page (placeholder)
- [ ] Contact Us page

### 1.3 SSO Authentication Setup
- [ ] Create SSO login page (email input → redirect to IdP)
- [ ] Backend SSO handlers (SAML 2.0 / OIDC)
- [ ] SSO callback handler

## Phase 2: Multi-Step Onboarding Wizard

### 2.1 Step 1: Organization Details
- [ ] Company Name input
- [ ] Industry dropdown
- [ ] Estimated employees slider/input

### 2.2 Step 2: Account Creation
- [ ] Primary Admin details form
- [ ] SSO tenant connection or password setup
- [ ] Email verification

### 2.3 Step 3: RBAC & Team Setup
- [ ] Invite other admins via email
- [ ] Role assignment (Admin, HR, Dept Manager)
- [ ] Email invitation system

### 2.4 Step 4: Provisioning & Credentials
- [ ] Backend tenant provisioning
- [ ] Display 10-digit Kiosk Code
- [ ] Success screen with next steps

## Phase 3: Super Admin Portal

### 3.1 Global Dashboard
- [ ] Global metrics cards (organizations, users, check-ins)
- [ ] Billing data visualization
- [ ] Charts for aggregate data

### 3.2 Organization Management
- [ ] List all organizations
- [ ] Upgrade/downgrade subscriptions
- [ ] Activate/deactivate org admin accounts
- [ ] PII protection (no access to user data)

## Phase 4: Organization Admin Portal

### 4.1 Enhanced Dashboard
- [ ] Daily/weekly/monthly presence charts
- [ ] Responsive data cards (collapse on mobile)
- [ ] Real-time metrics

### 4.2 RBAC Implementation
- [ ] Role-based navigation
- [ ] Permission checks on all routes
- [ ] Admin (full access)
- [ ] HR (reports, users, leave)
- [ ] Dept Manager (department-only)

### 4.3 Data Management
- [ ] Department CRUD
- [ ] Employee management (create/edit/delete)
- [ ] Employee form with all fields:
  - First Name, Last Name, Employee ID
  - Email, Phone
  - Department, Designation
  - Date of Joining
  - Shift Timings
  - Emergency Contact

### 4.4 Shift Logic & Timeout
- [ ] Shift length configuration
- [ ] In/Out pair detection
- [ ] Anomaly flagging (missing check-outs)
- [ ] Manual review queue

### 4.5 Integration Hub
- [ ] HRMS connectors (Workday, SAP, BambooHR)
- [ ] Webhook configuration UI
- [ ] API key management
- [ ] Payroll export automation

### 4.6 Device Management
- [ ] Kiosk list with status
- [ ] Heartbeat monitoring (5-minute ping)
- [ ] Revoke Kiosk functionality
- [ ] HMAC key blacklisting

## Phase 5: Advanced Employee Registration

### 5.1 Compliance & Consent
- [ ] Data Privacy Consent screen
- [ ] Legal checkbox before camera
- [ ] Consent tracking in database

### 5.2 Registration Methods
- [ ] In-person registration UI
- [ ] Remote registration link generator
- [ ] One-time, time-sensitive URLs
- [ ] Identity Gateway (SSO/OTP)

### 5.3 Active Liveness Detection
- [ ] Depth perception (move closer/further)
- [ ] TensorFlow.js integration
- [ ] Spoofing prevention

### 5.4 Automated Compliance
- [ ] Termination webhook handler
- [ ] 30-day automated purging
- [ ] GDPR/CCPA compliance

## Phase 6: Enhanced Kiosk Portal

### 6.1 UI Improvements
- [ ] Camera permission handling
- [ ] Dark environment brightness boost
- [ ] Real-time feedback messages
- [ ] White overlay for dark rooms

### 6.2 Offline Mode
- [ ] Network status indicator
- [ ] Monotonic clock implementation
- [ ] Time offset calculation
- [ ] Encrypted IndexedDB storage
- [ ] Automatic sync on reconnect

### 6.3 Physical Access Control
- [ ] IoT door relay integration
- [ ] Wiegand Protocol support
- [ ] Door open signal on match

### 6.4 Biometric Fallback
- [ ] PIN fallback option
- [ ] Lightweight face detection
- [ ] Buddy punching detection
- [ ] Flagged records in HR dashboard

## Phase 7: Security Enhancements

### 7.1 Backend Security
- [ ] mTLS implementation
- [ ] HMAC payload signing
- [ ] Device certificate validation
- [ ] Trusted device registry

### 7.2 Encryption
- [ ] Asymmetric encryption for offline data
- [ ] Public key in frontend
- [ ] Private key in backend
- [ ] AES-256 for vectors

### 7.3 Database Security
- [ ] Row-Level Security (RLS) policies
- [ ] Tenant isolation enforcement
- [ ] PII encryption at rest

## Phase 8: UI/UX Improvements

### 8.1 Loading States
- [ ] Skeleton loaders for all data fetches
- [ ] Replace spinning wheels
- [ ] Progressive loading

### 8.2 Notifications
- [ ] Toast notification system
- [ ] Success/warning/error states
- [ ] Non-intrusive placement

### 8.3 Navigation
- [ ] Logout in top-right (avatar menu)
- [ ] Logout in bottom of left nav
- [ ] Consistent navigation patterns

### 8.4 Responsive Design
- [ ] Mobile-first approach
- [ ] Data tables → cards on mobile
- [ ] Touch-friendly interfaces
- [ ] Kiosk screen optimization

## Phase 9: AI & Biometric Enhancements

### 9.1 Continuous Learning
- [ ] 98%+ confidence threshold
- [ ] Vector blending (95% old, 5% new)
- [ ] Weekly/monthly throttling
- [ ] Update tracking

### 9.2 HNSW Indexing
- [ ] pgvector extension setup
- [ ] HNSW index creation
- [ ] Fast 1:N similarity search
- [ ] Performance optimization

### 9.3 One-Way Vectorization
- [ ] Raw image deletion
- [ ] Vector-only storage
- [ ] AES-256 encryption
- [ ] No photo retention

## Implementation Order

1. **Week 1**: Foundation (shadcn/ui, landing page, basic auth)
2. **Week 2**: Onboarding wizard, Super Admin portal
3. **Week 3**: Organization Admin portal, RBAC
4. **Week 4**: Kiosk enhancements, offline mode
5. **Week 5**: Security, compliance, integrations
6. **Week 6**: UI/UX polish, testing, optimization

## Key Files to Create/Modify

### Frontend
- `app/(public)/` - Public landing pages
- `app/(auth)/` - Authentication pages
- `app/(admin)/super-admin/` - Super Admin portal
- `app/(admin)/org-admin/` - Organization Admin portal
- `app/kiosk/[code]/` - Enhanced kiosk portal
- `components/ui/` - shadcn/ui components
- `components/skeletons/` - Skeleton loader
- `lib/sso/` - SSO integration
- `lib/integrations/` - HRMS connectors

### Backend
- `internal/handlers/sso.go` - SSO handlers
- `internal/handlers/onboarding.go` - Onboarding API
- `internal/handlers/integrations.go` - HRMS webhooks
- `internal/services/device.go` - Device management
- `internal/middleware/mtls.go` - mTLS validation

### Database
- `migrations/003_add_sso_config.sql` - SSO configuration
- `migrations/004_add_device_certificates.sql` - Device certs
- `migrations/005_add_consent_tracking.sql` - Consent tracking

