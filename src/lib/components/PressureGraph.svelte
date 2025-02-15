<script lang="ts">
  import { onMount } from 'svelte';
  import Chart from 'chart.js/auto';

  export let forecast: any;

  let canvas: HTMLCanvasElement;
  let chart: Chart;

  $: if (forecast && canvas) {
    const pressureData = forecast.forecastday.flatMap((day: any) =>
      day.hour
        .filter((_: any, index: number) => index % 4 === 0)
        .map((hour: any) => ({
          time: hour.time,
          pressure: hour.pressure_in,
        }))
    );

    if (chart) {
      chart.destroy();
    }

    chart = new Chart(canvas, {
      type: 'line',
      data: {
        labels: pressureData.map((d: { time: string }) => {
          const date = new Date(d.time.replace(' ', 'T'));
          return date.toLocaleDateString('en-US', { weekday: 'short' });
        }),
        datasets: [
          {
            label: 'Pressure (mmHg)',
            data: pressureData.map((d: { pressure: number }) => d.pressure),
            borderColor: 'rgb(45, 212, 191)',
            backgroundColor: 'rgba(45, 212, 191, 0.1)',
            tension: 0.4,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: {
            beginAtZero: false,
            grid: {
              color: 'rgba(0, 0, 0, 0.05)',
            },
          },
          x: {
            grid: {
              display: false,
            },
            ticks: {
              maxRotation: 0,
              autoSkip: true,
              maxTicksLimit: 3,
            },
          },
        },
        plugins: {
          legend: {
            display: false,
          },
        },
      },
    });
  }

  onMount(() => {
    return () => {
      if (chart) {
        chart.destroy();
      }
    };
  });
</script>

<div class="graph-container">
  <canvas bind:this={canvas}></canvas>
</div>

<style>
  .graph-container {
    background: white;
    padding: 1rem;
    border-radius: 1rem;
    border: 1px solid var(--teal-100);
    height: 150px;
    margin-top: 1rem;
  }

  @media (max-width: 768px) {
    .graph-container {
      border-radius: 0.75rem;
      padding: 0.75rem;
    }
  }
</style>
