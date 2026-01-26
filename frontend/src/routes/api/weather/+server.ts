import { WEATHER_API_KEY } from '$env/static/private';
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch }) => {
  try {
    const response = await fetch(
      `http://api.weatherapi.com/v1/forecast.json?key=${WEATHER_API_KEY}&q=Austin&days=3&aqi=yes&units=imperial`
    );

    if (!response.ok) {
      throw new Error('Failed to fetch weather data');
    }

    const data = await response.json();
    return json(data);
  } catch (error) {
    return new Response(JSON.stringify({ error: 'Failed to load weather data' }), {
      status: 500,
    });
  }
};
