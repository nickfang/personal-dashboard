import Image from "next/image"

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
    const res = await fetch(`https://api.weatherapi.com/v1/current.json?key=${process.env.WEATHER_API_KEY}&q=austin&dt=2024-01-03`);

    const data = await res.json();
    return data;
  } catch (error) {
    console.log(error);
    throw new Error('Failed to fetch');
  }
}

const WeatherPage = async () => {
  const data = await getCurrentWeather();
  return (
    <div>
      <h1>Weather</h1>
      <p>{process.env.WEATHER_API_KEY}</p>
      <div>
        <h2>Location</h2>
        <p>{data.location.name}</p>
        <p>{data.location.region}</p>
        <p>{data.location.country}</p>
      </div>
      <div>
        <h2>Current</h2>
        <p>{data.current.temp_f}</p>
        <p>{data.current.condition.text}</p>
        {/* <p>{data.current.condition.icon}</p> */}
        <Image src={`https:${data.current.condition.icon}`} width="64" height="64" alt="Current Condition Icon."/>
      </div>
      {/* <pre>{JSON.stringify(data, null, 2)}</pre> */}
    </div>
  );
}

export default WeatherPage;