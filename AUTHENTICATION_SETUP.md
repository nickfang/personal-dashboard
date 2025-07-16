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
User clicks "Sign Out" → Clears session → Clears tokens → Redirects to login page
```

**Note**: Currently using direct logout without Cognito hosted UI to avoid redirect issues.

## Key Features Implemented

### ✅ **Login Page (`/login`)**
- Beautiful, responsive design perfect for large displays
- Google OAuth integration through Cognito
- Automatic redirect to dashboard after successful login
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

### ✅ **Token Management**
- Automatic handling of expired tokens
- Graceful fallback to login page when authentication fails
- Server-side session verification
- Improved logout with fallback handling

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
- **CSRF protection**: Proper cookie configuration

## Testing the Setup

1. **Start development server**: `npm run dev`
2. **Visit**: `http://localhost:5173`
3. **Test flow**: Should redirect to login → sign in with Google → redirect to dashboard
4. **Test logout**: Click sign out → should return to login page
5. **Test protection**: Try accessing `/dashboard` without auth → should redirect to login

## Production Deployment

1. Update environment variables for production domain
2. Configure Cognito callback URLs for production
3. Ensure HTTPS is enabled
4. Set secure cookie flags in production

Your authentication system is now ready for a professional living room dashboard display! 🚀
