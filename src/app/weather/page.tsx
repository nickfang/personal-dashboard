import { useState } from 'react';
import Dropdown from '../../components/Dropdown';

type WeatherData = {
  location: {
    name: string
    region: string
    country: string
    lat: number
    lon: number
    tz_id: string
    localtime_epoch: number
    localtime: string
  }
  current: {
    last_updated_epoch: number
    last_updated: string
    temp_c: number
    temp_f: number
    is_day: number
    condition: {
      text: string
      icon: string
      code: number
    }
    wind_mph: number
    wind_kph: number
    wind_degree: number
    wind_dir: string
    pressure_mb: number
    pressure_in: number
    precip_mm: number
    precip_in: number
    humidity: number
    cloud: number
    feelslike_c: number
    feelslike_f: number
    vis_km: number
    vis_miles: number
    uv: number
    gust_mph: number
    gust_kph: number
  }
}

const getCurrentWeather = async():Promise<WeatherData> => {
  try {
    const res = await fetch(`https://api.weatherapi.com/v1/current.json?key=${process.env.WEATHER_API_KEY}&q=austin&dt=2023-12-02`);

    const data = await res.json();
    return data;
  } catch (error) {
    console.log(error);
    throw new Error('Failed to fetch');
  }
}

export default async function WeatherPage() {
  const [system, setSystem] = useState<'imperial' | 'metric'>('imperial'); // ['imperial', 'metric'
  const data = await getCurrentWeather();
  return (
    <div>
      <h1>Weather</h1>
      <Dropdown />
      <p>{process.env.WEATHER_API_KEY}</p>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}