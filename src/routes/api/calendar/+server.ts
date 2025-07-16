import { GOOGLE_CALENDAR_ID, GOOGLE_CALENDAR_PRIVATE_URL } from '$env/static/private';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async (event) => {
  console.log(
    'GOOGLE_CALENDAR_PRIVATE_URL:',
    GOOGLE_CALENDAR_PRIVATE_URL ? 'Available' : 'Not available'
  );

  // Check authentication
  if (!event.locals.user) {
    return json({ error: 'Unauthorized' }, { status: 401 });
  }

  try {
    // Try public URL first
    const publicUrl = `https://calendar.google.com/calendar/ical/${GOOGLE_CALENDAR_ID}/public/basic.ics`;
    console.log('Attempting to fetch calendar from public URL:', publicUrl);

    let response = await fetch(publicUrl);
    let dataSource = 'public';

    // If public URL fails, try to use private URL if available
    if (!response.ok) {
      if (GOOGLE_CALENDAR_PRIVATE_URL) {
        console.log('Public URL failed, trying private URL');

        response = await fetch(GOOGLE_CALENDAR_PRIVATE_URL);
        dataSource = 'private';

        if (response.ok) {
          console.log('Private URL succeeded');
        }
      }

      if (!response.ok) {
        console.log(`Calendar fetch failed with status: ${response.status}`);
        return json(
          {
            error:
              'Calendar is not accessible. Please either:\n1. Make your Google Calendar public, or\n2. Add your private ICS URL to GOOGLE_CALENDAR_PRIVATE_URL environment variable.',
            events: [],
            status: 'calendar_not_accessible',
            instructions: {
              public: 'Go to Calendar Settings → Access permissions → Make available to public',
              private:
                'Go to Calendar Settings → Integrate calendar → Copy "Secret address in iCal format"',
            },
          },
          { status: 403 }
        );
      }
    }

    const data = await response.text();
    console.log('Calendar data length:', data.length);
    console.log('Data source:', dataSource);

    const events = parseICSData(data);

    // Debug: Log basic calendar info
    console.log('Calendar Response:', {
      totalEvents: events.length,
      firstEvent: events[0],
    });

    return json({ events: events.slice(0, 50) }); // Return more events since we're limiting the range
  } catch (error) {
    console.error('Calendar API Error:', error);
    return json(
      {
        error: 'Failed to fetch calendar data. Please check your calendar configuration.',
        events: [],
        status: 'fetch_error',
      },
      { status: 500 }
    );
  }
};

function parseICSData(icsData: string) {
  const events = [];
  const lines = icsData.split('\n');
  let currentEvent: any = {};

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line.startsWith('BEGIN:VEVENT')) {
      currentEvent = {};
    } else if (line.startsWith('END:VEVENT')) {
      // Validate that the event has the required fields before adding
      if (
        currentEvent.summary &&
        currentEvent.start &&
        (currentEvent.start.dateTime || currentEvent.start.date)
      ) {
        // Ensure end date exists, if not, use start date
        if (!currentEvent.end) {
          if (currentEvent.start.date) {
            currentEvent.end = { date: currentEvent.start.date };
          } else if (currentEvent.start.dateTime) {
            currentEvent.end = { dateTime: currentEvent.start.dateTime };
          }
        }
        events.push(currentEvent);
      } else {
        console.warn('Skipping event with missing required fields:', {
          summary: currentEvent.summary,
          start: currentEvent.start,
          hasDateTime: !!currentEvent.start?.dateTime,
          hasDate: !!currentEvent.start?.date,
        });
      }
    } else if (line.startsWith('SUMMARY:')) {
      currentEvent.summary = line.substring(8);
    } else if (line.startsWith('DTSTART')) {
      try {
        // Handle different DTSTART formats:
        // DTSTART:20250716T120000Z
        // DTSTART;VALUE=DATE:20250716
        // DTSTART;TZID=America/Chicago:20250716T120000
        let dateStr = '';
        if (line.includes(':')) {
          // Get everything after the last colon
          dateStr = line.substring(line.lastIndexOf(':') + 1);
        } else {
          dateStr = line.substring(8);
        }

        const date = parseICSDate(dateStr);

        // Format date to match Google Calendar API structure
        if (line.includes('VALUE=DATE')) {
          // All-day event - only set start here, end will be set by DTEND
          currentEvent.start = { date: date.toISOString().split('T')[0] };
        } else {
          // Timed event
          currentEvent.start = { dateTime: date.toISOString() };
        }
      } catch (error) {
        console.warn('Error parsing DTSTART:', line, error);
      }
    } else if (line.startsWith('DTEND')) {
      try {
        // Handle different DTEND formats similarly
        let dateStr = '';
        if (line.includes(':')) {
          // Get everything after the last colon
          dateStr = line.substring(line.lastIndexOf(':') + 1);
        } else {
          dateStr = line.substring(6);
        }

        const date = parseICSDate(dateStr);

        if (line.includes('VALUE=DATE')) {
          currentEvent.end = { date: date.toISOString().split('T')[0] };
        } else {
          currentEvent.end = { dateTime: date.toISOString() };
        }
      } catch (error) {
        console.warn('Error parsing DTEND:', line, error);
      }
    } else if (line.startsWith('DESCRIPTION:')) {
      currentEvent.description = line.substring(12);
    } else if (line.startsWith('LOCATION:')) {
      currentEvent.location = line.substring(9);
    }
  }

  // Filter for events from last week onwards and sort by date
  const lastWeek = new Date();
  lastWeek.setDate(lastWeek.getDate() - 7);
  lastWeek.setHours(0, 0, 0, 0);

  console.log('Filtering events. Cutoff date:', lastWeek.toISOString());

  const filteredEvents = events.filter((e) => {
    const eventDate = new Date(e.start.dateTime || e.start.date);

    // For all-day events, compare just the date part to avoid timezone issues
    if (e.start.date) {
      const eventDateString = e.start.date;
      const cutoffDateString = lastWeek.toISOString().split('T')[0];
      return eventDateString >= cutoffDateString;
    } else {
      // For timed events, use the existing logic but be more lenient
      return eventDate >= lastWeek;
    }
  });

  console.log('Total events after filtering:', filteredEvents.length);

  return filteredEvents.sort((a, b) => {
    const dateA = new Date(a.start.dateTime || a.start.date);
    const dateB = new Date(b.start.dateTime || b.start.date);
    return dateA.getTime() - dateB.getTime();
  });
}

function parseICSDate(dateStr: string): Date {
  try {
    // Clean up the date string
    dateStr = dateStr.trim();

    // Handle different ICS date formats
    if (dateStr.length === 8) {
      // YYYYMMDD format
      const year = parseInt(dateStr.substr(0, 4));
      const month = parseInt(dateStr.substr(4, 2)) - 1; // Month is 0-based
      const day = parseInt(dateStr.substr(6, 2));
      const result = new Date(year, month, day);
      return result;
    } else if (dateStr.length === 16 && dateStr.endsWith('Z')) {
      // YYYYMMDDTHHMMSSZ format (UTC) - 16 characters including Z
      const year = parseInt(dateStr.substr(0, 4));
      const month = parseInt(dateStr.substr(4, 2)) - 1;
      const day = parseInt(dateStr.substr(6, 2));
      const hour = parseInt(dateStr.substr(9, 2));
      const minute = parseInt(dateStr.substr(11, 2));
      const second = parseInt(dateStr.substr(13, 2));
      const result = new Date(Date.UTC(year, month, day, hour, minute, second));
      return result;
    } else if (dateStr.length === 15 && dateStr[8] === 'T') {
      // YYYYMMDDTHHMMSS format (local time) - like 20120405T150000
      const year = parseInt(dateStr.substr(0, 4));
      const month = parseInt(dateStr.substr(4, 2)) - 1;
      const day = parseInt(dateStr.substr(6, 2));
      const hour = parseInt(dateStr.substr(9, 2));
      const minute = parseInt(dateStr.substr(11, 2));
      const second = parseInt(dateStr.substr(13, 2));
      const result = new Date(Date.UTC(year, month, day, hour, minute, second));
      return result;
    } else {
      // Try to parse as ISO date or other standard formats
      const parsed = new Date(dateStr);
      if (isNaN(parsed.getTime())) {
        // If all else fails, return current date to prevent errors
        console.warn('Could not parse date, using current date:', dateStr);
        return new Date();
      }
      return parsed;
    }
  } catch (error) {
    console.warn('Error parsing date, using current date:', dateStr, error);
    return new Date(); // Return current date as fallback
  }
}
