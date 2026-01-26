import { GOOGLE_CALENDAR_ID } from '$env/static/private';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
  return {
    calendarUrl: `https://calendar.google.com/calendar/embed?src=${GOOGLE_CALENDAR_ID}&ctz=America%2FChicago`,
  };
};
