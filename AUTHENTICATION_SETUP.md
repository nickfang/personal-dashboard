# Authentication Setup Documentation

## Overview
Your personal dashboard now has a complete Cognito authentication system that works seamlessly for a living room display setup.

## Authentication Flow

### 1. User Access Flow
```
User visits / → Checks auth status → Redirects to /login or /dashboard
```

### 2. Login Flow
```
User clicks "Sign in with Google" → Cognito OAuth → Callback → Dashboard
```

### 3. Logout Flow
```
User clicks "Sign Out" → Clears local session → Redirects to Cognito logout → Global sign-out → Redirects to login
```

**Updated**: Now uses proper Cognito global sign-out for complete authentication cleanup.

## Key Features Implemented

### ✅ **Login Page (`/login`)**
- Beautiful, responsive design perfect for large displays
- Google OAuth integration through Cognito
- Automatic redirect to dashboard after successful login
- Manual fallback link if auto-redirect fails
- Stale authentication state detection and cleanup
- Responsive design that scales up to 4K displays

### ✅ **Dashboard (`/dashboard`)**
- Clean navigation bar with user info and logout button
- Responsive grid layout
- Optimized for large TV displays (1920px, 4K+)
- Follows app.css design system with teal colors

### ✅ **Protected Routes**
- Server-side authentication checks
- Automatic redirect to login if not authenticated
- Session management with JWT cookies
- Proper handling of authentication state mismatches

### ✅ **Token Management**
- Automatic handling of expired tokens
- Graceful fallback to login page when authentication fails
- Server-side session verification with JWT_SECRET
- **Global sign-out**: Proper Cognito logout that clears all sessions
- Local sign-out option available for development/testing
- Stale token cleanup when server-side session creation fails

## File Structure

```
src/
├── lib/
│   ├── authConfig.ts         # Cognito configuration
│   └── authService.ts        # Authentication logic
├── routes/
│   ├── +page.svelte         # Root page (redirects based on auth)
│   ├── login/
│   │   └── +page.svelte     # Login page
│   ├── callback/
│   │   └── +page.svelte     # OAuth callback handler
│   ├── (protected)/
│   │   ├── +layout.server.ts # Auth guard
│   │   ├── +layout.svelte   # Protected layout
│   │   └── dashboard/
│   │       └── +page.svelte # Main dashboard
│   └── api/auth/session/
│       └── +server.ts       # Session management API
└── hooks.server.ts          # Server-side auth middleware
```

## Environment Variables Required

```bash
# AWS Cognito Configuration
VITE_AWS_REGION=us-east-1
VITE_COGNITO_USER_POOL_ID=your-user-pool-id
VITE_COGNITO_CLIENT_ID=your-client-id

# OAuth Callback URLs
VITE_COGNITO_CALLBACK_DEV=http://localhost:5173/callback
VITE_COGNITO_CALLBACK_PROD=https://yourdomain.com/callback

# Logout Redirect URLs
VITE_COGNITO_LOGOUT_DEV=http://localhost:5173/login
VITE_COGNITO_LOGOUT_PROD=https://yourdomain.com/login

# Server Session Management
JWT_SECRET=your-secure-jwt-secret
```

## Cognito Setup Required

### 1. **User Pool Configuration**
- Enable Google as an identity provider
- Configure OAuth scopes: `email`, `openid`, `phone`
- Set callback URLs to match your environment variables

### 2. **App Client Settings**
- Enable "Authorization code grant"
- Enable "Allow sign-in using providers"
- Configure allowed OAuth flows and scopes
- **IMPORTANT**: Add sign-out URLs in "Allowed sign-out URLs":
  - `http://localhost:5173/login` (development)
  - `https://yourdomain.com/login` (production)

### 3. **Google OAuth Setup**
- Create Google OAuth app in Google Cloud Console
- Add authorized redirect URIs pointing to Cognito
- Configure Cognito to use Google client ID/secret

## Display Optimization

### **Responsive Breakpoints**
- **Mobile**: < 768px (single column)
- **Standard**: 768px - 1920px (default grid)
- **Large TV**: 1920px+ (enhanced spacing and fonts)
- **4K+**: 3840px+ (maximum scaling)

### **Living Room Features**
- High contrast colors for visibility
- Large, readable fonts
- Intuitive navigation
- Real-time clock display
- Professional gradient backgrounds

## Security Features

- **Server-side authentication**: All protected routes verified server-side
- **JWT sessions**: Secure session management with expiration
- **HTTPS ready**: Configured for production deployment
- **Token expiration handling**: Graceful logout when tokens expire
- **Global sign-out**: Complete Cognito logout prevents session persistence
- **Stale state cleanup**: Automatic detection and cleanup of mismatched auth states
- **CSRF protection**: Proper cookie configuration
- **Environment variable validation**: JWT_SECRET required for session creation

## Recent Updates & Fixes

### **Version 2.0 Authentication Improvements**

#### **Global Sign-Out Implementation**
- **Problem**: Previous sign-out only cleared local tokens, leaving users signed into Cognito
- **Solution**: Now redirects to Cognito logout endpoint for complete global sign-out
- **Benefit**: Prevents automatic re-login and ensures proper security

#### **Stale Authentication State Handling**
- **Problem**: When JWT_SECRET was missing, server sessions failed but client tokens remained
- **Solution**: Login page now detects and clears stale authentication states
- **Benefit**: Automatic recovery from authentication state mismatches

#### **Deployment Environment Variables**
- **Problem**: Missing JWT_SECRET in production caused session creation failures
- **Solution**: All required environment variables now properly configured in Vercel
- **Benefit**: Reliable authentication across all environments

#### **Login UX Improvements**
- **Added**: Manual "Go to Dashboard" fallback link if auto-redirect fails
- **Added**: Better error handling and timeout logic for redirects
- **Added**: Console logging for easier debugging

### **Available Sign-Out Methods**
```typescript
startSignOut()      // Recommended: Global Cognito sign-out
startSignOutLocal() // Local only (development/testing)
forceSignOut()      // Emergency local sign-out
```

## Troubleshooting

### **Common Issues & Solutions**

#### **Issue**: "Server misconfiguration: JWT secret not set"
- **Cause**: Missing JWT_SECRET environment variable
- **Solution**: Add JWT_SECRET to your deployment platform (Vercel/etc.)
- **Status**: ✅ Fixed in production

#### **Issue**: Stuck on login page with "Redirecting to dashboard..."
- **Cause**: Stale client tokens with failed server session
- **Solution**: Login page now auto-detects and clears stale state
- **Status**: ✅ Fixed with automatic recovery

#### **Issue**: User still logged into Cognito after sign-out
- **Cause**: Local-only sign-out didn't clear Cognito session
- **Solution**: Updated to use global Cognito logout endpoint
- **Status**: ✅ Fixed with proper global sign-out

### **Environment Variable Checklist**
- ✅ `JWT_SECRET` - Required for server session creation
- ✅ `VITE_COGNITO_*` - All Cognito configuration variables
- ✅ `WEATHER_API_KEY` - For weather component functionality
- ✅ `GOOGLE_CALENDAR_*` - For calendar integration

## Testing the Setup

## Testing the Setup

### **Complete Authentication Flow Test**

1. **Start development server**: `npm run dev`
2. **Visit**: `http://localhost:5173`
3. **Test authentication flow**:
   - Should redirect to login → sign in with Google → redirect to dashboard
4. **Test global logout**:
   - Click sign out → redirected to Cognito logout → returned to login page
   - Try signing in again → should require full Google authentication (not auto-login)
5. **Test protection**:
   - Try accessing `/dashboard` without auth → should redirect to login with redirectTo parameter
6. **Test recovery**:
   - If you encounter stale auth state → login page should auto-detect and clear it

### **Production Deployment Verification**

1. ✅ Environment variables configured in deployment platform
2. ✅ JWT_SECRET set for session creation
3. ✅ Cognito callback URLs match production domain
4. ✅ HTTPS enabled for secure cookie handling
5. ✅ All authentication flows tested in production environment

Your authentication system is now ready for a professional living room dashboard display! 🚀
