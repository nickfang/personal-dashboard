# Dashboard Responsive Design - Architecture Plan

## Problem Summary

- **Weather**: Temperature values overflow and get hidden
- **Word of the Day**: Disproportionate whitespace, doesn't scale well
- **Calendar**: Too wide/short (7-column layout), events go off-screen
- **Headers**: Take up valuable space that could show content
- **Constraint**: Kiosk/TV display - scrolling may not be possible

---

## Root Cause Analysis

### Finding: Extensive Responsive CSS Already Exists

The codebase has **500+ lines of responsive CSS** across files:

| File | Breakpoint Types |
|------|------------------|
| `+page.svelte` (dashboard) | Aspect ratio (4), height (1), width (3), 4K (2) |
| `Weather.svelte` | Height (1), width (2) - **missing aspect-ratio** |
| `SatWord.svelte` | Aspect ratio (4), dimensions (1), width (1) + JS logic |

### Problem 1: Viewport Queries vs Allocated Space (Root Cause)

Components use **viewport media queries** but receive **allocated container space** from the grid:

```
┌─────────────────────────────────────────────────────────┐
│ Viewport: 1920 × 1080 (16:9)                            │
├─────────────────────────────────────────────────────────┤
│ Nav: 75px                                               │
├───────────────────────────┬─────────────────────────────┤
│ Weather                   │ SatWord                     │
│ Gets: ~465px height       │ Gets: ~465px height         │
│ Thinks: 1080px viewport   │ Thinks: 1080px viewport     │
│ → Doesn't trigger compact │ → Doesn't trigger compact   │
├───────────────────────────┴─────────────────────────────┤
│ Calendar: Gets ~465px height                            │
└─────────────────────────────────────────────────────────┘
```

**Example**: Weather has `@media (max-height: 768px)` to shrink the chart. On 1920×1080, viewport = 1080px so it **never triggers**, but Weather's actual allocated height is only ~465px → overflow.

### Problem 2: Scattered Responsive Logic

Responsive behavior is split across:
- Dashboard media queries
- Component media queries
- SatWord's JavaScript `updateDisplayMode()` function

No clear ownership of responsive behavior.

### Problem 3: Not Leveraging Svelte

Current code uses imperative JavaScript patterns:
- Manual `addEventListener`/`removeEventListener`
- Imperative update functions
- Direct `window.innerWidth` reads

Instead of Svelte's reactive features.

---

## Component Layout Options

### Weather Component - Current Structure
```
┌─────────────────────────────────────────────────┐
│ [Header: "Weather"]                             │
├────────────────────┬────────────────────────────┤
│ [Icon] Austin, TX  │  [Day1]  [Day2]  [Day3]   │
│        72°F        │   Hi/Lo   Hi/Lo   Hi/Lo   │
│        Sunny       │   rain    rain    rain    │
│   💧 45%  🌧 0"    │                           │
├────────────────────┴────────────────────────────┤
│ [Pressure Graph - fixed 200px height]           │
└─────────────────────────────────────────────────┘
```

**Compact Layout** (chart hidden):
```
┌─────────────────────────────────────┐
│ [Icon]  72°F Sunny  💧45% 🌧0"     │
├─────────────────────────────────────┤
│ Mon 80°/65°  Tue 82°/67°  Wed 79°  │
└─────────────────────────────────────┘
```

---

### SatWord Component - Current Structure
```
┌─────────────────────────────────────────────────┐
│ [Header: "Word of the Day"]                     │
├─────────────────────────────────────────────────┤
│              EPHEMERAL                          │
│           (3 definitions)                       │
├──────────────────┬──────────────────────────────┤
│ (adjective)      │ (adjective)                  │
│ Definition 1...  │ Definition 2...              │
│ "Example..."     │ "Example..."                 │
└──────────────────┴──────────────────────────────┘
```

**Compact Layout** (single definition, cycling):
```
┌─────────────────────────────────────────────────┐
│ EPHEMERAL (adj) - lasting for a very short time │
│ "The ephemeral beauty of cherry blossoms..."    │
│                              [1/3] [←] [→]      │
└─────────────────────────────────────────────────┘
```

---

### Calendar Component

- **Week view** (7 columns): Default for larger containers
- **3-day view** (3 columns): More vertical space per day
- **Auto-detect + manual toggle**: Based on container height with user override

---

## Chosen Approach: CSS Container Queries

### Architecture Principle

**Each component is responsible for adapting to whatever space it's given.**

| Layer | Responsibility |
|-------|----------------|
| `:root` (app.css) | Static design tokens (colors, spacing, fonts) |
| Dashboard | Grid layout + container declarations |
| Components | Full ownership of responsive behavior via container queries |

### Why Container Queries

| Problem | How Container Queries Solve It |
|---------|-------------------------------|
| Viewport vs container mismatch | Components query their **actual** container size |
| Scattered responsive logic | Each component owns its own responsive CSS |
| Complex JS logic | CSS handles responsive, JS only for app logic |

### Trade-off: Accepted

Container queries mean breakpoint values are repeated per-component. **This is intentional.**

Weather hiding its chart at 300px and Calendar switching to 3-day at 350px are **independent design decisions**. They shouldn't be coupled.

---

## Implementation Architecture

### Layer 1: Design Tokens (`:root`)

Static values that **do not change** at breakpoints:

```css
/* src/app.css or global styles */
:root {
  /* Colors - already exist */
  --teal-50: #f0fdfa;
  --teal-600: #0d9488;
  /* ... */

  /* Spacing tokens */
  --space-xs: 0.25rem;
  --space-sm: 0.5rem;
  --space-md: 1rem;
  --space-lg: 1.5rem;
  --space-xl: 2rem;

  /* Font tokens */
  --font-xs: 0.75rem;
  --font-sm: 0.875rem;
  --font-base: 1rem;
  --font-lg: 1.25rem;
  --font-xl: 1.5rem;
  --font-2xl: 2rem;
}
```

### Layer 2: Dashboard (+page.svelte)

Dashboard's **only** responsive responsibilities:

1. Define the grid
2. Declare containers
3. Adjust grid-level spacing at viewport breakpoints

```css
/* Container declarations - components can now query their size */
.weather-section { container-type: size; }
.word-section { container-type: size; }
.calendar-section { container-type: size; }

/* Dashboard-level responsive (grid only, not component internals) */
.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: auto 1fr 1fr;
  gap: var(--space-lg);
}

@media (max-height: 768px) {
  .dashboard-grid {
    gap: var(--space-md);
  }
}
```

**Remove from dashboard:**
- `:global()` rules that reach into components
- Media queries controlling component internals

### Layer 3: Components

Each component owns its responsive behavior via `@container` queries.

**Weather.svelte:**
```css
.weather-container {
  padding: var(--space-md);
}

.weather-chart {
  height: 200px;
}

/* Component adapts to its container */
@container (max-height: 400px) {
  .weather-chart {
    height: 120px;
  }
}

@container (max-height: 300px) {
  .weather-chart {
    display: none;
  }

  .forecast-card {
    min-height: auto;
    padding: var(--space-sm);
  }
}
```

**SatWord.svelte:**
```css
.all-definitions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--space-md);
}

.cycling-view {
  display: none;
}

@container (max-height: 350px) {
  .all-definitions {
    display: none;
  }

  .cycling-view {
    display: block;
  }
}
```

**Calendar2.svelte:**
```css
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(var(--columns, 7), 1fr);
}

/* CSS fallback for auto-detect */
@container (max-height: 400px) {
  .calendar-grid:not(.manual-override) {
    grid-template-columns: repeat(3, 1fr);
  }
}
```

---

## Svelte Best Practices

### Use Reactive Declarations, Not Imperative Functions

**Remove this pattern:**
```javascript
function updateDisplayMode() {
  const aspectRatio = window.innerWidth / window.innerHeight;
  if (aspectRatio > 1.8) {
    showAllDefinitions = false;
  }
  // ... 40 more lines
}

onMount(() => {
  window.addEventListener('resize', updateDisplayMode);
});
```

**With Container Queries, this becomes CSS only.** No JS needed.

For logic that must stay in JS (like Calendar's manual override):

```svelte
<script>
  let containerHeight = 0;

  // Reactive declaration - automatically updates
  $: autoColumns = containerHeight < 400 ? 3 : 7;
</script>

<div bind:clientHeight={containerHeight}>
```

### Use `bind:` Instead of Manual Listeners

**Avoid:**
```javascript
onMount(() => {
  const observer = new ResizeObserver(entries => {
    containerHeight = entries[0].contentRect.height;
  });
  observer.observe(element);
  return () => observer.disconnect();
});
```

**Use:**
```svelte
<div bind:clientHeight={containerHeight}>
```

### Use `class:` and `style:` Directives

**Avoid:**
```svelte
<div class="grid {isCompact ? 'compact' : ''}">
<div style="--columns: {columns}">
```

**Use:**
```svelte
<div class="grid" class:compact={isCompact}>
<div style:--columns={columns}>
```

### Use `<svelte:window>` for Window Bindings

**Avoid:**
```javascript
window.addEventListener('resize', handler);
```

**Use:**
```svelte
<svelte:window bind:innerWidth bind:innerHeight />
```

### Use Stores for Persistent Preferences

```typescript
// src/lib/stores/preferences.ts
import { writable } from 'svelte/store';
import { browser } from '$app/environment';

function createPersistedStore<T>(key: string, initial: T) {
  const stored = browser ? localStorage.getItem(key) : null;
  const store = writable<T>(stored ? JSON.parse(stored) : initial);

  if (browser) {
    store.subscribe(value => localStorage.setItem(key, JSON.stringify(value)));
  }

  return store;
}

export const calendarViewMode = createPersistedStore<'auto' | '3-day' | 'week'>('calendar-view', 'auto');
```

---

## Calendar: Auto-Detect + Manual Toggle

```svelte
<!-- Calendar2.svelte -->
<script lang="ts">
  import { calendarViewMode } from '$lib/stores/preferences';

  let containerHeight = 0;

  // Reactive: combines user preference with auto-detection
  $: columns =
    $calendarViewMode === '3-day' ? 3 :
    $calendarViewMode === 'week' ? 7 :
    containerHeight < 400 ? 3 : 7;  // auto
</script>

<div class="calendar-wrapper" bind:clientHeight={containerHeight}>
  <header class="calendar-header">
    <h2>Calendar</h2>
    <div class="view-toggle">
      <button
        class:active={$calendarViewMode === 'auto'}
        on:click={() => $calendarViewMode = 'auto'}>
        Auto
      </button>
      <button
        class:active={$calendarViewMode === '3-day'}
        on:click={() => $calendarViewMode = '3-day'}>
        3 Day
      </button>
      <button
        class:active={$calendarViewMode === 'week'}
        on:click={() => $calendarViewMode = 'week'}>
        Week
      </button>
    </div>
  </header>

  <div class="calendar-grid" style:--columns={columns}>
    <!-- Day columns rendered based on columns value -->
  </div>
</div>

<style>
  .calendar-grid {
    display: grid;
    grid-template-columns: repeat(var(--columns), 1fr);
  }
</style>
```

---

## Migration Steps

### Step 1: Dashboard Setup (Small Change)

Add container declarations:
```css
.weather-section { container-type: size; }
.word-section { container-type: size; }
.calendar-section { container-type: size; }
```

### Step 2: Add Design Tokens (Optional, Non-Breaking)

Add to `app.css`:
```css
:root {
  --space-xs: 0.25rem;
  --space-sm: 0.5rem;
  /* ... */
}
```

### Step 3: Migrate Weather

1. Remove viewport media queries
2. Add container queries
3. Test: resize browser, verify chart hides when container is small

### Step 4: Migrate SatWord

1. Remove `updateDisplayMode()` function
2. Remove resize event listener
3. Add container queries for layout switching
4. Keep cycling JS logic (app behavior, not responsive)

### Step 5: Update Calendar

1. Add 3-day view support
2. Create `calendarViewMode` store
3. Add view toggle UI
4. Implement auto-detect with `bind:clientHeight`

### Step 6: Clean Up Dashboard

1. Remove `:global()` overrides for component internals
2. Remove component-specific media queries
3. Keep only grid-level responsive logic

---

## Summary

| Aspect | Approach |
|--------|----------|
| **Responsive mechanism** | CSS Container Queries |
| **Design tokens** | Static CSS Variables in `:root` |
| **Dashboard role** | Grid layout + container declarations only |
| **Component role** | Full ownership of responsive behavior |
| **Svelte usage** | Reactive declarations, `bind:`, stores |
| **DRY trade-off** | Accept repeated breakpoints for clear ownership |

**Result:**
- Components respond to their **actual** allocated space
- Clear ownership: component overflow → look at that component
- Minimal JavaScript for responsive behavior
- Svelte features properly leveraged
- Dashboard becomes simpler (~50% less CSS)

---

## Target Devices

The dashboard must support these screen sizes:

| Device | Resolution | Aspect Ratio | Grid Layout | Notes |
|--------|------------|--------------|-------------|-------|
| Kiosk | Varies | Varies | 2-column | No scrolling, full viewport |
| 1080p Monitor | 1920×1080 | 16:9 | 2-column | Common desktop |
| 2K Monitor | 2560×1440 | 16:9 | 2-column | Larger desktop |
| 4K Monitor | 3840×2160 | 16:9 | 2-column | High DPI desktop |
| Tablet | ~768-1024px | ~4:3 | 1-column | Portrait or landscape |
| Mobile | ~320-480px | ~9:16 | 1-column | Portrait, scrollable |

### Device-Specific Considerations

**Desktop/Kiosk (1080p, 2K, 4K):**
- 2-column grid layout
- Components use container queries to adapt
- No scrolling expected - everything must fit

**Tablet:**
- 1-column stacked layout (viewport query)
- Components still use container queries for their internals
- May allow some scrolling

**Mobile:**
- 1-column stacked layout (viewport query)
- Scrolling expected
- Components in "full width" mode

---

## Viewport Queries vs Container Queries - What Goes Where

### Viewport Media Queries (Dashboard Level)

These **stay in the dashboard** because they control the overall grid layout:

```css
/* Dashboard grid layout changes */
@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr;  /* Switch to single column */
  }
}

@media (max-width: 768px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
    height: auto;  /* Allow scrolling on mobile */
    overflow: visible;
  }
}

/* Dashboard spacing adjustments */
@media (max-height: 768px) {
  .dashboard-grid {
    gap: var(--space-sm);
    padding: var(--space-sm);
  }
}
```

### Container Queries (Component Level)

These **move to components** because they control component internals:

```css
/* Component adapts to its allocated space */
@container (max-height: 300px) {
  .weather-chart { display: none; }
}

@container (max-width: 400px) {
  .forecast-grid {
    grid-template-columns: 1fr;  /* Stack forecast cards */
  }
}
```

### Summary: What Stays vs Moves

| Query Type | Location | Purpose |
|------------|----------|---------|
| Grid layout changes (1-col vs 2-col) | Dashboard (viewport) | Major layout shift |
| Dashboard gap/padding | Dashboard (viewport) | Overall spacing |
| Nav height | Dashboard (viewport) | Dashboard structure |
| Component internals (chart, fonts, etc.) | Components (container) | Component adaptation |

---

## Fullscreen Pages

The app has fullscreen routes that display a single component:
- `/fullscreen/weather`
- `/fullscreen/sat-word`
- (possibly `/fullscreen/calendar`)

### Fullscreen Container Setup

Each fullscreen page must set up its container:

```svelte
<!-- src/routes/fullscreen/weather/+page.svelte -->
<div class="fullscreen-wrapper">
  <Weather />
</div>

<style>
  .fullscreen-wrapper {
    container-type: size;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
  }
</style>
```

### Fullscreen Behavior

In fullscreen mode:
- Component has full viewport as its container
- Container queries still work (container = viewport)
- Component can show "full" layout with all elements visible
- On smaller screens, component still adapts appropriately

---

## Breakpoint Determination Procedure

Since each component determines its own breakpoints, follow this procedure:

### Step 1: Identify Content Priority

List what the component shows, in priority order:

**Weather example:**
1. Current temperature (must show)
2. Current conditions (must show)
3. Forecast days (important)
4. Humidity/rain details (nice to have)
5. Pressure chart (optional)

### Step 2: Find the Overflow Point

1. Open the component in a resizable container (DevTools or test harness)
2. Set container to a large size (e.g., 600px height)
3. Slowly shrink the container height
4. Note when each element starts to:
   - Overflow its bounds
   - Get cut off
   - Look cramped/unreadable

### Step 3: Define Breakpoints

Add breakpoints **slightly above** where problems occur:

```css
/*
 * Breakpoint: 400px
 * Reason: Chart gets cut off at ~380px, hide before that happens
 */
@container (max-height: 400px) {
  .weather-chart { height: 120px; }
}

/*
 * Breakpoint: 300px
 * Reason: Even small chart doesn't fit below 300px
 */
@container (max-height: 300px) {
  .weather-chart { display: none; }
}
```

### Step 4: Document the Breakpoints

Add comments explaining why each breakpoint exists:

```css
/*
 * BREAKPOINTS:
 * - 400px: Reduce chart height (chart starts getting cut off)
 * - 300px: Hide chart entirely (no room for any chart)
 * - 250px: Switch to compact forecast layout (cards overflow)
 */
```

### Step 5: Test at Target Resolutions

Verify behavior at each target device's allocated component size:

| Device | Approx Component Height | Expected Behavior |
|--------|------------------------|-------------------|
| 4K | ~900px | Full layout, large chart |
| 2K | ~600px | Full layout |
| 1080p | ~465px | Full layout, maybe smaller chart |
| Tablet | ~300-400px | No chart, compact forecast |
| Mobile | Full width, variable height | Scrollable, full content |

---

## Animation Strategy

### Recommended Approach: Instant for Now, Upgrade if Needed

Start with **instant transitions** (no animation). If testing reveals jarring layout shifts, upgrade specific elements.

### Option A: Instant (Default)

```css
@container (max-height: 300px) {
  .weather-chart { display: none; }
}
```

- **Pros:** Simple, no jank, CSS-only
- **Cons:** Can feel abrupt
- **Use when:** Kiosk displays that don't actively resize

### Option B: CSS Fade + Collapse

```css
.weather-chart {
  max-height: 200px;
  opacity: 1;
  overflow: hidden;
  transition: max-height 0.2s ease, opacity 0.2s ease;
}

@container (max-height: 300px) {
  .weather-chart {
    max-height: 0;
    opacity: 0;
  }
}
```

- **Pros:** Smooth, CSS-only
- **Cons:** `max-height` transition needs defined value, can't use `auto`
- **Use when:** Smooth transitions desired without JS

### Option C: Svelte Transitions (Hybrid)

```svelte
<script>
  let containerHeight = 0;
  $: showChart = containerHeight > 300;
</script>

<div bind:clientHeight={containerHeight}>
  {#if showChart}
    <div transition:slide={{ duration: 200 }}>
      <PressureGraph />
    </div>
  {/if}
</div>
```

- **Pros:** Smoothest, works with Svelte's transition system
- **Cons:** Requires JS to detect size
- **Use when:** Polish is important, willing to add JS complexity

### Recommendation

1. **Start with Option A** (instant) for all components
2. **During testing**, identify any transitions that feel jarring
3. **Upgrade to Option B or C** only for those specific elements
4. **Kiosk displays** likely don't need animations (no active resizing)

---

## Testing Strategy

### Manual Testing Checklist

For each component, test at these container sizes:

#### Weather Component

| Container Height | Expected Behavior | Pass/Fail |
|-----------------|-------------------|-----------|
| 500px+ | Full layout: current, forecast, chart |  |
| 400px | Chart reduced to 120px |  |
| 300px | Chart hidden, forecast visible |  |
| 250px | Compact forecast layout |  |

#### SatWord Component

| Container Height | Expected Behavior | Pass/Fail |
|-----------------|-------------------|-----------|
| 400px+ | All definitions in grid |  |
| 350px | Single definition with cycling |  |
| 250px | Compact single definition |  |

#### Calendar Component

| Container Height | Expected Behavior | Pass/Fail |
|-----------------|-------------------|-----------|
| 450px+ | 7-day view (auto) |  |
| 400px | 3-day view (auto) |  |
| Manual override | Respects user selection regardless of size |  |

### Testing Procedure

1. **DevTools Responsive Mode**
   - Open DevTools → Toggle device toolbar
   - Test at preset device sizes (iPhone, iPad, Desktop)
   - Use custom sizes for specific container testing

2. **Component Isolation**
   - Create a test page that renders just one component
   - Wrap in a resizable container
   - Verify container queries work in isolation

3. **Full Dashboard Testing**
   - Test at each target resolution (1080p, 2K, 4K, tablet, mobile)
   - Verify grid layout changes at tablet/mobile breakpoints
   - Verify components adapt within their allocated space

4. **Fullscreen Page Testing**
   - Test each fullscreen route at various sizes
   - Verify container queries work when component is full viewport

### Automated Testing (Optional)

If visual regression testing is desired:

```typescript
// tests/responsive.spec.ts (Playwright)
import { test, expect } from '@playwright/test';

const viewports = [
  { name: '1080p', width: 1920, height: 1080 },
  { name: '2K', width: 2560, height: 1440 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'mobile', width: 375, height: 667 },
];

for (const viewport of viewports) {
  test(`dashboard renders correctly at ${viewport.name}`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto('/dashboard');
    await expect(page).toHaveScreenshot(`dashboard-${viewport.name}.png`);
  });
}
```

---

## What to Remove vs Keep

### Remove from Dashboard

| Item | Reason |
|------|--------|
| `:global(.weather-grid)` overrides | Component owns its responsive behavior |
| `:global(.word-container)` overrides | Component owns its responsive behavior |
| Media queries for component fonts/padding | Move to component container queries |
| Aspect-ratio queries affecting component internals | Move to component container queries |

### Keep in Dashboard

| Item | Reason |
|------|--------|
| `grid-template-columns` media queries | Dashboard grid layout |
| `gap` and `padding` media queries | Dashboard spacing |
| Nav styling media queries | Dashboard structure |
| `container-type: size` declarations | Enables container queries |

### Remove from Components

| Item | Reason |
|------|--------|
| `@media (max-height: ...)` for internals | Replace with `@container` |
| `@media (min-aspect-ratio: ...)` for internals | Replace with `@container` |
| `window.addEventListener('resize', ...)` | Use Svelte `bind:` or CSS |
| `updateDisplayMode()` and similar functions | CSS container queries handle this |

### Keep in Components

| Item | Reason |
|------|--------|
| Base styles (no media/container queries) | Default appearance |
| `@container` queries | Component responsive behavior |
| Svelte `bind:clientHeight` (if needed for JS logic) | Manual override support |
| Application logic (cycling definitions, etc.) | Not responsive-related |

---

## Summary Table

| Aspect | Approach |
|--------|----------|
| **Responsive mechanism** | CSS Container Queries |
| **Design tokens** | Static CSS Variables in `:root` |
| **Dashboard role** | Grid layout + container declarations |
| **Component role** | Full ownership of responsive behavior |
| **Svelte usage** | Reactive declarations, `bind:`, stores |
| **Viewport queries** | Dashboard grid layout only |
| **Container queries** | All component internals |
| **Fullscreen pages** | Same container setup, components adapt |
| **Animations** | Start instant, upgrade if needed |
| **Testing** | Manual checklist + optional Playwright |
| **Target devices** | Kiosk, 1080p, 2K, 4K, tablet, mobile |
