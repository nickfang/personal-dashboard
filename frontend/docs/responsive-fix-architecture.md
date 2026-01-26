# Responsive Layout Fix - Architecture Document

## Current Problem

Looking at the screenshots, content is being cut off across multiple screen sizes. The core issue is a **fundamental conflict** between two CSS features:

### The Conflict

1. **`container-type: size`** - Requires the container to have explicit dimensions on BOTH width AND height. This enables `@container (max-height: ...)` queries.

2. **Scrollable layout** - In single-column mode, we want content to grow naturally and the page to scroll. But with `container-type: size`, the container needs a fixed height.

### What's Happening Now

```
Dashboard sections have:
├── container-type: size  (needs explicit height)
├── min-height: 300-600px (constrains the container)
└── overflow: hidden      (clips content)

Components see:
├── Container height = 300px (the min-height)
├── @container (max-height: 300px) fires
└── Content gets hidden (chart, events, etc.)

But we're in scrollable mode!
└── Hidden content should be visible
└── User should be able to scroll
```

---

## Proposed Fix

### Key Insight

CSS `container-type` has different values:

| Value | Width Queries | Height Queries | Use Case |
|-------|--------------|----------------|----------|
| `size` | ✅ Works | ✅ Works | Fixed layouts where content must fit |
| `inline-size` | ✅ Works | ❌ Disabled | Scrollable layouts where height can grow |

### The Fix

In **scrollable single-column mode** (tablet/mobile), change:
- `container-type: size` → `container-type: inline-size`
- `overflow: hidden` → `overflow: visible`
- Remove `min-height` constraints

### Why This Works

**Multi-column fixed mode (wide screens):**
- Keep `container-type: size`
- Height queries fire → content adapts to fit
- `overflow: hidden` → no scrolling within sections

**Single-column scrollable mode (tablet/mobile):**
- Use `container-type: inline-size`
- Height queries DON'T fire → all content shows
- Width queries still work → stacking/layout changes happen
- Sections grow to fit content naturally
- Page scrolls

---

## Validation Questions

### Q1: Will width-based container queries still work?

**Yes.** `container-type: inline-size` enables width containment. Queries like `@container (max-width: 500px)` will still fire and change layouts (e.g., stacking forecast cards vertically).

### Q2: What happens to height-based queries in scrollable mode?

They **don't fire**. With `inline-size`, there's no height containment, so `@container (max-height: 300px)` won't match. This is exactly what we want - in scrollable mode, show all content.

### Q3: Will components need changes?

**No.** Components already have both width and height queries:
- Width queries → still work for layout changes
- Height queries → won't fire in scrollable mode → full content shown

### Q4: What about the calendar's 3-day/7-day auto mode?

The calendar uses JavaScript (`containerWidth` and `containerHeight` via `bind:clientHeight`) not CSS queries for this logic. We'd need to update the JS logic to detect scrollable mode, OR we could:
- In scrollable mode, default to 3-day for narrow widths (already works via `containerWidth < 700`)
- Let users manually select Week view if they want it

### Q5: Could this break anything?

**Risk:** Very long content makes pages very tall.
**Mitigation:** This is acceptable - scrolling is expected behavior on mobile/tablet.

**Risk:** Components look different in scrollable vs fixed mode.
**Mitigation:** This is intentional - scrollable mode should show more content.

---

## Implementation Plan

### Files to Change

1. **`src/routes/(protected)/dashboard/+page.svelte`**
   - In `@media (max-width: 1024px)` (tablet): Add `container-type: inline-size`, `overflow: visible`, remove `min-height`
   - In `@media (max-width: 768px)` (mobile): Same changes
   - In `@media (max-width: 480px)` (small mobile): Same changes

### What NOT to Change

- Component files (Weather, SatWord, Calendar2) - they already have the right queries
- Fullscreen layouts - they're always fixed/constrained

---

## Testing Matrix

| Screen Size | Layout | container-type | Height Queries | Expected Behavior |
|-------------|--------|----------------|----------------|-------------------|
| > 1024px wide | 2-column | `size` | Active | Content fits, may hide chart/events |
| 769-1024px | 1-column | `inline-size` | Disabled | All content visible, page scrolls |
| < 768px | 1-column | `inline-size` | Disabled | All content visible, stacked layout |
| < 480px | 1-column | `inline-size` | Disabled | Compact stacked, all content visible |

---

## Open Questions

1. **Should we add a visual indicator when content is hidden in fixed mode?** (e.g., "scroll for more" or expand button)

2. ~~**Is the 700px width threshold for calendar 3-day mode correct?** May need adjustment based on testing.~~ → Resolved: Use List view instead (see below)

3. **Should fullscreen pages also be scrollable?** Currently they're fixed - may want to reconsider for mobile.

---

## Calendar List View for Scrollable Mode

### Problem

In narrow/scrollable mode, the calendar grid (even 3-day) has cramped columns where event text wraps vertically character-by-character. A grid layout doesn't work well on narrow screens.

### Solution

Add a **List view** (agenda-style) that shows events in a scrollable vertical list with date dividers. In scrollable mode, "Auto" should default to List view.

### View Mode Options

| Mode | Behavior |
|------|----------|
| Auto | Wide → 7-day grid, Narrow → List view |
| List | Always agenda/list format |
| 3 Day | Always 3-column grid |
| Week | Always 7-column grid |

### List View Layout

```
─── Mon, Jan 5 ───────────────────────────

  ┌ STUDENT HOLIDAY - No School
  │ All day
  └────────────────────────────────────────

  ┌ Nick Meet w/ Rahul
  │ 10:00 AM - 2:00 PM
  └────────────────────────────────────────

─── Tue, Jan 6 ───────────────────────────

  ┌ Ian - Violin Lesson
  │ 5:15 PM - 6:00 PM
  │ 📍 5404 Chevy Circle Austin, TX 78723
  └────────────────────────────────────────

─── Wed, Jan 7 ───────────────────────────

  No events

```

### Implementation

#### 1. Update Preferences Store
**File:** `src/lib/stores/preferences.ts`

Change type to include 'list':
```typescript
type CalendarViewMode = 'auto' | '3-day' | 'week' | 'list';
```

#### 2. Update Calendar2 Component
**File:** `src/lib/components/Calendar2.svelte`

**Add view mode logic:**
```javascript
// Determine if we should show list view
$: isListView =
  $calendarViewMode === 'list' ||
  ($calendarViewMode === 'auto' && containerWidth < 600);

// For grid views, determine columns (only used when !isListView)
$: columns =
  $calendarViewMode === '3-day' ? 3
  : $calendarViewMode === 'week' ? 7
  : 7; // auto in wide mode = week
```

**Add conditional rendering:**
```svelte
{#if isListView}
  <div class="calendar-list">
    {#each currentDays as day}
      <div class="list-day-section" class:today={isToday(day)}>
        <div class="list-day-header">{formatDate(day)}</div>
        {#each getEventsForDay(day) as event}
          <div class="list-event" class:all-day={!event.start?.dateTime}>
            <!-- event details -->
          </div>
        {:else}
          <div class="list-no-events">No events</div>
        {/each}
      </div>
    {/each}
  </div>
{:else}
  <div class="calendar-grid" style:--columns={columns}>
    <!-- existing grid markup -->
  </div>
{/if}
```

**Add "List" button to toggle:**
```svelte
<div class="view-toggle">
  <button class:active={$calendarViewMode === 'auto'}>Auto</button>
  <button class:active={$calendarViewMode === 'list'}>List</button>
  <button class:active={$calendarViewMode === '3-day'}>3 Day</button>
  <button class:active={$calendarViewMode === 'week'}>Week</button>
</div>
```

#### 3. List View Styles

```css
.calendar-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.list-day-section {
  /* Contains header and events for one day */
}

.list-day-header {
  font-weight: 600;
  color: var(--teal-700);
  border-bottom: 2px solid var(--teal-200);
  padding-bottom: var(--space-xs);
  margin-bottom: var(--space-sm);
}

.list-day-section.today .list-day-header {
  color: white;
  background: var(--teal-600);
  padding: var(--space-xs) var(--space-sm);
  border-radius: 0.25rem;
  border-bottom: none;
}

.list-event {
  background: var(--teal-50);
  border-left: 3px solid var(--teal-600);
  padding: var(--space-sm) var(--space-md);
  margin-bottom: var(--space-xs);
  border-radius: 0 0.25rem 0.25rem 0;
}

.list-event.all-day {
  background: var(--teal-600);
  color: white;
  border-left-color: var(--teal-800);
}

.list-no-events {
  color: var(--teal-500);
  font-style: italic;
  padding: var(--space-sm) 0;
}
```

### Auto Mode Behavior

| Container Width | Auto Resolves To |
|-----------------|------------------|
| < 600px | List view |
| ≥ 600px | 7-day grid |

### Days Shown in List View

Show 7 days (same as week view) starting from Monday of current week. This gives users a full week overview in an easily scrollable format.
