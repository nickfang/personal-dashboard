import { GOOGLE_CALENDAR_ID } from '$env/static/private';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
  try {
    const calendarUrl = `https://calendar.google.com/calendar/ical/${GOOGLE_CALENDAR_ID}/public/basic.ics`;
    const response = await fetch(calendarUrl);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch calendar: ${response.status}`);
    }

    const data = await response.text();
    const events = parseICSData(data);
    
    console.log('Calendar Response:', {
      totalEvents: events.length,
      firstEvent: events[0]
    });

    return json(events.slice(0, 10));
  } catch (error) {
    console.error('Calendar API Error:', error);
    return new Response(JSON.stringify({ error: 'Failed to fetch calendar data' }), {
      status: 500
    });
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
      if (currentEvent.summary && currentEvent.start) {
        events.push(currentEvent);
      }
    } else if (line.startsWith('SUMMARY:')) {
      currentEvent.summary = line.substring(8);
    } else if (line.startsWith('DTSTART')) {
      const dateStr = line.includes(';') ? 
        line.split(':')[1] : 
        line.substring(8);
      currentEvent.start = new Date(dateStr);
    }
  }

  return events
    .filter(e => e.start > new Date())
    .sort((a, b) => a.start.getTime() - b.start.getTime());
}  