import { GOOGLE_CALENDAR_ID } from '$env/static/private';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
  // Add additional calendar IDs to your .env file
  const additionalCalendars = [
    'en.usa#holiday@group.v.calendar.google.com', // US Holidays
    // Add more calendar IDs here
  ];

  const calendarSources = [GOOGLE_CALENDAR_ID, ...additionalCalendars]
    .map((id) => `src=${encodeURIComponent(id)}`)
    .join('&');

  return {
    calendarUrl: `https://calendar.google.com/calendar/embed?${calendarSources}&ctz=America%2FChicago&mode=WEEK`,
  };
};
