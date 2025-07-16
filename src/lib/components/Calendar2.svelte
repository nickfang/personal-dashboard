<script lang="ts">
  import { onMount } from 'svelte';
  import SectionHeader from './SectionHeader.svelte';

  let events: any[] = [];
  let cachedEvents: any[] = []; // Cache for events
  let loading = true;
  let error: string | null = null;
  let refreshError: string | null = null; // Separate error for failed refreshes
  let lastRefreshTime: Date | null = null; // Track when data was last refreshed
  let currentWeek: Date[] = [];
  let weekStartDate = new Date();
  let isFetching = false; // Prevent multiple simultaneous fetches

  function getWeekDays(startDate: Date): Date[] {
    const days: Date[] = [];
    const start = new Date(startDate);
    // Get Monday of the current week
    const dayOfWeek = start.getDay();
    const diff = start.getDate() - dayOfWeek + (dayOfWeek === 0 ? -6 : 1);
    start.setDate(diff);

    for (let i = 0; i < 7; i++) {
      const day = new Date(start);
      day.setDate(start.getDate() + i);
      days.push(day);
    }
    return days;
  }

  function formatDate(date: Date): string {
    return date.toLocaleDateString('en-US', {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
    });
  }

  function formatTime(dateString: string): string {
    const date = new Date(dateString);
    return date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    });
  }

  function formatLastRefresh(date: Date): string {
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    });
  }

  function isToday(date: Date): boolean {
    const today = new Date();
    return date.toDateString() === today.toDateString();
  }

  function getEventsForDay(date: Date): any[] {
    const dayString = date.toISOString().split('T')[0];
    return events.filter((event) => {
      if (!event.start) return false;

      // Handle all-day events (those with start.date instead of start.dateTime)
      if (event.start.date) {
        const eventStartDate = event.start.date;
        const eventEndDate = event.end?.date || event.start.date;

        // For all-day events, compare date strings directly to avoid timezone issues
        return dayString >= eventStartDate && dayString <= eventEndDate;
      } else {
        // For timed events, use the existing logic
        const eventDateString = new Date(event.start.dateTime).toISOString().split('T')[0];
        return eventDateString === dayString;
      }
    });
  }

  async function fetchCalendarEvents() {
    if (isFetching) {
      console.log('Already fetching calendar events, skipping');
      return;
    }

    try {
      isFetching = true;

      // Only show loading if we don't have cached events
      if (cachedEvents.length === 0) {
        loading = true;
      }

      error = null;
      refreshError = null;
      console.log('Starting calendar fetch...');

      // Add cache busting to ensure fresh data
      const response = await fetch(`/api/calendar?t=${Date.now()}`);

      const data = await response.json();

      if (!response.ok) {
        // If we have cached events, show refresh error instead of main error
        if (cachedEvents.length > 0) {
          refreshError = 'Failed to refresh calendar. Showing cached events.';
          events = cachedEvents;
        } else {
          if (data.status === 'calendar_not_accessible') {
            error =
              data.error || 'Calendar is not accessible. Please check your calendar configuration.';
          } else if (data.status === 'calendar_not_public') {
            error =
              'Calendar is not publicly accessible. To display your calendar events, please make your Google Calendar public in the sharing settings.';
          } else {
            error = data.error || `HTTP ${response.status}: ${response.statusText}`;
          }
          events = [];
        }
        loading = false;
        isFetching = false;
        return;
      }

      const newEvents = data.events || [];

      // Only update if we actually got events
      if (newEvents.length > 0) {
        events = newEvents;
        cachedEvents = [...newEvents]; // Update cache
        lastRefreshTime = new Date(); // Update refresh timestamp
        refreshError = null;
        console.log('Calendar events loaded:', events.length, 'events');
        console.log('First few events:', events.slice(0, 3));
      } else {
        // No events returned - keep cached events if we have them
        if (cachedEvents.length > 0) {
          refreshError = 'No events returned from calendar. Showing cached events.';
          events = cachedEvents;
        } else {
          events = [];
        }
      }

      loading = false;
      isFetching = false;
    } catch (e) {
      console.error('Calendar fetch error:', e);
      console.log('Fetch failed, using cached events if available');

      // If we have cached events, use them and show refresh error
      if (cachedEvents.length > 0) {
        events = cachedEvents;
        refreshError = 'Failed to connect to calendar service. Showing cached events.';
        error = null;
      } else {
        error = 'Unable to connect to calendar service. Please check your internet connection.';
        events = [];
      }

      loading = false;
      isFetching = false;
    }
  }

  function refreshCalendar() {
    fetchCalendarEvents();
  }

  function previousWeek() {
    weekStartDate.setDate(weekStartDate.getDate() - 7);
    weekStartDate = new Date(weekStartDate);
    currentWeek = getWeekDays(weekStartDate);
  }

  function nextWeek() {
    weekStartDate.setDate(weekStartDate.getDate() + 7);
    weekStartDate = new Date(weekStartDate);
    currentWeek = getWeekDays(weekStartDate);
  }

  function goToToday() {
    weekStartDate = new Date();
    currentWeek = getWeekDays(weekStartDate);
  }

  onMount(() => {
    currentWeek = getWeekDays(weekStartDate);
    // Fetch calendar events immediately on mount
    fetchCalendarEvents();
  });
</script>

<div class="calendar2-container">
  <SectionHeader
    title="Calendar"
    fullscreenPath="/fullscreen/calendar"
    onRefresh={refreshCalendar}
  />

  {#if loading}
    <div class="calendar-loading">
      <div class="loading-spinner"></div>
      <p>Loading calendar events...</p>
    </div>
  {:else if error}
    <div class="calendar-error">
      <div class="error-icon">📅</div>
      <p class="error-message">{error}</p>
      {#if error.includes('publicly accessible')}
        <div class="help-text">
          <p>To make your Google Calendar public:</p>
          <ol>
            <li>Open Google Calendar</li>
            <li>Go to Settings → Settings for my calendars</li>
            <li>Select your calendar</li>
            <li>Under "Access permissions," check "Make available to public"</li>
          </ol>
        </div>
      {/if}
      <button class="retry-button" on:click={refreshCalendar}> Try Again </button>
    </div>
  {:else}
    <!-- Show refresh error if present -->
    {#if refreshError}
      <div class="refresh-error">
        <div class="refresh-error-content">
          <span class="refresh-error-icon">⚠️</span>
          <span class="refresh-error-message">{refreshError}</span>
          <button class="refresh-error-button" on:click={refreshCalendar}> Refresh Now </button>
        </div>
      </div>
    {/if}

    <div class="calendar-content">
      <!-- Week Navigation -->
      <div class="week-nav">
        <button
          class="nav-button"
          on:click={previousWeek}
          title="Previous Week"
          aria-label="Previous Week"
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="15,18 9,12 15,6"></polyline>
          </svg>
        </button>

        <button class="today-button" on:click={goToToday}> Today </button>

        <button class="nav-button" on:click={nextWeek} title="Next Week" aria-label="Next Week">
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="9,18 15,12 9,6"></polyline>
          </svg>
        </button>
      </div>

      <!-- Calendar Grid -->
      <div class="calendar-grid">
        {#each currentWeek as day}
          <div class="day-column" class:today={isToday(day)}>
            <div class="day-header">
              <div class="day-name">{formatDate(day)}</div>
            </div>

            <div class="day-events">
              {#each getEventsForDay(day) as event}
                <div class="event" class:all-day={!event.start?.dateTime}>
                  <div class="event-title">
                    {@html (event.summary || 'Untitled Event')
                      .replace(/\\n/g, '<br>')
                      .replace(/\n/g, '<br>')}
                  </div>
                  {#if event.location}
                    <div class="event-location">
                      📍 {@html event.location
                        .replace(/\\n/g, '<br>')
                        .replace(/\n/g, '<br>')
                        .replace(/\\,/g, ',')}
                    </div>
                  {/if}
                  {#if event.start?.dateTime}
                    <div class="event-time">
                      {formatTime(event.start.dateTime)}
                      {#if event.end?.dateTime && event.start.dateTime !== event.end.dateTime}
                        - {formatTime(event.end.dateTime)}
                      {/if}
                    </div>
                  {:else}
                    <div class="event-time">All day</div>
                  {/if}
                </div>
              {:else}
                <div class="no-events">No events</div>
              {/each}
            </div>
          </div>
        {/each}
      </div>

      <!-- Last refresh info -->
      {#if lastRefreshTime}
        <div class="last-refresh">
          <span class="last-refresh-text">
            Last updated: {formatLastRefresh(lastRefreshTime)}
          </span>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .calendar2-container {
    padding: 1.5rem;
    height: 100%;
    display: flex;
    flex-direction: column;
    font-size: clamp(0.875rem, 1.2vw, 1.1rem);
  }

  .calendar-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 300px;
    color: var(--teal-600, #006666);
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--teal-100, #a3dfdf);
    border-top: 3px solid var(--teal-600, #006666);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
  }

  .refresh-error {
    background: var(--teal-50, #f0fffe);
    border: 1px solid var(--teal-200, #a3dfdf);
    border-radius: 8px;
    padding: 0.75rem 1rem;
    margin-bottom: 1rem;
  }

  .refresh-error-content {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .refresh-error-icon {
    font-size: 1.25rem;
  }

  .refresh-error-message {
    flex: 1;
    min-width: 200px;
    color: var(--teal-700, #004d4d);
    font-size: 0.9rem;
  }

  .refresh-error-button {
    background: var(--teal-600, #006666);
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    font-size: 0.85rem;
    cursor: pointer;
    transition: background-color 0.2s;
  }

  .refresh-error-button:hover {
    background: var(--teal-700, #004d4d);
  }

  .last-refresh {
    margin-top: 1rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--teal-100, #a3dfdf);
    text-align: center;
  }

  .last-refresh-text {
    color: var(--teal-600, #006666);
    font-size: 0.8rem;
    font-style: italic;
  }

  .calendar-error {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 300px;
    text-align: center;
    color: var(--teal-600, #006666);
  }

  .error-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .error-message {
    margin-bottom: 1rem;
    color: var(--gray-800, #1f2937);
  }

  .help-text {
    background: var(--teal-50, #d1efef);
    border: 1px solid var(--teal-100, #a3dfdf);
    border-radius: 0.375rem;
    padding: 1rem;
    margin-bottom: 1rem;
    text-align: left;
    max-width: 400px;
  }

  .help-text p {
    margin: 0 0 0.5rem 0;
    font-weight: 500;
    color: var(--teal-800, #004444);
  }

  .help-text ol {
    margin: 0;
    padding-left: 1.25rem;
    color: var(--gray-700, #374151);
  }

  .help-text li {
    margin-bottom: 0.25rem;
    font-size: 0.875rem;
  }

  .retry-button {
    background: var(--teal-600, #006666);
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 0.375rem;
    cursor: pointer;
    font-size: 0.875rem;
    transition: background-color 0.2s ease;
  }

  .retry-button:hover {
    background: var(--teal-800, #004444);
  }

  .calendar-content {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .week-nav {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    margin-bottom: 1rem;
    padding: 0.5rem;
  }

  .nav-button {
    background: white;
    border: 1px solid var(--teal-100, #a3dfdf);
    border-radius: 0.375rem;
    padding: 0.5rem;
    cursor: pointer;
    color: var(--teal-600, #006666);
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .nav-button:hover {
    background: var(--teal-50, #d1efef);
    border-color: var(--teal-600, #006666);
  }

  .today-button {
    background: var(--teal-600, #006666);
    color: white;
    border: none;
    border-radius: 0.375rem;
    padding: 0.5rem 1rem;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    transition: background-color 0.2s ease;
  }

  .today-button:hover {
    background: var(--teal-800, #004444);
  }

  .calendar-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 0.5rem;
    flex: 1;
    min-height: 0;
  }

  .day-column {
    background: white;
    border-radius: 0.375rem;
    border: 1px solid var(--teal-100, #a3dfdf);
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .day-column.today {
    border-color: var(--teal-600, #006666);
    border-width: 2px;
  }

  .day-header {
    background: var(--teal-50, #d1efef);
    padding: 0.5rem;
    text-align: center;
    border-bottom: 1px solid var(--teal-100, #a3dfdf);
  }

  .today .day-header {
    background: var(--teal-600, #006666);
    color: white;
  }

  .day-name {
    font-size: clamp(0.8rem, 1vw, 1rem);
    font-weight: 500;
  }

  .day-events {
    flex: 1;
    padding: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    overflow-y: auto;
  }

  .event {
    background: var(--teal-100, #a3dfdf);
    border-radius: 0.25rem;
    padding: 0.375rem 0.5rem;
    font-size: clamp(0.75rem, 0.9vw, 0.875rem);
    border-left: 3px solid var(--teal-600, #006666);
  }

  .event.all-day {
    background: var(--teal-600, #006666);
    color: white;
  }

  .event-title {
    font-weight: 500;
    margin-bottom: 0.125rem;
    word-break: break-word;
    line-height: 1.3;
  }

  .event-location {
    font-size: clamp(0.65rem, 0.8vw, 0.75rem);
    color: var(--teal-600, #006666);
    opacity: 0.8;
    margin-bottom: 0.125rem;
    word-break: break-word;
    line-height: 1.2;
  }

  .event-time {
    font-size: clamp(0.7rem, 0.85vw, 0.8rem);
    opacity: 0.8;
  }

  .no-events {
    text-align: center;
    color: var(--teal-600, #006666);
    opacity: 0.6;
    font-size: clamp(0.75rem, 0.9vw, 0.875rem);
    padding: 1rem 0;
  }

  @keyframes spin {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
  }

  /* Responsive design */
  @media (max-width: 768px) {
    .calendar2-container {
      padding: 1rem;
      font-size: 0.9rem;
    }

    .calendar-grid {
      gap: 0.25rem;
    }

    .day-events {
      padding: 0.25rem;
    }

    .event {
      padding: 0.3rem 0.4rem;
      font-size: 0.8rem;
    }

    .event-location {
      font-size: 0.7rem;
    }

    .event-time {
      font-size: 0.75rem;
    }

    .day-name {
      font-size: 0.85rem;
    }
  }

  @media (max-width: 1024px) {
    .calendar2-container {
      font-size: 0.85rem;
    }

    .calendar-grid {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }

    .day-column {
      min-height: auto;
      flex-direction: row;
      align-items: flex-start;
      border-radius: 0.5rem;
      overflow: hidden;
    }

    .day-header {
      flex-shrink: 0;
      width: 100px;
      padding: 0.75rem 0.5rem;
      border-bottom: none;
      border-right: 1px solid var(--teal-100, #a3dfdf);
      text-align: left;
      display: flex;
      align-items: center;
      overflow: hidden;
    }

    .today .day-header {
      background: var(--teal-600, #006666);
      color: white;
    }

    .day-name {
      font-size: 0.85rem;
      font-weight: 600;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .day-events {
      flex: 1;
      padding: 0.5rem;
      flex-direction: column;
      gap: 0.25rem;
      align-items: stretch;
      overflow: hidden;
      min-width: 0;
    }

    .event {
      font-size: 0.8rem;
      padding: 0.4rem 0.6rem;
      margin: 0;
      border-radius: 0.375rem;
      border-left: 3px solid var(--teal-600, #006666);
      background: var(--teal-100, #a3dfdf);
      max-width: 100%;
      box-sizing: border-box;
    }

    .event.all-day {
      background: var(--teal-600, #006666);
      color: white;
      border-left-color: var(--teal-800, #004444);
    }

    .event-title {
      font-size: 0.8rem;
      margin-bottom: 0.125rem;
      line-height: 1.3;
      word-wrap: break-word;
      overflow-wrap: break-word;
    }

    .event-location {
      font-size: 0.7rem;
      margin-bottom: 0.125rem;
      margin-top: 0;
      word-wrap: break-word;
      overflow-wrap: break-word;
    }

    .event-time {
      font-size: 0.7rem;
      margin-top: 0;
      word-wrap: break-word;
      overflow-wrap: break-word;
    }

    .no-events {
      color: var(--teal-500, #006666);
      opacity: 0.6;
      font-size: 0.75rem;
      padding: 0.75rem 0;
      font-style: italic;
    }
  }

  /* iPhone and small mobile sizes - vertical stacking */
  @media (max-width: 600px) {
    .day-column {
      flex-direction: column;
      align-items: stretch;
      overflow: hidden;
    }

    .day-header {
      width: 100%;
      border-right: none;
      border-bottom: 1px solid var(--teal-100, #a3dfdf);
      text-align: center;
      padding: 0.6rem 0.5rem;
      overflow: hidden;
    }

    .day-name {
      font-size: 0.9rem;
      font-weight: 600;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .day-events {
      padding: 0.6rem;
      overflow: hidden;
      min-height: 0;
    }

    .event {
      font-size: 0.85rem;
      padding: 0.5rem 0.7rem;
      max-width: 100%;
      box-sizing: border-box;
    }

    .event-title {
      font-size: 0.85rem;
      word-wrap: break-word;
      overflow-wrap: break-word;
      hyphens: auto;
    }

    .event-location {
      font-size: 0.75rem;
      word-wrap: break-word;
      overflow-wrap: break-word;
      hyphens: auto;
    }

    .event-time {
      font-size: 0.75rem;
      word-wrap: break-word;
      overflow-wrap: break-word;
    }
  }

  /* Large displays */
  @media (min-width: 1920px) {
    .calendar2-container {
      padding: 2rem;
    }

    .week-nav {
      margin-bottom: 1.5rem;
    }

    .calendar-grid {
      gap: 1rem;
    }
  }
</style>
