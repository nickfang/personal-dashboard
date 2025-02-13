<script lang="ts">
  import { onMount } from 'svelte';
  import Chart from 'chart.js/auto';

  export let forecast: any;

  let canvas: HTMLCanvasElement;
  let chart: Chart;

  $: if (forecast && canvas) {
    const pressureData = forecast.forecastday.flatMap((day: any) => 
      day.hour.filter((_: any, index: number) => index % 4 === 0).map((hour: any) => ({
        time: new Date(hour.time).toLocaleString('en-US', { 
          weekday: 'short',
          hour: 'numeric',
          hour12: true
        }),
        pressure: hour.pressure_in
      }))
    );

    if (chart) {
      chart.destroy();
    }

    chart = new Chart(canvas, {
      type: 'line',
      data: {
        labels: pressureData.map((d: { time: string }) => d.time),
        datasets: [{
          label: 'Pressure (mmHg)',
          data: pressureData.map((d: { pressure: number }) => d.pressure),
          borderColor: 'rgb(45, 212, 191)',
          backgroundColor: 'rgba(45, 212, 191, 0.1)',
          tension: 0.3,
          fill: true
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            display: false
          }
        },
        scales: {
          y: {
            beginAtZero: false,
            grid: {
              color: 'rgba(0, 0, 0, 0.05)'
            }
          },
          x: {
            grid: {
              display: false
            },
            ticks: {
              callback: function(val, index) {
                const label = pressureData[index].time;
                return label.includes('AM') || label.includes('PM') ? label : '';
              }
            }
          }
        }
      }
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

<style>
  .graph-container {
    background: white;
    padding: 1rem;
    border-radius: 1rem;
    border: 1px solid var(--teal-100);
    height: 150px;
    margin-top: 1rem;
  }
</style>

<div class="graph-container">
  <canvas bind:this={canvas}></canvas>
</div> 